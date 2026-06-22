package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesBacklinksCommandReturnsLinks(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440001", "Target Note", "target-note", nil, "target body\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "source-note.md"), "550e8400-e29b-41d4-a716-446655440002", "Source Note", "source-note", nil, "[[Target Note]]\n## Relations\n- depends_on [[Target Note]]\n- relates_to [[Target Note]]\n")

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("notes", "backlinks", "target-note", "--project", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Links []struct {
			NoteID       string `json:"note_id"`
			Slug         string `json:"slug"`
			Title        string `json:"title"`
			Path         string `json:"path"`
			RelationType string `json:"relation_type"`
			SourceLine   int    `json:"source_line"`
		} `json:"links"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Links)
	for _, link := range got.Links {
		require.Equal(t, "550e8400-e29b-41d4-a716-446655440002", link.NoteID)
		require.Positive(t, link.SourceLine)
		require.Equal(t, "Source Note", link.Title)
	}
}

func TestNotesBacklinksCommandMissingNoteAndMissingIndex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "backlinks", "missing-note", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440001", "Target Note", "target-note", nil, "target body\n")

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result = executeCommand("notes", "backlinks", "target-note", "--project", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Links []struct {
			NoteID       string `json:"note_id"`
			Slug         string `json:"slug"`
			Title        string `json:"title"`
			Path         string `json:"path"`
			RelationType string `json:"relation_type"`
			SourceLine   int    `json:"source_line"`
		} `json:"links"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Empty(t, got.Links)
}
