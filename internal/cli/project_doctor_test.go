package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectDoctorReturnsOkForHealthyProject(t *testing.T) {
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
	_, err = notes.Create(notes.CreateInput{
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
		Body:    []byte("[[Target Note]]\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440002"
		},
	})
	require.NoError(t, err)
	_, err = index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot)
	require.NoError(t, err)

	before := mustNoteSnapshot(t, memoriesRoot)
	result := executeCommand("project", "doctor", "personal", "--json")
	require.NoError(t, result.Err, "project doctor returned error\nstderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, `"status": "ok"`)
	after := mustNoteSnapshot(t, memoriesRoot)
	require.Equal(t, before, after, "markdown snapshot changed")
}

func TestProjectDoctorMissingIndexReturnsNeedsReindex(t *testing.T) {
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

	result := executeCommand("project", "doctor", "personal", "--json")
	require.NoError(t, result.Err, "project doctor returned error\nstderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, `"status": "needs_reindex"`)
}

func TestProjectDoctorCorruptedIndexReturnsExitSix(t *testing.T) {
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

	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Dir(indexPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(indexPath, []byte("broken"), 0o644)
	require.NoError(t, err)

	result := executeCommand("project", "doctor", "personal", "--json")
	require.Error(t, result.Err, "project doctor error = nil, want corrupted index")
	require.Equal(t, 6, ExitCodeForError(result.Err))
}

func TestProjectDoctorReportsStaleTempFileWithoutDeletingIt(t *testing.T) {
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
	_, err = notes.Create(notes.CreateInput{
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

	tempPath := filepath.Join(projectRoot, ".tmp-test")
	err = os.WriteFile(tempPath, []byte("stale\n"), 0o644)
	require.NoError(t, err)

	result := executeCommand("project", "doctor", "personal", "--json")
	require.NoError(t, result.Err, "project doctor returned error\nstderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, `"status": "warning"`)
	require.Contains(t, result.Stdout, `"name": "stale temp files"`)
	require.Contains(t, result.Stdout, `"count": 1`)
	require.Contains(t, result.Stdout, tempPath)
	_, err = os.Stat(tempPath)
	require.NoError(t, err, "Stat(%q) error = %v", tempPath, err)
}

func mustNoteSnapshot(t *testing.T, root string) string {
	t.Helper()
	paths, err := notes.Walk(root)
	require.NoError(t, err)
	var b strings.Builder
	for _, rel := range paths {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		require.NoError(t, err, "ReadFile(%q) error = %v", rel, err)
		info, err := os.Stat(abs)
		require.NoError(t, err, "Stat(%q) error = %v", rel, err)
		b.WriteString(rel)
		b.WriteByte('\n')
		b.WriteString(info.ModTime().UTC().Format(time.RFC3339Nano))
		b.WriteByte('\n')
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}
