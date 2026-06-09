package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
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
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}

	result := executeCommand("project", "remove", "backend", "--json")
	if result.Err != nil {
		t.Fatalf("project remove returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectRemoveJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.ProjectID != projectID {
		t.Fatalf("project_id = %q, want %q", got.ProjectID, projectID)
	}
	if got.Mode != "soft" {
		t.Fatalf("mode = %q, want soft", got.Mode)
	}
	if !got.RegistryRemoved {
		t.Fatal("registry_removed = false, want true")
	}
	if got.IndexDeleted {
		t.Fatal("index_deleted = true, want false")
	}
	if got.StateMarkersDeleted {
		t.Fatal("state_markers_deleted = true, want false")
	}
	if got.MarkdownDeleted {
		t.Fatal("markdown_deleted = true, want false")
	}
	if got.FullWipe {
		t.Fatal("full_wipe = true, want false")
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); err != nil {
		t.Fatalf("markdown directory missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml")); err != nil {
		t.Fatalf("state marker missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock")); err != nil {
		t.Fatalf("write lock missing unexpectedly: %v", err)
	}

	list := executeCommand("project", "list", "--json")
	if list.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", list.Err, list.Stderr)
	}

	var listJSON struct {
		Projects []any `json:"projects"`
	}
	if err := json.Unmarshal([]byte(list.Stdout), &listJSON); err != nil {
		t.Fatalf("failed to decode list JSON: %v\nstdout: %s", err, list.Stdout)
	}
	if len(listJSON.Projects) != 0 {
		t.Fatalf("len(projects) = %d, want 0", len(listJSON.Projects))
	}

	second := executeCommand("project", "remove", "backend", "--json")
	if second.Err == nil {
		t.Fatal("second remove error = nil, want not-found")
	}
	if got := ExitCodeForError(second.Err); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func TestProjectRemoveCommandHardRemovesStateArtifactsButKeepsMarkdown(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}

	result := executeCommand("project", "remove", "backend", "--hard", "--json")
	if result.Err != nil {
		t.Fatalf("project remove returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectRemoveJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "hard" {
		t.Fatalf("mode = %q, want hard", got.Mode)
	}
	if !got.RegistryRemoved {
		t.Fatal("registry_removed = false, want true")
	}
	if !got.IndexDeleted {
		t.Fatal("index_deleted = false, want true")
	}
	if !got.StateMarkersDeleted {
		t.Fatal("state_markers_deleted = false, want true")
	}
	if got.MarkdownDeleted {
		t.Fatal("markdown_deleted = true, want false")
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); err != nil {
		t.Fatalf("markdown directory missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("index still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("index wal still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath + "-shm"); !os.IsNotExist(err) {
		t.Fatalf("index shm still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml")); !os.IsNotExist(err) {
		t.Fatalf("state marker still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock")); err != nil {
		t.Fatalf("write lock missing unexpectedly: %v", err)
	}
}

func TestProjectRemoveCommandDeleteMarkdownRemovesMarkdownAndIndexButKeepsRegistry(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}

	result := executeCommand("project", "remove", "backend", "--delete-markdown", "--json")
	if result.Err != nil {
		t.Fatalf("project remove returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectRemoveJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "markdown-only" {
		t.Fatalf("mode = %q, want markdown-only", got.Mode)
	}
	if got.RegistryRemoved {
		t.Fatal("registry_removed = true, want false")
	}
	if !got.IndexDeleted {
		t.Fatal("index_deleted = false, want true")
	}
	if !got.StateMarkersDeleted {
		t.Fatal("state_markers_deleted = false, want true")
	}
	if !got.MarkdownDeleted {
		t.Fatal("markdown_deleted = false, want true")
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic")); err != nil {
		t.Fatalf(".mnemonic marker missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); !os.IsNotExist(err) {
		t.Fatalf("markdown directory still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("index still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml")); !os.IsNotExist(err) {
		t.Fatalf("state marker still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "locks", "write.lock")); err != nil {
		t.Fatalf("write lock missing unexpectedly: %v", err)
	}

	show := executeCommand("project", "show", "backend", "--json")
	if show.Err != nil {
		t.Fatalf("project show returned error after --delete-markdown: %v\nstderr: %s", show.Err, show.Stderr)
	}
}

func TestProjectRemoveCommandWipeRemovesEverythingManagedByRemove(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)
	indexPath, err := index.Path(projectID)
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}

	result := executeCommand("project", "remove", "backend", "--wipe", "--json")
	if result.Err != nil {
		t.Fatalf("project remove returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectRemoveJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "wipe" {
		t.Fatalf("mode = %q, want wipe", got.Mode)
	}
	if !got.RegistryRemoved || !got.IndexDeleted || !got.StateMarkersDeleted || !got.MarkdownDeleted || !got.FullWipe {
		t.Fatalf("unexpected wipe output: %+v", got)
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); !os.IsNotExist(err) {
		t.Fatalf("markdown directory still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("index still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml")); !os.IsNotExist(err) {
		t.Fatalf("state marker still exists or stat failed unexpectedly: %v", err)
	}

	list := executeCommand("project", "list", "--json")
	if list.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", list.Err, list.Stderr)
	}
	var listJSON struct {
		Projects []any `json:"projects"`
	}
	if err := json.Unmarshal([]byte(list.Stdout), &listJSON); err != nil {
		t.Fatalf("failed to decode list JSON: %v\nstdout: %s", err, list.Stdout)
	}
	if len(listJSON.Projects) != 0 {
		t.Fatalf("len(projects) = %d, want 0", len(listJSON.Projects))
	}
}

func TestProjectRemoveCommandRejectsHardAndDeleteMarkdownTogether(t *testing.T) {
	seedRemovableLocalProject(t)

	result := executeCommand("project", "remove", "backend", "--hard", "--delete-markdown", "--json")
	if result.Err == nil {
		t.Fatal("project remove error = nil, want CLI usage error")
	}
	if got := ExitCodeForError(result.Err); got != 2 {
		t.Fatalf("exit code = %d, want 2", got)
	}
	if !strings.Contains(result.Err.Error(), "--wipe") {
		t.Fatalf("error %q does not mention --wipe", result.Err.Error())
	}
}

func seedRemovableLocalProject(t *testing.T) (cwd, stateHome, projectID string) {
	t.Helper()

	cwd = testutil.CleanEnvForTest(t)
	stateHome = filepath.Join(cwd, "state")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
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

	if err := project.InitProject(project.InitInput{
		CWD:          cwd,
		MemoriesHome: t.TempDir(),
		Name:         "backend",
		Mode:         project.InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject(local) error = %v", err)
	}

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := project.ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	projectID = parsedProject.Projects[0].ID

	stateDir := filepath.Join(stateHome, "mnemonic", "projects", projectID)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.toml"), []byte("status = \"ok\"\n"), 0o644); err != nil {
		t.Fatalf("failed to seed state file: %v", err)
	}
	indexPath := filepath.Join(stateDir, "index.sqlite")
	if err := os.WriteFile(indexPath, []byte("sqlite-index"), 0o644); err != nil {
		t.Fatalf("failed to seed index file: %v", err)
	}
	if err := os.WriteFile(indexPath+"-wal", []byte("wal"), 0o644); err != nil {
		t.Fatalf("failed to seed index wal file: %v", err)
	}
	if err := os.WriteFile(indexPath+"-shm", []byte("shm"), 0o644); err != nil {
		t.Fatalf("failed to seed index shm file: %v", err)
	}
	locksDir := filepath.Join(stateDir, "locks")
	if err := os.MkdirAll(locksDir, 0o755); err != nil {
		t.Fatalf("failed to seed locks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(locksDir, "write.lock"), []byte("lock"), 0o644); err != nil {
		t.Fatalf("failed to seed write lock: %v", err)
	}

	return cwd, stateHome, projectID
}
