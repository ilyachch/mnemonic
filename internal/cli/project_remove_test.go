package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

type projectRemoveJSON struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	Removed         bool   `json:"removed"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	StateDeleted    bool   `json:"state_deleted"`
}

func TestProjectRemoveCommandPreservesMarkdownByDefault(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)

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
	if !got.Removed {
		t.Fatal("removed = false, want true")
	}
	if got.MarkdownDeleted {
		t.Fatal("markdown_deleted = true, want false")
	}
	if !got.StateDeleted {
		t.Fatal("state_deleted = false, want true")
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); err != nil {
		t.Fatalf("markdown directory missing unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID)); !os.IsNotExist(err) {
		t.Fatalf("state directory still exists or stat failed unexpectedly: %v", err)
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

func TestProjectRemoveCommandDeletesMarkdownWhenRequested(t *testing.T) {
	cwd, stateHome, projectID := seedRemovableLocalProject(t)

	result := executeCommand("project", "remove", "backend", "--delete-markdown", "--json")
	if result.Err != nil {
		t.Fatalf("project remove returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectRemoveJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if !got.MarkdownDeleted {
		t.Fatal("markdown_deleted = false, want true")
	}

	if _, err := os.Stat(filepath.Join(cwd, ".mnemonic-memories", "backend")); !os.IsNotExist(err) {
		t.Fatalf("markdown directory still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", projectID)); !os.IsNotExist(err) {
		t.Fatalf("state directory still exists or stat failed unexpectedly: %v", err)
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

	return cwd, stateHome, projectID
}
