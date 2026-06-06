package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesDeleteCommandDryRunShowsTrashPath(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	initial := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth migration",
		Slug:           "auth-migration",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n"),
	}
	rendered, err := markdown.RenderNote(initial)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(notePath, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--dry-run", "--json")
	if result.Err != nil {
		t.Fatalf("notes delete returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	var got struct {
		Mode      string `json:"mode"`
		TrashPath string `json:"trash_path"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "trash" {
		t.Fatalf("mode = %q, want trash", got.Mode)
	}
	if got.TrashPath == "" {
		t.Fatal("trash_path is empty")
	}
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("source note stat error = %v", err)
	}
	if _, err := os.Stat(got.TrashPath); !os.IsNotExist(err) {
		t.Fatalf("trash note stat = %v, want not exist", err)
	}
}

func TestNotesDeleteCommandMovesNoteToTrash(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	initial := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth migration",
		Slug:           "auth-migration",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n"),
	}
	rendered, err := markdown.RenderNote(initial)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(notePath, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("notes delete returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	var got struct {
		Mode      string `json:"mode"`
		Path      string `json:"path"`
		TrashPath string `json:"trash_path"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "trash" {
		t.Fatalf("mode = %q, want trash", got.Mode)
	}
	if got.Path != "auth-migration.md" {
		t.Fatalf("path = %q", got.Path)
	}
	if got.TrashPath == "" {
		t.Fatal("trash_path is empty")
	}
	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Fatalf("source note stat = %v, want not exist", err)
	}
	if _, err := os.Stat(got.TrashPath); err != nil {
		t.Fatalf("trash note stat = %v", err)
	}

	listResult := executeCommand("notes", "list", "--project", "personal", "--json")
	if listResult.Err != nil {
		t.Fatalf("notes list returned error: %v\nstderr: %s", listResult.Err, listResult.Stderr)
	}
	var listGot struct {
		Notes []any `json:"notes"`
	}
	if err := json.Unmarshal([]byte(listResult.Stdout), &listGot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, listResult.Stdout)
	}
	if len(listGot.Notes) != 0 {
		t.Fatalf("notes len = %d, want 0", len(listGot.Notes))
	}
}

func TestNotesDeleteCommandHardRequiresYes(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	initial := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth migration",
		Slug:           "auth-migration",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n"),
	}
	rendered, err := markdown.RenderNote(initial)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(notePath, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--hard", "--json")
	if result.Err == nil {
		t.Fatal("notes delete error = nil, want unsafe error")
	}
	if ExitCodeForError(result.Err) != 5 {
		t.Fatalf("exit code = %d, want 5", ExitCodeForError(result.Err))
	}
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("source note stat error = %v", err)
	}
}

func TestNotesDeleteCommandHardDeletesWhenConfirmed(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	initial := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth migration",
		Slug:           "auth-migration",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n"),
	}
	rendered, err := markdown.RenderNote(initial)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(notePath, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--hard", "--yes", "--json")
	if result.Err != nil {
		t.Fatalf("notes delete returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	var got struct {
		Mode string `json:"mode"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if got.Mode != "hard" {
		t.Fatalf("mode = %q, want hard", got.Mode)
	}
	if got.Path != "auth-migration.md" {
		t.Fatalf("path = %q", got.Path)
	}
	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Fatalf("source note stat = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".trash")); !os.IsNotExist(err) {
		t.Fatalf("trash dir stat = %v, want not exist", err)
	}
}
