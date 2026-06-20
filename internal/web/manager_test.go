package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/ilyachch/mnemonic/internal/webauth"
	"github.com/stretchr/testify/require"
)

type webFixture struct {
	manager *ServerManager
	store   *webauth.Store
	token   string
	userID  string
	count   *atomic.Int32
}

func newWebFixture(t *testing.T, superuserToken string) *webFixture {
	t.Helper()

	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	slug := "demo"
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectRoot := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = projectID
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "demo.md"), []byte("# Demo\n\nBody"), 0o644))
	_, err := index.RebuildProjectIndex(projectID, projectRoot)
	require.NoError(t, err)

	authPath, err := webauth.ResolveDBPath("")
	require.NoError(t, err)
	store, err := webauth.NewStore(authPath)
	require.NoError(t, err)

	token := "user-token-123"
	user, err := store.CreateUser(context.Background(), "alice", token)
	require.NoError(t, err)

	manager, err := NewServerManager(store, superuserToken, memoriesHome, []string{slug})
	require.NoError(t, err)

	count := &atomic.Int32{}
	baseFactory := manager.instanceFactory
	manager.instanceFactory = func(requested string) (*ProjectInstance, error) {
		count.Add(1)
		return baseFactory(requested)
	}

	t.Cleanup(func() {
		_ = manager.Close()
	})

	return &webFixture{
		manager: manager,
		store:   store,
		token:   token,
		userID:  user.UserID,
		count:   count,
	}
}

func (f *webFixture) request(method, path, token string) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://example.test"+path, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	f.manager.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func TestServerManagerAuth(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		f := newWebFixture(t, "")
		resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", "")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid token", func(t *testing.T) {
		f := newWebFixture(t, "")
		resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", "does-not-exist")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("missing permission", func(t *testing.T) {
		f := newWebFixture(t, "")
		resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("superuser bypass", func(t *testing.T) {
		f := newWebFixture(t, "super-token")
		resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", "super-token")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.EqualValues(t, 1, f.count.Load())
	})
}

func TestServerManagerLazyInitAndCache(t *testing.T) {
	f := newWebFixture(t, "")
	require.NoError(t, f.store.GrantPermission(context.Background(), f.userID, "demo", "ro"))

	resp1, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp1.StatusCode)
	_ = resp1.Body.Close()

	resp2, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)
	_ = resp2.Body.Close()

	require.EqualValues(t, 1, f.count.Load())
}

func TestServerManagerCachesInstancesByAccessLevel(t *testing.T) {
	f := newWebFixture(t, "")

	rwUser, err := f.store.CreateUser(context.Background(), "bob", "user-token-456")
	require.NoError(t, err)
	require.NoError(t, f.store.GrantPermission(context.Background(), f.userID, "demo", "ro"))
	require.NoError(t, f.store.GrantPermission(context.Background(), rwUser.UserID, "demo", "rw"))

	resp1, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp1.StatusCode)
	_ = resp1.Body.Close()

	resp2, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", "user-token-456")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)
	_ = resp2.Body.Close()

	resp3, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp3.StatusCode)
	_ = resp3.Body.Close()

	require.EqualValues(t, 2, f.count.Load())
}

func TestServerManagerConcurrentRequestsShareOneInstance(t *testing.T) {
	f := newWebFixture(t, "")
	require.NoError(t, f.store.GrantPermission(context.Background(), f.userID, "demo", "ro"))

	const concurrency = 8
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing", f.token)
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				errCh <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, f.count.Load())
}
