package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	clockpkg "github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectInitCommandRequiresName(t *testing.T) {
	result := executeCommand("project", "init")
	require.Error(t, result.Err, "expected init without NAME to fail")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestProjectInitCommandRejectsMultipleNames(t *testing.T) {
	result := executeCommand("project", "init", "one", "two")
	require.Error(t, result.Err, "expected init with multiple NAME args to fail")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestProjectInitCommandLocalCreatesLocalProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	result := executeCommand("project", "init", "backend", "--local")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	projectPath := filepath.Join(cwd, ".mnemonic")
	_, err = os.Stat(projectPath)
	require.True(t, os.IsNotExist(err), ".mnemonic anchor file should not be created")

	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	_, err = os.Stat(localPath)
	require.NoError(t, err, "local memories directory missing")

	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")
	_, err = os.Stat(manifestPath)
	require.True(t, os.IsNotExist(err), "unexpected central manifest")

	indexPath := testIndexPath(cwd, "550e8400-e29b-41d4-a716-446655440000")
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectInitCommandCentralCreatesCentralProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	result := executeCommand("project", "init", "personal")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "personal", "mnemonic.toml")

	_, err = os.Stat(projectPath)
	require.True(t, os.IsNotExist(err), ".mnemonic exists or stat failed unexpectedly")

	_, err = os.Stat(manifestPath)
	require.NoError(t, err, "manifest file missing")

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := manifestfmt.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, manifestfmt.ManifestType(""), parsedManifest.Type)

	indexPath := testIndexPath(cwd, parsedManifest.ProjectID)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectInitCommandRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	first := executeCommand("project", "init", "backend", "--local")
	require.NoError(t, first.Err, "stderr: %s", first.Stderr)

	restore()
	restore = clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	result := executeCommand("project", "init", "Backend", "--local")
	require.Error(t, result.Err, "second init error = nil, want duplicate slug rejection")
	require.Equal(t, 4, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, `project slug "backend" already exists`)
}

func TestProjectInitCommandCentralWithDescription(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	desc := "Backend architecture decisions, API contracts, and database schemas."
	result := executeCommand("project", "init", "backend", "--description", desc)
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")
	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := manifestfmt.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, desc, parsedManifest.Description)
}

func TestProjectInitCommandHumanWarnsOnStaleIndex(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	stateHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	require.NoError(t, os.MkdirAll(filepath.Join(stateHome, "mnemonic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateHome, "mnemonic", "projects"), []byte("block rebuild"), 0o644))
	t.Setenv("MNEMONIC_STATE_HOME", stateHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	result := executeCommand("project", "init", "backend")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, "backend initialized")
	require.Contains(t, result.Stderr, "warning: index is stale:")
}

func TestProjectInitCommandJsonReportsStaleIndex(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	stateHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	require.NoError(t, os.MkdirAll(filepath.Join(stateHome, "mnemonic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateHome, "mnemonic", "projects"), []byte("block rebuild"), 0o644))
	t.Setenv("MNEMONIC_STATE_HOME", stateHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := clockpkg.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	result := executeCommand("project", "init", "backend", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	require.Empty(t, result.Stderr)
	require.Contains(t, result.Stdout, `"index_status": "stale"`)
	require.Contains(t, result.Stdout, `"index_error":`)

	var got struct {
		IndexStatus string `json:"index_status"`
		IndexError  string `json:"index_error"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Equal(t, "stale", got.IndexStatus)
	require.NotEmpty(t, got.IndexError)
}
