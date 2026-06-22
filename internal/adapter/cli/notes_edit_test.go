package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
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

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	editResult := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--append", "Next step", "--json")
	require.NoError(t, editResult.Err, "stderr: %s", editResult.Stderr)

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
	require.NoError(t, showResult.Err, "stderr: %s", showResult.Stderr)
	require.NoError(t, json.Unmarshal([]byte(showResult.Stdout), &got), "stdout: %s", showResult.Stdout)
	require.Equal(t, "## Summary\nNext step", got.Note.Body)
	require.NotEmpty(t, got.Note.ContentHash)

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	require.NoError(t, err)
	require.Equal(t, "## Summary\nNext step", got.Note.Body)
	require.Equal(t, "2026-06-02T12:34:56Z", parsed.CreatedAt.UTC().Format(time.RFC3339))
	require.Equal(t, "2026-06-02T12:35:56Z", parsed.UpdatedAt.UTC().Format(time.RFC3339))
}

func TestNotesEditCommandReplacesBodyFromFile(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	clock := testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	restoreClock := project.SetClock(clock)
	defer restoreClock()

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))
	bodyFile := filepath.Join(projectRoot, "body.md")
	require.NoError(t, os.WriteFile(bodyFile, []byte("Replacement body\n"), 0o644))

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	require.NoError(t, err)
	require.Equal(t, "Replacement body\n", string(parsed.Body))
	require.Equal(t, "2026-06-02T12:34:56Z", parsed.CreatedAt.UTC().Format(time.RFC3339))
	require.Equal(t, "2026-06-02T12:35:56Z", parsed.UpdatedAt.UTC().Format(time.RFC3339))
}

func TestNotesEditCommandAllowsEmptyBodyFile(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	clock := testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	restoreClock := project.SetClock(clock)
	defer restoreClock()

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))
	bodyFile := filepath.Join(projectRoot, "body.md")
	require.NoError(t, os.WriteFile(bodyFile, nil, 0o644))

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	parsed, err := markdown.ParseNote(readNoteFile(t, notePath))
	require.NoError(t, err)
	require.Empty(t, parsed.Body)
	require.Equal(t, "2026-06-02T12:34:56Z", parsed.CreatedAt.UTC().Format(time.RFC3339))
	require.Equal(t, "2026-06-02T12:35:56Z", parsed.UpdatedAt.UTC().Format(time.RFC3339))
}

func TestNotesEditCommandSetsFrontmatterField(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	clock := testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	restoreClock := project.SetClock(clock)
	defer restoreClock()

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--set", "type=decision", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	showResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	require.NoError(t, showResult.Err, "stderr: %s", showResult.Stderr)
	var got struct {
		Note struct {
			Frontmatter map[string]any `json:"frontmatter"`
		} `json:"note"`
	}
	require.NoError(t, json.Unmarshal([]byte(showResult.Stdout), &got), "stdout: %s", showResult.Stdout)
	require.Equal(t, "decision", got.Note.Frontmatter["type"])
}

func TestNotesEditCommandRejectsProtectedFrontmatterField(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	bodyFile := filepath.Join(projectRoot, "body.md")
	require.NoError(t, os.WriteFile(bodyFile, []byte("replacement\n"), 0o644))

	result := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--body-file", bodyFile, "--set", "created_at=2026-06-02T12:00:00Z", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 5, ExitCodeForError(result.Err))
}

func TestNotesEditCommandEnforcesIfMatch(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	showResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	require.NoError(t, showResult.Err, "stderr: %s", showResult.Stderr)
	var showGot struct {
		Note struct {
			ContentHash string `json:"content_hash"`
		} `json:"note"`
	}
	require.NoError(t, json.Unmarshal([]byte(showResult.Stdout), &showGot), "stdout: %s", showResult.Stdout)

	first := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--if-match", showGot.Note.ContentHash, "--append", "A", "--json")
	require.NoError(t, first.Err, "stderr: %s", first.Stderr)

	second := executeCommand("notes", "edit", "auth-migration", "--project", "personal", "--if-match", showGot.Note.ContentHash, "--append", "B", "--json")
	require.Error(t, second.Err)
	require.Equal(t, 5, ExitCodeForError(second.Err))
	require.Contains(t, second.Stderr, "content hash mismatch")

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Contains(t, string(data), "A")
	require.NotContains(t, string(data), "B")
}

func readNoteFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
