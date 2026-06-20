package web

import (
	"context"
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
	"github.com/ilyachch/mnemonic/internal/webauth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProjectInstance owns the cached MCP server state for a single project.
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

type projectMetadata struct {
	Resolution app.ProjectResolution
}

type statusError struct {
	status int
	msg    string
	err    error
}

func (e *statusError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

func (e *statusError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *statusError) StatusCode() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	return e.status
}

func newStatusError(status int, msg string, err error) error {
	return &statusError{status: status, msg: msg, err: err}
}

// ServerManager validates access and lazily caches per-project MCP instances.
type ServerManager struct {
	authStore      *webauth.Store
	superuserToken string
	serveSlugs     map[string]struct{}
	projects       map[string]projectMetadata
	instances      map[string]*ProjectInstance
	memoriesHome   string

	instanceFactory func(slug string) (*ProjectInstance, error)
	mu              sync.RWMutex
}

// NewServerManager validates the requested slugs and prepares a manager.
func NewServerManager(authStore *webauth.Store, superuserToken, memoriesHome string, slugs []string) (*ServerManager, error) {
	projects, err := loadServeProjects(memoriesHome, slugs)
	if err != nil {
		return nil, err
	}

	manager := &ServerManager{
		authStore:      authStore,
		superuserToken: strings.TrimSpace(superuserToken),
		serveSlugs:     make(map[string]struct{}, len(projects)),
		projects:       projects,
		instances:      make(map[string]*ProjectInstance),
		memoriesHome:   memoriesHome,
	}
	for slug := range projects {
		manager.serveSlugs[slug] = struct{}{}
	}
	manager.instanceFactory = manager.buildProjectInstance
	return manager, nil
}

// Close releases all cached project instances and the auth store.
func (m *ServerManager) Close() error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []string
	for slug, instance := range m.instances {
		if instance == nil {
			continue
		}
		if err := instance.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
		}
	}
	m.instances = make(map[string]*ProjectInstance)
	if m.authStore != nil {
		if err := m.authStore.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close web manager: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ServeHTTP routes authenticated MCP traffic for the configured projects.
func (m *ServerManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug, endpoint, ok := parseProjectEndpoint(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, allowed := m.serveSlugs[slug]; !allowed {
		http.NotFound(w, r)
		return
	}

	if _, err := m.authorize(r.Context(), r, slug); err != nil {
		writeAuthError(w, err)
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

func (m *ServerManager) authorize(ctx context.Context, r *http.Request, slug string) (*webauth.User, error) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return nil, newStatusError(http.StatusUnauthorized, "authorization bearer token is required", nil)
	}

	if m.superuserToken != "" && token == m.superuserToken {
		return &webauth.User{Username: "superuser"}, nil
	}

	if m.authStore == nil {
		return nil, app.NewInternalError("web auth store is not configured", nil)
	}

	user, err := m.authStore.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	perms, err := m.authStore.GetPermissions(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	for _, perm := range perms {
		if perm.ProjectSelector == slug {
			return user, nil
		}
	}

	return nil, newStatusError(http.StatusForbidden, "forbidden", nil)
}

func (m *ServerManager) instanceForSlug(slug string) (*ProjectInstance, error) {
	m.mu.RLock()
	instance := m.instances[slug]
	m.mu.RUnlock()
	if instance != nil {
		return instance, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if instance = m.instances[slug]; instance != nil {
		return instance, nil
	}

	factory := m.instanceFactory
	if factory == nil {
		factory = m.buildProjectInstance
	}
	instance, err := factory(slug)
	if err != nil {
		return nil, err
	}
	m.instances[slug] = instance
	return instance, nil
}

func (m *ServerManager) buildProjectInstance(slug string) (*ProjectInstance, error) {
	meta, ok := m.projects[slug]
	if !ok {
		return nil, app.NewNotFoundError(fmt.Sprintf("project %q is not configured for web serving", slug), nil)
	}

	indexPath, err := index.Path(meta.Resolution.Project.ID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return nil, app.NewNotFoundError(fmt.Sprintf("index for %q is missing; run `mnemonic project reindex`", slug), nil)
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

	srv := mcp.NewServerWithIndexDB(meta.Resolution, paths.EffectivePaths{MemoriesHome: m.memoriesHome}, db)
	sdkServer := srv.BuildSDKServer()

	return &ProjectInstance{
		MCPServer: sdkServer,
		IndexDB:   db,
		handler:   newProjectSessionHandler(sdkServer),
	}, nil
}

func loadServeProjects(memoriesHome string, slugs []string) (map[string]projectMetadata, error) {
	if strings.TrimSpace(memoriesHome) == "" {
		return nil, app.NewCLIUsageError("memories home is required", nil)
	}

	projects := make(map[string]projectMetadata, len(slugs))
	for _, rawSlug := range slugs {
		slug := strings.TrimSpace(rawSlug)
		if slug == "" {
			continue
		}
		if _, exists := projects[slug]; exists {
			continue
		}

		meta, err := loadProjectMetadata(memoriesHome, slug)
		if err != nil {
			return nil, err
		}
		projects[slug] = meta
	}
	if len(projects) == 0 {
		return nil, app.NewCLIUsageError("MNEMONIC_SERVE_PROJECTS did not contain any project slugs", nil)
	}
	return projects, nil
}

func loadProjectMetadata(memoriesHome, slug string) (projectMetadata, error) {
	centralManifest := filepath.Join(memoriesHome, slug, "mnemonic.toml")
	centralExists := false
	if info, err := os.Stat(centralManifest); err == nil && !info.IsDir() {
		centralExists = true
	} else if err != nil && !os.IsNotExist(err) {
		return projectMetadata{}, fmt.Errorf("stat manifest %q: %w", centralManifest, err)
	}

	pointerPath := filepath.Join(memoriesHome, slug+".toml")
	pointerExists := false
	if info, err := os.Stat(pointerPath); err == nil && !info.IsDir() {
		pointerExists = true
	} else if err != nil && !os.IsNotExist(err) {
		return projectMetadata{}, fmt.Errorf("stat pointer %q: %w", pointerPath, err)
	}

	switch {
	case centralExists && pointerExists:
		return projectMetadata{}, app.NewAmbiguousError(fmt.Sprintf("project %q exists as both a central directory and a pointer file", slug), nil)
	case centralExists:
		return loadCentralProjectMetadata(slug, centralManifest)
	case pointerExists:
		return loadLocalProjectMetadata(slug, pointerPath)
	default:
		return projectMetadata{}, app.NewNotFoundError(fmt.Sprintf("project %q not found under %s", slug, memoriesHome), nil)
	}
}

func loadCentralProjectMetadata(slug, manifestPath string) (projectMetadata, error) {
	manifest, err := project.ParseMnemonicManifestFromFile(manifestPath)
	if err != nil {
		return projectMetadata{}, err
	}
	if manifest.Slug != slug {
		return projectMetadata{}, app.NewNotFoundError(fmt.Sprintf("project %q manifest slug mismatch: %q", slug, manifest.Slug), nil)
	}

	memoriesAbs := filepath.Dir(manifestPath)
	return projectMetadata{
		Resolution: app.ProjectResolution{
			MnemonicFilePath: manifestPath,
			RepoRootAbs:      memoriesAbs,
			MemoriesAbs:      memoriesAbs,
			ManifestAbs:      manifestPath,
			Project: app.ProjectRecord{
				ID:           manifest.ProjectID,
				Name:         manifest.Name,
				Slug:         manifest.Slug,
				Kind:         "central",
				MemoriesPath: filepath.Base(memoriesAbs),
			},
		},
	}, nil
}

func loadLocalProjectMetadata(slug, pointerPath string) (projectMetadata, error) {
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		return projectMetadata{}, fmt.Errorf("read pointer %q: %w", pointerPath, err)
	}
	pointer, err := project.ParsePointerFile(data)
	if err != nil {
		return projectMetadata{}, err
	}

	manifestPath := pointer.ManifestPath
	manifest, err := project.ParseMnemonicManifestFromFile(manifestPath)
	if err != nil {
		return projectMetadata{}, err
	}
	if manifest.Slug != slug {
		return projectMetadata{}, app.NewNotFoundError(fmt.Sprintf("project %q manifest slug mismatch: %q", slug, manifest.Slug), nil)
	}

	memoriesAbs := filepath.Dir(manifestPath)
	repoRootAbs := filepath.Dir(filepath.Dir(manifestPath))
	return projectMetadata{
		Resolution: app.ProjectResolution{
			MnemonicFilePath: manifestPath,
			RepoRootAbs:      repoRootAbs,
			MemoriesAbs:      memoriesAbs,
			ManifestAbs:      manifestPath,
			Project: app.ProjectRecord{
				ID:           manifest.ProjectID,
				Name:         manifest.Name,
				Slug:         manifest.Slug,
				Kind:         "local",
				MemoriesPath: filepath.Base(memoriesAbs),
			},
		},
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

func bearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusForbidden
	var statusErr *statusError
	switch {
	case errors.As(err, &statusErr):
		status = statusErr.StatusCode()
	case appErrorCode(err) == app.CodeNotFound:
		status = http.StatusUnauthorized
	case appErrorCode(err) == app.CodeCLIUsage:
		status = http.StatusUnauthorized
	}
	http.Error(w, err.Error(), status)
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

// ParseServeProjects returns the comma-separated project slugs in the provided value.
func ParseServeProjects(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	slugs := make([]string, 0, len(parts))
	for _, part := range parts {
		slug := strings.TrimSpace(part)
		if slug != "" {
			slugs = append(slugs, slug)
		}
	}
	return slugs
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
