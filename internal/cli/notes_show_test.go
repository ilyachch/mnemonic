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

func TestNotesShowCommandReturnsJSONAndHumanOutput(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	clock := testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	restoreClock := project.SetClock(clock)
	defer restoreClock()

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth migration",
		Slug:           "auth-migration",
		Tags:           []string{"django", "auth"},
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC),
		Body:           []byte("## Summary\n\nPlan.\n"),
	}
	rendered, err := markdown.RenderNote(note)
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

	jsonResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	if jsonResult.Err != nil {
		t.Fatalf("notes show json returned error: %v\nstderr: %s", jsonResult.Err, jsonResult.Stderr)
	}

	var got struct {
		Note struct {
			NoteID      string         `json:"note_id"`
			Slug        string         `json:"slug"`
			Title       string         `json:"title"`
			Path        string         `json:"path"`
			Frontmatter map[string]any `json:"frontmatter"`
			Body        string         `json:"body"`
			ContentHash string         `json:"content_hash"`
			UpdatedAt   string         `json:"updated_at"`
		} `json:"note"`
	}
	if err := json.Unmarshal([]byte(jsonResult.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, jsonResult.Stdout)
	}
	if got.Note.NoteID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("note_id = %q", got.Note.NoteID)
	}
	if got.Note.Path != "auth-migration.md" {
		t.Fatalf("path = %q", got.Note.Path)
	}
	if got.Note.Body != string(note.Body) {
		t.Fatalf("body = %q, want %q", got.Note.Body, string(note.Body))
	}
	if got.Note.ContentHash == "" {
		t.Fatal("content_hash is empty")
	}

	humanResult := executeCommand("notes", "show", "auth-migration", "--project", "personal")
	if humanResult.Err != nil {
		t.Fatalf("notes show human returned error: %v\nstderr: %s", humanResult.Err, humanResult.Stderr)
	}
	if humanResult.Stdout != string(rendered) {
		t.Fatalf("stdout = %q, want %q", humanResult.Stdout, string(rendered))
	}
}

func TestNotesShowCommandMissingSelector(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "show", "missing", "--project", "personal", "--json")
	if result.Err == nil {
		t.Fatal("notes show error = nil, want not found error")
	}
	if ExitCodeForError(result.Err) != 3 {
		t.Fatalf("exit code = %d, want 3", ExitCodeForError(result.Err))
	}
}
