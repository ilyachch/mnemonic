package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesSearchCommandReturnsHits(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "auth-migration.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth migration", "auth-migration", nil, "Search this body.\nObservation queryterm.\n")

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--json", "--debug")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Hits []struct {
			NoteID      string  `json:"note_id"`
			Slug        string  `json:"slug"`
			Title       string  `json:"title"`
			Path        string  `json:"path"`
			Snippet     string  `json:"snippet"`
			Score       float64 `json:"score"`
			ContentHash string  `json:"content_hash"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Hits)
	first := got.Hits[0]
	require.NotEmpty(t, first.NoteID)
	require.NotEmpty(t, first.Slug)
	require.NotEmpty(t, first.Title)
	require.NotEmpty(t, first.Path)
	require.NotEmpty(t, first.Snippet)
	require.NotEmpty(t, first.ContentHash)
	require.Equal(t, "auth-migration", first.Slug)
}

func TestNotesSearchCommandRespectsLimit(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	for i := 0; i < 3; i++ {
		title := "Query Term " + string(rune('A'+i))
		uid := "550e8400-e29b-41d4-a716-44665544000" + string(rune('2'+i))
		writeTaggedNote(t, filepath.Join(projectRoot, ".mnemonic-memories", "personal", "query-term-"+string(rune('a'+i))+".md"), uid, title, "query-term-"+string(rune('a'+i)), nil, "queryterm queryterm\n")
	}

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("notes", "search", "--query", "queryterm", "--project", "personal", "--limit", "2", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	var got struct {
		Hits []any `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Len(t, got.Hits, 2)
}

func TestNotesSearchCommandFiltersByTag(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "frontmatter-tag.md"), "550e8400-e29b-41d4-a716-446655440001", "Frontmatter tag", "frontmatter-tag", []string{"python"}, "auth queryterm\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "inline-tag.md"), "550e8400-e29b-41d4-a716-446655440002", "Inline tag", "inline-tag", nil, "auth queryterm #python\n")
	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--tag", "python", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)
	var got struct {
		Hits []struct {
			Slug string `json:"slug"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.Len(t, got.Hits, 2)
}

func TestNotesSearchCommandMissingIndexSuggestsReindex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, "mnemonic project reindex")
}

func TestNotesSearchCommandWithoutQueryArgUsesTimeFilter(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "recent-note.md"), "550e8400-e29b-41d4-a716-446655440001", "Recent Note", "recent-note", nil, "Some content.\n")

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "stderr: %s", reindexResult.Stderr)

	result := executeCommand("notes", "search", "--project", "personal", "--created-since", "1000h", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Hits []struct {
			NoteID  string  `json:"note_id"`
			Slug    string  `json:"slug"`
			Title   string  `json:"title"`
			Snippet string  `json:"snippet"`
			Path    string  `json:"path"`
			Score   float64 `json:"score"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Hits)
	require.Equal(t, "recent-note", got.Hits[0].Slug)
}

func TestNotesSearchCommandDebugHidesSensitiveFieldsByDefault(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "auth-migration.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth migration", "auth-migration", nil, "Search this body.\nObservation queryterm.\n")

	result := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	result = executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Hits []struct {
			Slug        string  `json:"slug"`
			Title       string  `json:"title"`
			Path        string  `json:"path,omitempty"`
			Score       float64 `json:"score,omitempty"`
			ContentHash string  `json:"content_hash,omitempty"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Hits)
	require.Equal(t, "auth-migration", got.Hits[0].Slug)
	require.Empty(t, got.Hits[0].Path)
	require.Zero(t, got.Hits[0].Score)
	require.Empty(t, got.Hits[0].ContentHash)
}

func TestNotesSearchCommandJSONContract(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "auth-migration.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth migration", "auth-migration", nil, "Search this body.\nObservation queryterm.\n")

	result := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	result = executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--json", "--debug")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Hits []struct {
			NoteID         string   `json:"note_id"`
			Slug           string   `json:"slug"`
			Title          string   `json:"title"`
			Snippet        string   `json:"snippet"`
			Summary        string   `json:"summary"`
			Tags           []string `json:"tags"`
			MatchedQueries []string `json:"matched_queries"`
			Path           string   `json:"path"`
			Score          *float64 `json:"score"`
			ContentHash    string   `json:"content_hash"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Hits)
	first := got.Hits[0]
	require.NotEmpty(t, first.NoteID)
	require.NotEmpty(t, first.Slug)
	require.NotEmpty(t, first.Title)
	require.NotEmpty(t, first.Snippet)
	require.NotNil(t, first.Score, "score should be present with debug")
	require.NotEmpty(t, first.Path, "path should be present with debug")
	require.NotEmpty(t, first.ContentHash, "content_hash should be present with debug")
}

func TestNotesSearchCommandDebugScorePresentAtZero(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	setLocalProjectMemoriesHome(t, projectRoot)
	require.NoError(t, writeLocalProjectFixture(t, projectRoot, "personal"))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "auth-migration.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth migration", "auth-migration", nil, "Search this body.\nObservation queryterm.\n")

	result := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	result = executeCommand("notes", "search", "--query", "auth", "--project", "personal", "--json", "--debug")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got struct {
		Hits []struct {
			Score *float64 `json:"score"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotEmpty(t, got.Hits)
	require.NotNil(t, got.Hits[0].Score, "score key must be present when debug is enabled")
}
