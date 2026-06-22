package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	clockpkg "github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesShowCommandReturnsJSONAndHumanOutput(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	testClock := testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	restoreClock := clockpkg.SetClock(testClock)
	defer restoreClock()

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	jsonResult := executeCommand("notes", "show", "auth-migration", "--project", "personal", "--json")
	require.NoError(t, jsonResult.Err, "stderr: %s", jsonResult.Stderr)

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
	require.NoError(t, json.Unmarshal([]byte(jsonResult.Stdout), &got), "stdout: %s", jsonResult.Stdout)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.Note.NoteID)
	require.Equal(t, "auth-migration.md", got.Note.Path)
	require.Equal(t, string(note.Body), got.Note.Body)
	require.NotEmpty(t, got.Note.ContentHash)

	humanResult := executeCommand("notes", "show", "auth-migration", "--project", "personal")
	require.NoError(t, humanResult.Err, "stderr: %s", humanResult.Stderr)
	require.Equal(t, string(rendered), humanResult.Stdout)
}

func TestNotesShowCommandMissingSelector(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "show", "missing", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))
}
