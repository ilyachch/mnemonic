package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesBacklinksCommandReturnsLinks(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	require.NoError(t, project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	_, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Target Note",
		Body:    []byte("target body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	})
	require.NoError(t, err)
	_, err = notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Source Note",
		Body:    []byte("[[Target Note]]\n## Relations\n- depends_on [[Target Note]]\n- relates_to [[Missing Note]]\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440002"
		},
	})
	require.NoError(t, err)

	_, err = index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot)
	require.NoError(t, err)

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
	require.Len(t, got.Links, 2)
	foundPlain := false
	foundRelation := false
	for _, link := range got.Links {
		require.Equal(t, "550e8400-e29b-41d4-a716-446655440002", link.NoteID)
		require.Positive(t, link.SourceLine)
		switch link.RelationType {
		case "":
			foundPlain = true
		case "depends_on":
			foundRelation = true
		default:
			t.Fatalf("unexpected relation_type = %q", link.RelationType)
		}
	}
	require.True(t, foundPlain, "missing plain backlink")
	require.True(t, foundRelation, "missing relation backlink")
}

func TestNotesBacklinksCommandMissingNoteAndMissingIndex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	require.NoError(t, project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}))

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "backlinks", "missing-note", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	_, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Target Note",
		Body:    []byte("target body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	})
	require.NoError(t, err)

	result = executeCommand("notes", "backlinks", "target-note", "--project", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 3, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, "mnemonic project reindex")
}