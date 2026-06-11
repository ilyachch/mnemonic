package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

type projectRemoveJSON struct {
	ProjectID           string `json:"project_id"`
	Slug                string `json:"slug"`
	Mode                string `json:"mode"`
	RegistryRemoved     bool   `json:"registry_removed"`
	IndexDeleted        bool   `json:"index_deleted"`
	StateMarkersDeleted bool   `json:"state_markers_deleted"`
	MarkdownDeleted     bool   `json:"markdown_deleted"`
	FullWipe            bool   `json:"full_wipe"`
}

func TestProjectRemoveCommandSoftDeletesOnlyByDefault(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	require.NoError(t, err)

	result := executeCommand("project", "remove", "backend", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, projectID, got.ProjectID)
	require.Equal(t, "soft", got.Mode)
	require.True(t, got.RegistryRemoved, "registry_removed = false, want true")
	require.False(t, got.IndexDeleted, "index_deleted = true, want false")
	require.False(t, got.StateMarkersDeleted, "state_markers_deleted = true, want false")
	require.False(t, got.MarkdownDeleted, "markdown_deleted = true, want false")
	require.False(t, got.FullWipe, "full_wipe = true, want false")

	_, err = os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend"))
	require.NoError(t, err, "markdown directory missing unexpectedly")
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index missing unexpectedly")
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml"))
	require.NoError(t, err, "state marker missing unexpectedly")
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock"))
	require.NoError(t, err, "write lock missing unexpectedly")

	list := executeCommand("project", "list", "--json")
	require.NoError(t, list.Err, "project list returned error\nstderr: %s", list.Stderr)

	var listJSON struct {
		Projects []any `json:"projects"`
	}
	err = json.Unmarshal([]byte(list.Stdout), &listJSON)
	require.NoError(t, err, "failed to decode list JSON\nstdout: %s", list.Stdout)
	require.Len(t, listJSON.Projects, 0)

	second := executeCommand("project", "remove", "backend", "--json")
	require.Error(t, second.Err, "second remove error = nil, want not-found")
	require.Equal(t, 3, ExitCodeForError(second.Err))
}

func TestProjectRemoveCommandHardRemovesStateArtifactsButKeepsMarkdown(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	require.NoError(t, err)

	result := executeCommand("project", "remove", "backend", "--hard", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "hard", got.Mode)
	require.True(t, got.RegistryRemoved, "registry_removed = false, want true")
	require.True(t, got.IndexDeleted, "index_deleted = false, want true")
	require.True(t, got.StateMarkersDeleted, "state_markers_deleted = false, want true")
	require.False(t, got.MarkdownDeleted, "markdown_deleted = true, want false")

	_, err = os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend"))
	require.NoError(t, err, "markdown directory missing unexpectedly")
	_, err = os.Stat(indexPath)
	require.True(t, os.IsNotExist(err), "index still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(indexPath + "-wal")
	require.True(t, os.IsNotExist(err), "index wal still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(indexPath + "-shm")
	require.True(t, os.IsNotExist(err), "index shm still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml"))
	require.True(t, os.IsNotExist(err), "state marker still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock"))
	require.NoError(t, err, "write lock missing unexpectedly")
}

func TestProjectRemoveCommandDeleteMarkdownRemovesMarkdownAndIndexButKeepsRegistry(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	require.NoError(t, err)

	result := executeCommand("project", "remove", "backend", "--delete-markdown", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "markdown-only", got.Mode)
	require.False(t, got.RegistryRemoved, "registry_removed = true, want false")
	require.True(t, got.IndexDeleted, "index_deleted = false, want true")
	require.True(t, got.StateMarkersDeleted, "state_markers_deleted = false, want true")
	require.True(t, got.MarkdownDeleted, "markdown_deleted = false, want true")

	_, err = os.Stat(filepath.Join(cwd, ".mnemonic"))
	require.NoError(t, err, ".mnemonic marker missing unexpectedly")
	_, err = os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend"))
	require.True(t, os.IsNotExist(err), "markdown directory still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(indexPath)
	require.True(t, os.IsNotExist(err), "index still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml"))
	require.True(t, os.IsNotExist(err), "state marker still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock"))
	require.NoError(t, err, "write lock missing unexpectedly")

	show := executeCommand("project", "show", "backend", "--json")
	require.NoError(t, show.Err, "project show returned error after --delete-markdown\nstderr: %s", show.Stderr)
}

func TestProjectRemoveCommandWipeRemovesEverythingManagedByRemove(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	require.NoError(t, err)

	result := executeCommand("project", "remove", "backend", "--wipe", "--json")
	require.NoError(t, result.Err, "project remove returned error\nstderr: %s", result.Stderr)

	var got projectRemoveJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "wipe", got.Mode)
	require.True(t, got.RegistryRemoved, "registry_removed = false, want true")
	require.True(t, got.IndexDeleted, "index_deleted = false, want true")
	require.True(t, got.StateMarkersDeleted, "state_markers_deleted = false, want true")
	require.True(t, got.MarkdownDeleted, "markdown_deleted = false, want true")
	require.True(t, got.FullWipe, "full_wipe = false, want true")

	_, err = os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend"))
	require.True(t, os.IsNotExist(err), "markdown directory still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(indexPath)
	require.True(t, os.IsNotExist(err), "index still exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml"))
	require.True(t, os.IsNotExist(err), "state marker still exists or stat failed unexpectedly: %v", err)

	list := executeCommand("project", "list", "--json")
	require.NoError(t, list.Err, "project list returned error\nstderr: %s", list.Stderr)
	var listJSON struct {
		Projects []any `json:"projects"`
	}
	err = json.Unmarshal([]byte(list.Stdout), &listJSON)
	require.NoError(t, err, "failed to decode list JSON\nstdout: %s", list.Stdout)
	require.Len(t, listJSON.Projects, 0)
}

func TestProjectRemoveCommandRejectsHardAndDeleteMarkdownTogether(t *testing.T) {
	seedRemovableLocalProject(t)

	result := executeCommand("project", "remove", "backend", "--hard", "--delete-markdown", "--json")
	require.Error(t, result.Err, "project remove error = nil, want CLI usage error")
	require.Equal(t, 2, ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "--wipe")
}

func seedRemovableLocalProject(t *testing.T) (cwd, stateHome, projectID string) {
	t.Helper()

	cwd = testutil.CleanEnvForTest(t)
	stateHome = filepath.Join(cwd, "state")

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClock{
		now: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{
			"550e8400-e29b-41d4-a716-446655440000",
		},
	})
	t.Cleanup(restore)

	err = project.InitProject(project.InitInput{
		CWD:          cwd,
		MemoriesHome: t.TempDir(),
		Name:         "backend",
		Mode:         project.InitModeLocal,
	})
	require.NoError(t, err)

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	require.NoError(t, err)
	parsedProject, err := project.ParseMnemonicFile(projectData)
	require.NoError(t, err)
	projectID = parsedProject.Projects[0].ID

	stateDir := filepath.Join(stateHome, "mnemonic", "projects", projectID)
	err = os.MkdirAll(stateDir, 0o755)
	require.NoError(t, err, "failed to create state dir")
	err = os.WriteFile(filepath.Join(stateDir, "state.toml"), []byte("status = \"ok\"\n"), 0o644)
	require.NoError(t, err, "failed to seed state file")
	indexPath := filepath.Join(stateDir, "index.sqlite")
	err = os.WriteFile(indexPath, []byte("sqlite-index"), 0o644)
	require.NoError(t, err, "failed to seed index file")
	err = os.WriteFile(indexPath+"-wal", []byte("wal"), 0o644)
	require.NoError(t, err, "failed to seed index wal file")
	err = os.WriteFile(indexPath+"-shm", []byte("shm"), 0o644)
	require.NoError(t, err, "failed to seed index shm file")
	locksDir := filepath.Join(stateDir, "locks")
	err = os.MkdirAll(locksDir, 0o755)
	require.NoError(t, err, "failed to seed locks dir")
	err = os.WriteFile(filepath.Join(locksDir, "write.lock"), []byte("lock"), 0o644)
	require.NoError(t, err, "failed to seed write lock")

	return cwd, stateHome, projectID
}