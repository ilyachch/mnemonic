package cli

import (
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"

	webadapter "github.com/ilyachch/mnemonic/internal/adapter/web"
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestWebHelpShowsOnlyServe(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("web", "--help")
	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, "serve")
	require.NotContains(t, result.Stdout, "users")
	require.NotContains(t, result.Stdout, "perms")
}

func TestWebServeBuildsRuntimeOnceBeforeListen(t *testing.T) {
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_WEB_ADDR", ":9090")

	var capturedRuntime struct {
		ProjectID    string
		ProjectToken string
		ReadOnly     bool
	}
	var calls atomic.Int32
	var sequence []string
	runtimeKB := kb.KnowledgeBase{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Name:    "Demo",
		Slug:    "demo",
		Kind:    "central",
		RootDir: filepath.Join(t.TempDir(), "demo"),
	}
	origNew := newWebServer
	origListen := webServeListenAndServe
	origResolve := resolveRuntimeApp
	resolveRuntimeApp = func(_ *cobra.Command) (*app.RuntimeApp, error) {
		sequence = append(sequence, "resolve")
		runtime, err := app.NewRuntimeApp(app.RuntimeInput{KB: runtimeKB})
		if err != nil {
			return nil, err
		}
		sequence = append(sequence, "runtime")
		return runtime, nil
	}
	newWebServer = func(input webadapter.ServerInput) (*webadapter.Server, error) {
		sequence = append(sequence, "build")
		calls.Add(1)
		capturedRuntime.ProjectID = input.KB.ID
		capturedRuntime.ProjectToken = input.ProjectToken
		capturedRuntime.ReadOnly = input.ReadOnly
		return &webadapter.Server{}, nil
	}
	webServeListenAndServe = func(server *http.Server) error {
		require.EqualValues(t, 1, calls.Load())
		sequence = append(sequence, "listen")
		require.Equal(t, ":9090", server.Addr)
		require.NotNil(t, server.Handler)
		return http.ErrServerClosed
	}
	t.Cleanup(func() {
		newWebServer = origNew
		webServeListenAndServe = origListen
		resolveRuntimeApp = origResolve
	})

	result := executeCommand("web", "serve")
	require.NoError(t, result.Err)
	require.EqualValues(t, 1, calls.Load())
	require.Equal(t, []string{"resolve", "runtime", "build", "listen"}, sequence)
	require.Equal(t, runtimeKB.ID, capturedRuntime.ProjectID)
	require.False(t, capturedRuntime.ReadOnly)
}

func TestWebServeRejectsMissingProjectSelection(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("web", "serve")
	require.Error(t, result.Err)
	require.Equal(t, int(apperr.CodeCLIUsage), ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "no project selected")
}

func TestWebServeRejectsInvalidProjectSelection(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", filepath.Join(root, ".mnemonic"))
	t.Setenv("MNEMONIC_PROJECT", "missing")

	result := executeCommand("web", "serve")
	require.Error(t, result.Err)
	require.Equal(t, int(apperr.CodeNotFound), ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), `project "missing" not found`)
}
