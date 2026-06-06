package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesCreateCommandCreatesMarkdownNote(t *testing.T) {
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

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--json")
	if result.Err != nil {
		t.Fatalf("notes create returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got struct {
		NoteID      string `json:"note_id"`
		Slug        string `json:"slug"`
		Path        string `json:"path"`
		ContentHash string `json:"content_hash"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if got.NoteID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("note_id = %q", got.NoteID)
	}
	if got.Slug != "auth-migration" {
		t.Fatalf("slug = %q", got.Slug)
	}
	if got.Path != "auth-migration.md" {
		t.Fatalf("path = %q", got.Path)
	}
	if got.ContentHash == "" {
		t.Fatal("content_hash is empty")
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if note.MnemonicNoteID != got.NoteID {
		t.Fatalf("MnemonicNoteID = %q, want %q", note.MnemonicNoteID, got.NoteID)
	}
	if note.Title != "Auth migration" {
		t.Fatalf("Title = %q", note.Title)
	}
	if note.Slug != "auth-migration" {
		t.Fatalf("Slug = %q", note.Slug)
	}
}

func TestNotesCreateCommandReadsBodyFromStdin(t *testing.T) {
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

	body := []byte("## Summary\n\nPlan.\n")
	restoreStdin := setStdin(t, body)
	defer restoreStdin()

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--stdin", "--json")
	if result.Err != nil {
		t.Fatalf("notes create returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if string(note.Body) != string(body) {
		t.Fatalf("body = %q, want %q", note.Body, body)
	}
}

func TestNotesCreateCommandRejectsStdinAndBodyFile(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	writeLocalProjectFixture(t, projectRoot, "personal")
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	bodyFile := filepath.Join(projectRoot, "body.md")
	if err := os.WriteFile(bodyFile, []byte("replacement\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--stdin", "--body-file", bodyFile, "--json")
	if result.Err == nil {
		t.Fatal("notes create error = nil, want usage error")
	}
	if ExitCodeForError(result.Err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCodeForError(result.Err))
	}
}

func TestNotesCreateCommandWritesDeduplicatedTags(t *testing.T) {
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

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Tagged", "--tag", "django", "--tag", "auth", "--tag", "django", "--json")
	if result.Err != nil {
		t.Fatalf("notes create returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "tagged.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if got, want := note.Tags, []string{"django", "auth"}; len(got) != len(want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("tags = %#v, want %#v", got, want)
			}
		}
	}
}

func chdirForNotesTest(t *testing.T, dir string) func() {
	t.Helper()

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	return func() {
		_ = os.Chdir(prev)
	}
}

func setStdin(t *testing.T, content []byte) func() {
	t.Helper()

	original := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	if _, err := io.Copy(w, bytes.NewReader(content)); err != nil {
		_ = r.Close()
		_ = w.Close()
		t.Fatalf("copy stdin content error = %v", err)
	}
	if err := w.Close(); err != nil {
		_ = r.Close()
		t.Fatalf("close write pipe error = %v", err)
	}
	os.Stdin = r
	return func() {
		os.Stdin = original
		_ = r.Close()
	}
}
