package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

type projectRemoveJSON struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	RegistryRemoved bool   `json:"registry_removed"`
	IndexDeleted    bool   `json:"index_deleted"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	FullWipe        bool   `json:"full_wipe"`
}

func TestProjectRemoveCommand(t *testing.T) {
	_, _, projectID, memoriesHome := seedRemovableCentralProject(t)

	result := executeCommand("project", "remove", "backend", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, projectID, got.ProjectID)
	require.True(t, got.RegistryRemoved, "registry_removed = false, want true")

	// Verify markdown directory still exists
	_, err = os.Stat(filepath.Join(memoriesHome, "backend"))
	require.NoError(t, err, "markdown directory missing unexpectedly")

	// Verify manifest was removed
	_, err = os.Stat(filepath.Join(memoriesHome, "backend", "mnemonic.toml"))
	require.True(t, os.IsNotExist(err), "manifest still exists after remove")
}

func TestProjectRemoveCommandWipe(t *testing.T) {
	_, _, projectID, memoriesHome := seedRemovableCentralProject(t)

	result := executeCommand("project", "remove", "backend", "--wipe", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, projectID, got.ProjectID)
	require.True(t, got.FullWipe)

	// Verify markdown directory was removed
	_, err = os.Stat(filepath.Join(memoriesHome, "backend"))
	require.True(t, os.IsNotExist(err), "markdown directory still exists after wipe")
}

func TestProjectRemoveCommandMissingProject(t *testing.T) {
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", t.TempDir())

	result := executeCommand("project", "remove", "missing", "--json")
	require.Error(t, result.Err, "project remove error = nil, want not-found")
	require.Equal(t, 3, ExitCodeForError(result.Err))
}

func seedRemovableCentralProject(t *testing.T) (cwd, stateHome, projectID, memoriesHome string) {
	t.Helper()

	cwd = testutil.CleanEnvForTest(t)
	stateHome = filepath.Join(cwd, "state")
	memoriesHome = t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	projectID = "550e8400-e29b-41d4-a716-446655440000"

	slug := "backend"
	projectDir := writeCentralProjectFixture(t, memoriesHome, slug, projectID, time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC))

	// Create a note for wipe test
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "note.md"), []byte("# test\n"), 0o644))

	// Create an index for cleanup verification
	idxPath := testIndexPath(cwd, projectID)
	require.NoError(t, os.MkdirAll(filepath.Dir(idxPath), 0o755))
	require.NoError(t, os.WriteFile(idxPath, []byte("fake-index"), 0o644))

	return
}
