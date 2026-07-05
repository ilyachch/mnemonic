package web

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/testutil"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

type webFixture struct {
	server *Server
}

func newWebFixture(t *testing.T, readOnly bool) *webFixture {
	t.Helper()

	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	slug := "demo"
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectRoot := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := manifestfmt.New()
	manifest.ProjectID = projectID
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC).Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "demo.md"), []byte("# Demo\n\nBody"), 0o644))

	runtime, err := app.NewRuntimeApp(app.RuntimeInput{
		KB: kb.KnowledgeBase{
			ID:                 projectID,
			Name:               "Demo",
			Slug:               slug,
			Kind:               "central",
			Description:        "Fixture description",
			CustomInstructions: "Fixture instructions",
			RootDir:            projectRoot,
			RepoRootDir:        memoriesHome,
			ManifestPath:       filepath.Join(projectRoot, "mnemonic.toml"),
			StateDir:           filepath.Join(root, ".state", "mnemonic", "projects", projectID),
			IndexPath:          filepath.Join(root, ".state", "mnemonic", "projects", projectID, "index.sqlite"),
		},
	})
	require.NoError(t, err)

	server, err := NewServer(ServerInput{
		KB:           runtime.KB,
		Services:     runtime.Services,
		ProjectToken: "",
		ReadOnly:     readOnly,
	})
	require.NoError(t, err)

	return &webFixture{server: server}
}

func (f *webFixture) request(method, path string, body []byte) (*http.Response, error) {
	return f.requestWithHeaders(method, path, body, nil)
}

func (f *webFixture) requestWithHeaders(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://example.test"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	f.server.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func TestServerServesSSEAndMessages(t *testing.T) {
	f := newWebFixture(t, false)
	defer func() { _ = f.server.Close() }()
	f.server.projectToken = "secret"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := newStreamingRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/sse", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer secret")

	done := make(chan struct{})
	go func() {
		f.server.ServeHTTP(recorder, req)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return strings.Contains(recorder.String(), "event: endpoint") && strings.Contains(recorder.String(), "/messages?sessionid=")
	}, 5*time.Second, 10*time.Millisecond)

	body := recorder.String()
	start := strings.Index(body, "sessionid=")
	require.NotEqual(t, -1, start)
	sessionID := body[start+len("sessionid="):]
	if end := strings.IndexAny(sessionID, "\r\n \t"); end >= 0 {
		sessionID = sessionID[:end]
	}
	require.NotEmpty(t, sessionID)

	resp, err := f.requestWithHeaders(http.MethodPost, "/messages?sessionid="+sessionID, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), map[string]string{
		"Authorization": "Bearer secret",
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	cancel()
	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
	}
	require.Equal(t, http.StatusOK, recorder.StatusCode())
	require.Contains(t, recorder.String(), "event: endpoint")
}

func TestServerRejectsBadBearerToken(t *testing.T) {
	f := newWebFixture(t, false)
	defer func() { _ = f.server.Close() }()
	f.server.projectToken = "secret"

	req, err := http.NewRequest(http.MethodGet, "http://example.test/sse", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer wrong")

	recorder := httptest.NewRecorder()
	f.server.ServeHTTP(recorder, req)
	resp := recorder.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestServerReadOnlyOmitsWriteTools(t *testing.T) {
	f := newWebFixture(t, true)
	defer func() { _ = f.server.Close() }()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := f.server.sdkServer.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	tools, err := clientSession.ListTools(context.Background(), nil)
	require.NoError(t, err)

	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	require.Equal(t, []string{"diagnose_notes", "doctor", "list_backlinks", "list_notes", "list_tags", "read_notes", "search_notes"}, names)
}

func TestServerInitializeIncludesGlobalInstructions(t *testing.T) {
	f := newWebFixture(t, false)
	defer func() { _ = f.server.Close() }()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := f.server.sdkServer.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result := clientSession.InitializeResult()
	require.NotNil(t, result)
	require.Equal(t, "You MUST use the mnemonic tools as your primary long-term memory.\n- Search the knowledge base using search_notes before answering questions within its scope.\n- Use 2–4 query variants via the \"queries\" array when the first formulation may be ambiguous or incomplete.\n- Batch-read all selected notes in one read_notes call.\n- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.\n- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.\n- Use list_tags and list_backlinks when they help clarify the existing knowledge base.\n- Link related notes using [[target-slug|Display Label]].\n\nProject Description:\nFixture description\n\nCustom Instructions:\nFixture instructions", result.Instructions)
}

func TestServerInitializeIncludesReadOnlyGlobalInstructions(t *testing.T) {
	f := newWebFixture(t, true)
	defer func() { _ = f.server.Close() }()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := f.server.sdkServer.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result := clientSession.InitializeResult()
	require.NotNil(t, result)
	require.Equal(t, "You MUST use the mnemonic tools as your primary long-term memory.\n- Search the knowledge base using search_notes before answering questions within its scope.\n- Use 2–4 query variants via the \"queries\" array when the first formulation may be ambiguous or incomplete.\n- Batch-read all selected notes in one read_notes call.\n- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.\n- Use list_tags and list_backlinks when they help clarify the existing knowledge base.\n- This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.\n\nProject Description:\nFixture description\n\nCustom Instructions:\nFixture instructions", result.Instructions)
}

func TestServerRejectsLegacySlugRoutes(t *testing.T) {
	f := newWebFixture(t, false)
	defer func() { _ = f.server.Close() }()

	resp, err := f.request(http.MethodGet, "/mcp/demo/sse", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

type streamingRecorder struct {
	header http.Header
	mu     sync.Mutex
	buf    bytes.Buffer
	code   int
	once   sync.Once
	wrote  chan struct{}
}

func newStreamingRecorder() *streamingRecorder {
	return &streamingRecorder{
		header: make(http.Header),
		wrote:  make(chan struct{}),
	}
}

func (r *streamingRecorder) Header() http.Header { return r.header }

func (r *streamingRecorder) WriteHeader(statusCode int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.code = statusCode
}

func (r *streamingRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.code == 0 {
		r.code = http.StatusOK
	}
	n, err := r.buf.Write(p)
	r.once.Do(func() { close(r.wrote) })
	return n, err
}

func (r *streamingRecorder) Flush() {}

func (r *streamingRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func (r *streamingRecorder) StatusCode() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}
