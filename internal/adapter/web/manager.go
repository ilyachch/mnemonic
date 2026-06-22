package web

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ilyachch/mnemonic/internal/adapter/stdio"
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	sseEndpoint      = "/sse"
	messagesEndpoint = "/messages"
)

// ServerInput configures the HTTP/SSE adapter for one resolved knowledge base.
type ServerInput struct {
	KB           kb.KnowledgeBase
	Services     app.RuntimeServices
	ProjectToken string
	ReadOnly     bool
}

// Server serves one resolved project over HTTP/SSE.
type Server struct {
	sdkServer    *sdkmcp.Server
	projectToken string
	readOnly     bool

	sessions *projectSessionHandler
}

// NewServer prepares an eager MCP HTTP server for a single resolved project.
func NewServer(input ServerInput) (*Server, error) {
	if strings.TrimSpace(input.KB.ID) == "" {
		return nil, apperr.CLIUsage("knowledge base is required", nil)
	}
	if input.Services.Notes == nil {
		return nil, apperr.CLIUsage("notes service is required", nil)
	}
	if input.Services.Search == nil {
		return nil, apperr.CLIUsage("search service is required", nil)
	}
	if input.Services.Index == nil {
		return nil, apperr.CLIUsage("index service is required", nil)
	}

	stdioServer, err := stdio.NewServer(input.KB, stdio.Dependencies{
		Notes:  input.Services.Notes,
		Search: input.Services.Search,
		Index:  input.Services.Index,
	}, input.ReadOnly)
	if err != nil {
		return nil, err
	}
	sdkServer := stdioServer.BuildSDKServer()

	return &Server{
		sdkServer:    sdkServer,
		projectToken: strings.TrimSpace(input.ProjectToken),
		readOnly:     input.ReadOnly,
		sessions:     newProjectSessionHandler(sdkServer),
	}, nil
}

// Close releases server resources.
func (s *Server) Close() error {
	return nil
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
