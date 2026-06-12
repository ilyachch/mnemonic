package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestInitCommandRequiresName(t *testing.T) {
	result := executeCommand("init")
	require.Error(t, result.Err, "expected init without NAME to fail")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestInitCommandRejectsMultipleNames(t *testing.T) {
	result := executeCommand("init", "one", "two")
	require.Error(t, result.Err, "expected init with multiple NAME args to fail")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestInitCommandRejectsLocalAndDetachedTogether(t *testing.T) {
	result := executeCommand("init", "demo", "--local", "--detached")
	require.Error(t, result.Err, "expected init with incompatible flags to fail")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}

func TestInitCommandLocalCreatesLocalProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	result := executeCommand("init", "backend", "--local")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	projectPath := filepath.Join(cwd, ".mnemonic")
	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")

	_, err = os.Stat(projectPath)
	require.NoError(t, err, "project file missing")
	_, err = os.Stat(localPath)
	require.NoError(t, err, "local memories directory missing")
	_, err = os.Stat(manifestPath)
	require.True(t, os.IsNotExist(err), "unexpected detached-style manifest")

	projectData, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	parsedProject, err := project.ParseMnemonicFile(projectData)
	require.NoError(t, err)
	require.Len(t, parsedProject.Projects, 1)
	require.Equal(t, project.ProjectKindLocal, parsedProject.Projects[0].Kind)

	indexPath, err := index.Path(parsedProject.Projects[0].ID)
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestInitCommandDetachedCreatesDetachedProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	result := executeCommand("init", "personal", "--detached")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "personal", "mnemonic.toml")

	_, err = os.Stat(projectPath)
	require.True(t, os.IsNotExist(err), ".mnemonic exists or stat failed unexpectedly")

	_, err = os.Stat(manifestPath)
	require.NoError(t, err, "manifest file missing")

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := project.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, project.ManifestKindDetached, parsedManifest.Kind)

	indexPath, err := index.Path(parsedManifest.ProjectID)
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestInitCommandRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	first := executeCommand("init", "backend", "--local")
	require.NoError(t, first.Err, "stderr: %s", first.Stderr)

	result := executeCommand("init", "Backend", "--local")
	require.Error(t, result.Err, "second init error = nil, want duplicate slug rejection")
	require.Equal(t, 4, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, `project slug "backend" already exists`)
}

func projectClockForCLI() project.Clock {
	return projectClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440000"},
	}
}

type projectClock struct {
	now   time.Time
	uuids []string
}

func (c projectClock) Now() time.Time {
	return c.now
}

func (c projectClock) UUID() string {
	if len(c.uuids) == 0 {
		return ""
	}
	return c.uuids[0]
}
