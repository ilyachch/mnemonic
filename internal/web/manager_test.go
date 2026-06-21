package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

type webFixture struct {
	manager *ServerManager
	count   *atomic.Int32
}

func newWebFixture(t *testing.T) *webFixture {
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

	manager, err := NewServerManager(app.ProjectResolution{
		MnemonicFilePath: filepath.Join(projectRoot, "mnemonic.toml"),
		RepoRootAbs:      memoriesHome,
		MemoriesAbs:      projectRoot,
		ManifestAbs:      filepath.Join(projectRoot, "mnemonic.toml"),
		Project: app.ProjectRecord{
			ID:           projectID,
			Name:         "Demo",
			Slug:         slug,
			Kind:         "central",
			MemoriesPath: filepath.Base(projectRoot),
		},
	}, memoriesHome)
	require.NoError(t, err)

	count := &atomic.Int32{}
	baseFactory := manager.instanceFactory
	manager.instanceFactory = func() (*ProjectInstance, error) {
		count.Add(1)
		return baseFactory()
	}

	t.Cleanup(func() {
		_ = manager.Close()
	})

	return &webFixture{
		manager: manager,
		count:   count,
	}
}

func (f *webFixture) request(method, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://example.test"+path, nil)
	if err != nil {
		return nil, err
	}

	recorder := httptest.NewRecorder()
	f.manager.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func TestServerManagerRoutesConfiguredSlugWithoutAuth(t *testing.T) {
	f := newWebFixture(t)

	resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = f.request(http.MethodPost, "/mcp/other/messages?sessionid=missing")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.EqualValues(t, 1, f.count.Load())
}

func TestServerManagerCachesSingleInstance(t *testing.T) {
	f := newWebFixture(t)

	const concurrency = 8
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := f.request(http.MethodPost, "/mcp/demo/messages?sessionid=missing")
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
