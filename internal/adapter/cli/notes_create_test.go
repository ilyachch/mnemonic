package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	clockpkg "github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesCreateCommandCreatesMarkdownNote(t *testing.T) {
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

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		NoteID      string `json:"note_id"`
		Slug        string `json:"slug"`
		Path        string `json:"path"`
		ContentHash string `json:"content_hash"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.NoteID)
	require.Equal(t, "auth-migration", got.Slug)
	require.Equal(t, "auth-migration.md", got.Path)
	require.NotEmpty(t, got.ContentHash)

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, got.NoteID, note.MnemonicNoteID)
	require.Equal(t, "Auth migration", note.Title)
	require.Equal(t, "auth-migration", note.Slug)
}

func TestNotesCreateCommandReadsBodyFromStdin(t *testing.T) {
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

	body := []byte("## Summary\n\nPlan.\n")
	restoreStdin := setStdin(t, body)
	defer restoreStdin()

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--stdin", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, string(body), string(note.Body))
}

func TestNotesCreateCommandRejectsStdinAndBodyFile(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))
	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	bodyFile := filepath.Join(projectRoot, "body.md")
	require.NoError(t, os.WriteFile(bodyFile, []byte("replacement\n"), 0o644))

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Auth migration", "--stdin", "--body-file", bodyFile, "--json")
	require.Error(t, result.Err)
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestNotesCreateCommandWritesDeduplicatedTags(t *testing.T) {
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

	result := executeCommand("notes", "create", "--project", "personal", "--title", "Tagged", "--tag", "django", "--tag", "auth", "--tag", "django", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	data, err := os.ReadFile(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "tagged.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, []string{"django", "auth"}, note.Tags)
}

func chdirForNotesTest(t *testing.T, dir string) func() {
	t.Helper()

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	return func() {
		_ = os.Chdir(prev)
	}
}

func setStdin(t *testing.T, content []byte) func() {
	t.Helper()

	original := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = io.Copy(w, bytes.NewReader(content))
	if err != nil {
		_ = r.Close()
		_ = w.Close()
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	os.Stdin = r
	return func() {
		os.Stdin = original
		_ = r.Close()
	}
}
