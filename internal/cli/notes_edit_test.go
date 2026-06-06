package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesEditCommandAppendsBodyAndUpdatesTimestamp(t *testing.T) {
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

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	editResult := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--append", "Next step", "--json")
	if editResult.Err != nil {
		t.Fatalf("notes edit returned error: %v\nstderr: %s", editResult.Err, editResult.Stderr)
	}

	var got struct {
		Note struct {
			NoteID      string `json:"note_id"`
			Slug        string `json:"slug"`
			Path        string `json:"path"`
			Body        string `json:"body"`
			ContentHash string `json:"content_hash"`
		} `json:"note"`
	}
	showResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	if showResult.Err != nil {
		t.Fatalf("notes show returned error: %v\nstderr: %s", showResult.Err, showResult.Stderr)
	}
	if err := json.Unmarshal([]byte(showResult.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, showResult.Stdout)
	}
	if got.Note.Body != "## Summary\nNext step" {
		t.Fatalf("body = %q", got.Note.Body)
	}
	if got.Note.ContentHash == "" {
		t.Fatal("content_hash is empty")
	}

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if got.Note.Body != "## Summary\nNext step" {
		t.Fatalf("body = %q", got.Note.Body)
	}
	if parsed.CreatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:34:56Z" {
		t.Fatalf("created_at = %s", parsed.CreatedAt)
	}
	if parsed.UpdatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:35:56Z" {
		t.Fatalf("updated_at = %s", parsed.UpdatedAt)
	}
}

func TestNotesEditCommandReplacesBodyFromFile(t *testing.T) {
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
	bodyFile := filepath.Join(projectRoot, "body.md")
	if err := os.WriteFile(bodyFile, []byte("Replacement body\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() body error = %v", err)
	}

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--json")
	if result.Err != nil {
		t.Fatalf("notes edit returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if string(parsed.Body) != "Replacement body\n" {
		t.Fatalf("body = %q", parsed.Body)
	}
	if parsed.CreatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:34:56Z" {
		t.Fatalf("created_at = %s", parsed.CreatedAt)
	}
	if parsed.UpdatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:35:56Z" {
		t.Fatalf("updated_at = %s", parsed.UpdatedAt)
	}
}

func TestNotesEditCommandAllowsEmptyBodyFile(t *testing.T) {
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
	bodyFile := filepath.Join(projectRoot, "body.md")
	if err := os.WriteFile(bodyFile, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() body error = %v", err)
	}

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--json")
	if result.Err != nil {
		t.Fatalf("notes edit returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if len(parsed.Body) != 0 {
		t.Fatalf("body = %q, want empty", parsed.Body)
	}
	if parsed.CreatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:34:56Z" {
		t.Fatalf("created_at = %s", parsed.CreatedAt)
	}
	if parsed.UpdatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:35:56Z" {
		t.Fatalf("updated_at = %s", parsed.UpdatedAt)
	}
}

func TestNotesEditCommandSetsFrontmatterField(t *testing.T) {
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
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n"),
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

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--set", "type=decision", "--json")
	if result.Err != nil {
		t.Fatalf("notes edit returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	showResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	if showResult.Err != nil {
		t.Fatalf("notes show returned error: %v\nstderr: %s", showResult.Err, showResult.Stderr)
	}
	var got struct {
		Note struct {
			Frontmatter map[string]any `json:"frontmatter"`
		} `json:"note"`
	}
	if err := json.Unmarshal([]byte(showResult.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, showResult.Stdout)
	}
	if got.Note.Frontmatter["type"] != "decision" {
		t.Fatalf("frontmatter[type] = %#v", got.Note.Frontmatter["type"])
	}
}

func TestNotesEditCommandRejectsProtectedFrontmatterField(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

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

	bodyFile := filepath.Join(projectRoot, "body.md")
	if err := os.WriteFile(bodyFile, []byte("replacement\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--set", "created_at=2026-06-02T12:00:00Z", "--json")
	if result.Err == nil {
		t.Fatal("notes edit error = nil, want unsafe error")
	}
	if ExitCodeForError(result.Err) != 5 {
		t.Fatalf("exit code = %d, want 5", ExitCodeForError(result.Err))
	}
}

func TestNotesEditCommandEnforcesIfMatch(t *testing.T) {
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

	showResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	if showResult.Err != nil {
		t.Fatalf("notes show returned error: %v\nstderr: %s", showResult.Err, showResult.Stderr)
	}
	var showGot struct {
		Note struct {
			ContentHash string `json:"content_hash"`
		} `json:"note"`
	}
	if err := json.Unmarshal([]byte(showResult.Stdout), &showGot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, showResult.Stdout)
	}

	first := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--if-match", showGot.Note.ContentHash, "--append", "A", "--json")
	if first.Err != nil {
		t.Fatalf("first notes edit returned error: %v\nstderr: %s", first.Err, first.Stderr)
	}

	second := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--if-match", showGot.Note.ContentHash, "--append", "B", "--json")
	if second.Err == nil {
		t.Fatal("second notes edit error = nil, want unsafe error")
	}
	if ExitCodeForError(second.Err) != 5 {
		t.Fatalf("exit code = %d, want 5", ExitCodeForError(second.Err))
	}
	if got := second.Stderr; !strings.Contains(got, "content hash mismatch") {
		t.Fatalf("stderr = %q, want content hash mismatch", got)
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "A") {
		t.Fatalf("note body = %q, want contains A", string(data))
	}
	if strings.Contains(string(data), "B") {
		t.Fatalf("note body = %q, want not contains B", string(data))
	}
}

func readNoteFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}
