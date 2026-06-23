package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestTagsListCommandReturnsCountsAndSorts(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Note One", "note-one", []string{"django"}, "tagged twice #django\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-two.md"), "550e8400-e29b-41d4-a716-446655440002", "Note Two", "note-two", []string{"auth"}, "auth only\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-three.md"), "550e8400-e29b-41d4-a716-446655440003", "Note Three", "note-three", []string{"auth"}, "mixed tags #django\n")
	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("tags", "list", "--project", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Tags []struct {
			Tag   string `json:"tag"`
			Count int    `json:"count"`
		} `json:"tags"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Len(t, got.Tags, 2)
	require.Equal(t, "auth", got.Tags[0].Tag)
	require.Equal(t, 2, got.Tags[0].Count)
	require.Equal(t, "django", got.Tags[1].Tag)
	require.Equal(t, 2, got.Tags[1].Count)
}

func TestTagsListCommandMissingIndexSuggestsReindex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("tags", "list", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, "mnemonic project reindex")
}
