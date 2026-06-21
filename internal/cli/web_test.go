package cli

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestWebHelpShowsOnlyServe(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("web", "--help")
	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, "serve")
	require.NotContains(t, result.Stdout, "users")
	require.NotContains(t, result.Stdout, "perms")
}

func TestWebServeOpensIndexOnceBeforeListen(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	require.NoError(t, os.MkdirAll(filepath.Join(memoriesHome, "demo"), 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(memoriesHome, "demo", "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(memoriesHome, "demo", "demo.md"), []byte("# Demo\n"), 0o644))
	_, err := index.RebuildProjectIndex(manifest.ProjectID, filepath.Join(memoriesHome, "demo"))
	require.NoError(t, err)

	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_PROJECT", "demo")
	t.Setenv("MNEMONIC_WEB_ADDR", ":9090")

	var calls atomic.Int32
	origOpen := openWebIndexDB
	openWebIndexDB = func(resolution app.ProjectResolution) (*sql.DB, error) {
		calls.Add(1)
		return sql.Open("sqlite", ":memory:")
	}
	t.Cleanup(func() {
		openWebIndexDB = origOpen
	})

	result := executeCommand("web", "serve", "--port", "not-a-port")
	require.Error(t, result.Err)
	require.Equal(t, int(app.CodeInternal), ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "serve web MCP")
	require.EqualValues(t, 1, calls.Load())
}

func TestWebServeRejectsMissingProjectSelection(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("web", "serve", "--port", "not-a-port")
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "no project selected")
}

func TestWebServeRejectsInvalidProjectSelection(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", filepath.Join(root, ".mnemonic"))
	t.Setenv("MNEMONIC_PROJECT", "missing")

	result := executeCommand("web", "serve", "--port", "not-a-port")
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), `project "missing" not found`)
}
