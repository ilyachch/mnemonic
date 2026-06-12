package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--dry-run", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	var got struct {
		Mode      string `json:"mode"`
		TrashPath string `json:"trash_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Equal(t, "trash", got.Mode)
	require.NotEmpty(t, got.TrashPath)
	_, err = os.Stat(notePath)
	require.NoError(t, err)
	_, err = os.Stat(got.TrashPath)
	require.True(t, os.IsNotExist(err))
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	var got struct {
		Mode      string `json:"mode"`
		Path      string `json:"path"`
		TrashPath string `json:"trash_path"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Equal(t, "trash", got.Mode)
	require.Equal(t, "auth-migration.md", got.Path)
	require.NotEmpty(t, got.TrashPath)
	_, err = os.Stat(notePath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(got.TrashPath)
	require.NoError(t, err)

	listResult := executeCommand("notes", "list", "--project", "personal", "--json")
	require.NoError(t, listResult.Err, "stderr: %s", listResult.Stderr)
	var listGot struct {
		Notes []any `json:"notes"`
	}
	require.NoError(t, json.Unmarshal([]byte(listResult.Stdout), &listGot), "stdout: %s", listResult.Stdout)
	require.Empty(t, listGot.Notes)
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--hard", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 5, ExitCodeForError(result.Err))
	_, err = os.Stat(notePath)
	require.NoError(t, err)
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
	require.NoError(t, err)
	notePath := filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	result := executeCommand("notes", "delete", "auth-migration", "--project", "personal", "--hard", "--yes", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	var got struct {
		Mode string `json:"mode"`
		Path string `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Equal(t, "hard", got.Mode)
	require.Equal(t, "auth-migration.md", got.Path)
	_, err = os.Stat(notePath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(projectRoot, ".trash"))
	require.True(t, os.IsNotExist(err))
}
