package web

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/mcp"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	sseEndpoint      = "/sse"
	messagesEndpoint = "/messages"
)

// Server serves one resolved project over HTTP/SSE.
type Server struct {
	projectResolution app.ProjectResolution
	effectivePaths    paths.EffectivePaths
	indexDB           *sql.DB
	sdkServer         *sdkmcp.Server
	projectToken      string
	readOnly          bool

	sessions *projectSessionHandler
}

// NewServer prepares an eager MCP HTTP server for a single resolved project.
func NewServer(resolution app.ProjectResolution, effectivePaths paths.EffectivePaths, indexDB *sql.DB, projectToken string, readOnly bool) (*Server, error) {
	if strings.TrimSpace(resolution.Project.Slug) == "" {
		return nil, apperr.CLIUsage("project resolution is required", nil)
	}
	if strings.TrimSpace(resolution.Project.ID) == "" {
		return nil, apperr.CLIUsage("project id is required", nil)
	}
	if indexDB == nil {
		return nil, apperr.Internal("index database is required", nil)
	}

	mcpServer := mcp.NewServerWithIndexDB(resolution, effectivePaths, indexDB, readOnly)
	sdkServer := mcpServer.BuildSDKServer()

	return &Server{
		projectResolution: resolution,
		effectivePaths:    effectivePaths,
		indexDB:           indexDB,
		sdkServer:         sdkServer,
		projectToken:      strings.TrimSpace(projectToken),
		readOnly:          readOnly,
		sessions:          newProjectSessionHandler(sdkServer),
	}, nil
}

// Close releases the open index DB.
func (s *Server) Close() error {
	if s == nil || s.indexDB == nil {
		return nil
	}
	return s.indexDB.Close()
}

// ServeHTTP routes only /sse and /messages for the configured project.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch {
	case r.URL.Path == sseEndpoint:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.sessions.ServeGET(w, r)
	case r.URL.Path == messagesEndpoint:
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.sessions.ServePOST(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) authorize(r *http.Request) bool {
	if s == nil || s.projectToken == "" {
		return true
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, token, ok := strings.Cut(auth, " ")
	if !ok || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return false
	}
	return strings.TrimSpace(token) == s.projectToken
}

type projectSessionHandler struct {
	server   *sdkmcp.Server
	mu       sync.Mutex
	sessions map[string]*sdkmcp.SSEServerTransport
}

func newProjectSessionHandler(server *sdkmcp.Server) *projectSessionHandler {
	return &projectSessionHandler{
		server:   server,
		sessions: make(map[string]*sdkmcp.SSEServerTransport),
	}
}

func (h *projectSessionHandler) ServeGET(w http.ResponseWriter, r *http.Request) {
	if h.server == nil {
		http.Error(w, "no MCP server available", http.StatusInternalServerError)
		return
	}

	sessionID := project.NewUUID()
	messagesURL := *r.URL
	messagesURL.Path = messagesEndpoint
	q := messagesURL.Query()
	q.Set("sessionid", sessionID)
	messagesURL.RawQuery = q.Encode()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	transport := &sdkmcp.SSEServerTransport{
		Endpoint: messagesURL.RequestURI(),
		Response: w,
	}

	h.mu.Lock()
	h.sessions[sessionID] = transport
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
	}()

	session, err := h.server.Connect(r.Context(), transport, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer session.Close()
	_ = session.Wait()
}

func (h *projectSessionHandler) ServePOST(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionid")
	if sessionID == "" {
		http.Error(w, "sessionid must be provided", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	transport := h.sessions[sessionID]
	h.mu.Unlock()
	if transport == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	transport.ServeHTTP(w, r)
}

// ServeAddr resolves the listener address from flag/env/default precedence.
func ServeAddr(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	if envValue := strings.TrimSpace(os.Getenv("MNEMONIC_WEB_ADDR")); envValue != "" {
		return envValue
	}
	return ":8080"
}
