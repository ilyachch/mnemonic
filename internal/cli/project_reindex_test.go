package cli

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectReindexBatchModesDoNotHitRegistryBusy(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	))
	defer restoreClock()

	for _, name := range []string{"Personal", "Work"} {
		err := project.InitProject(project.InitInput{
			CWD:          projectRoot,
			MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
			Name:         name,
			Mode:         project.InitModeLocal,
		})
		require.NoError(t, err, "InitProject(%q) error = %v", name, err)
	}

	personal, err := queryProjectBySelector("personal")
	require.NoError(t, err)
	work, err := queryProjectBySelector("work")
	require.NoError(t, err)

	for _, tc := range []struct {
		rootDir string
		title   string
		uuid    string
	}{
		{rootDir: personal.Location.memoriesAbs, title: "Personal Note", uuid: "550e8400-e29b-41d4-a716-446655440010"},
		{rootDir: work.Location.memoriesAbs, title: "Work Note", uuid: "550e8400-e29b-41d4-a716-446655440011"},
	} {
		_, err := notes.Create(notes.CreateInput{
			RootDir: tc.rootDir,
			Title:   tc.title,
			Body:    []byte("body\n"),
			UUID: func(id string) func() string {
				return func() string { return id }
			}(tc.uuid),
		})
		require.NoError(t, err, "Create(%q) error = %v", tc.title, err)
	}

	container, err := mustAppContainer()
	require.NoError(t, err)

	allSummary, err := reindexAllProjects(container.Services.Registry, container.Paths)
	require.NoError(t, err)
	require.Equal(t, 2, allSummary.Indexed)

	_, err = container.Services.Registry.Exec(`UPDATE project_status SET needs_reindex = 1 WHERE project_id = ?`, personal.ProjectID)
	require.NoError(t, err, "mark personal needs_reindex")

	pendingSummary, err := reindexPendingProjects(container.Services.Registry, container.Paths)
	require.NoError(t, err)
	require.Equal(t, 1, pendingSummary.Indexed)
	require.Equal(t, 1, pendingSummary.Skipped)
}

func TestProjectReindexRebuildsIncompatibleIndexWithoutChangingMarkdown(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	err := project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	})
	require.NoError(t, err)

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	noteResult, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Healthy Note",
		Body:    []byte("body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	})
	require.NoError(t, err)
	_, err = index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot)
	require.NoError(t, err)

	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	db, err := sql.Open("sqlite", indexPath)
	require.NoError(t, err)
	_, err = db.Exec(`PRAGMA user_version = 999`)
	if err != nil {
		_ = db.Close()
		require.NoError(t, err, "set user_version=999")
	}
	err = db.Close()
	require.NoError(t, err)

	notePath := filepath.Join(memoriesRoot, filepath.FromSlash(noteResult.Path))
	beforeInfo, err := os.Stat(notePath)
	require.NoError(t, err)
	beforeContent, err := os.ReadFile(notePath)
	require.NoError(t, err)

	doctorResult := executeCommand("project", "doctor", "personal", "--json")
	require.NoError(t, doctorResult.Err, "project doctor returned error\nstderr: %s", doctorResult.Stderr)
	require.Contains(t, doctorResult.Stdout, `"status": "needs_reindex"`)
	require.Contains(t, doctorResult.Stdout, `"name": "index schema"`)

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	require.NoError(t, reindexResult.Err, "project reindex returned error\nstderr: %s", reindexResult.Stderr)

	afterInfo, err := os.Stat(notePath)
	require.NoError(t, err)
	require.True(t, afterInfo.ModTime().Equal(beforeInfo.ModTime()), "markdown mtime changed: before=%s after=%s", beforeInfo.ModTime(), afterInfo.ModTime())
	afterContent, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, string(beforeContent), string(afterContent), "markdown content changed")

	checkDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	require.NoError(t, err)
	defer func() { _ = checkDB.Close() }()
	status, err := index.CheckSchemaStatus(checkDB)
	require.NoError(t, err)
	require.Equal(t, index.SchemaStatusOK, status)
}
