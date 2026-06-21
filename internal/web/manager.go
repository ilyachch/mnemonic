package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/mcp"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProjectInstance owns the cached MCP server state for the served project.
type ProjectInstance struct {
	MCPServer *sdkmcp.Server
	IndexDB   *sql.DB

	handler *projectSessionHandler
}

// Close releases project resources.
func (p *ProjectInstance) Close() error {
	if p == nil || p.IndexDB == nil {
		return nil
	}
	return p.IndexDB.Close()
}

// ServerManager serves one resolved project over HTTP/SSE.
type ServerManager struct {
	projectResolution app.ProjectResolution
	memoriesHome      string
	projectSlug       string

	instanceFactory func() (*ProjectInstance, error)
	instance        *ProjectInstance
	mu              sync.Mutex
}

// NewServerManager prepares a manager for a single resolved project.
func NewServerManager(resolution app.ProjectResolution, memoriesHome string) (*ServerManager, error) {
	if strings.TrimSpace(memoriesHome) == "" {
		return nil, app.NewCLIUsageError("memories home is required", nil)
	}
	if strings.TrimSpace(resolution.Project.Slug) == "" {
		return nil, app.NewCLIUsageError("project resolution is required", nil)
	}

	manager := &ServerManager{
		projectResolution: resolution,
		memoriesHome:      memoriesHome,
		projectSlug:       resolution.Project.Slug,
	}
	manager.instanceFactory = manager.buildProjectInstance
	return manager, nil
}

// Close releases the cached project instance.
func (m *ServerManager) Close() error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance == nil {
		return nil
	}

	err := m.instance.Close()
	m.instance = nil
	return err
}

// ServeHTTP routes MCP traffic for the configured project.
func (m *ServerManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug, endpoint, ok := parseProjectEndpoint(r.URL.Path)
	if !ok || slug != m.projectSlug {
		http.NotFound(w, r)
		return
	}

	instance, err := m.instanceForSlug(slug)
	if err != nil {
		writeHTTPError(w, err)
		return
	}

	switch endpoint {
	case "sse":
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		instance.handler.ServeGET(w, r)
	case "messages":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		instance.handler.ServePOST(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *ServerManager) instanceForSlug(slug string) (*ProjectInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance != nil {
		return m.instance, nil
	}

	factory := m.instanceFactory
	if factory == nil {
		factory = m.buildProjectInstance
	}
	instance, err := factory()
	if err != nil {
		return nil, err
	}
	m.instance = instance
	return instance, nil
}

func (m *ServerManager) buildProjectInstance() (*ProjectInstance, error) {
	indexPath, err := index.Path(m.projectResolution.Project.ID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return nil, app.NewNotFoundError(fmt.Sprintf("index for %q is missing; run `mnemonic project reindex`", m.projectSlug), nil)
		}
		return nil, fmt.Errorf("stat index %q: %w", indexPath, err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply busy timeout: %w", err)
	}

	srv := mcp.NewServerWithIndexDB(m.projectResolution, paths.EffectivePaths{MemoriesHome: m.memoriesHome}, db, false)
	sdkServer := srv.BuildSDKServer()

	return &ProjectInstance{
		MCPServer: sdkServer,
		IndexDB:   db,
		handler:   newProjectSessionHandler(sdkServer),
	}, nil
}

func parseProjectEndpoint(path string) (slug, endpoint string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 || parts[0] != "mcp" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func writeHTTPError(w http.ResponseWriter, err error) {
	switch appErrorCode(err) {
	case app.CodeNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case app.CodeCLIUsage:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func appErrorCode(err error) app.ErrCode {
	var appErr *app.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return app.CodeInternal
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
	messagesURL.Path = strings.TrimRight(messagesURL.Path, "/")
	if messagesURL.Path == "" {
		messagesURL.Path = "/messages"
	} else {
		messagesURL.Path = strings.TrimSuffix(messagesURL.Path, "/sse") + "/messages"
	}
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
