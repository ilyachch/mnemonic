package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
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

	restore := project.SetClock(projectClockForCLI("550e8400-e29b-41d4-a716-446655440000"))
	t.Cleanup(restore)

	result := executeCommand("init", "backend", "--local")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	projectPath := filepath.Join(cwd, ".mnemonic")
	_, err = os.Stat(projectPath)
	require.True(t, os.IsNotExist(err), ".mnemonic anchor file should not be created")

	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	_, err = os.Stat(localPath)
	require.NoError(t, err, "local memories directory missing")

	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")
	_, err = os.Stat(manifestPath)
	require.True(t, os.IsNotExist(err), "unexpected central manifest")

	// Init also creates an index; resolve the project UUID by re-listing the registry.
	projects := listRegistryProjects(t)
	require.NotEmpty(t, projects)
	projectID := projects[0].projectID
	require.Equal(t, "backend", projects[0].slug)

	indexPath, err := index.Path(projectID)
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestInitCommandCentralCreatesCentralProject(t *testing.T) {
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

	restore := project.SetClock(projectClockForCLI("550e8400-e29b-41d4-a716-446655440000"))
	t.Cleanup(restore)

	result := executeCommand("init", "personal")
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
	require.Equal(t, project.ManifestType(""), parsedManifest.Type)

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

	restore := project.SetClock(projectClockForCLI("550e8400-e29b-41d4-a716-446655440000"))
	t.Cleanup(restore)

	first := executeCommand("init", "backend", "--local")
	require.NoError(t, first.Err, "stderr: %s", first.Stderr)

	restore()
	restore = project.SetClock(projectClockForCLI("7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1"))
	t.Cleanup(restore)

	result := executeCommand("init", "Backend", "--local")
	require.Error(t, result.Err, "second init error = nil, want duplicate slug rejection")
	require.Equal(t, 4, ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, `project slug "backend" already exists`)
}

func TestInitCommandCentralWithDescription(t *testing.T) {
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

	restore := project.SetClock(projectClockForCLI("550e8400-e29b-41d4-a716-446655440000"))
	t.Cleanup(restore)

	desc := "Backend architecture decisions, API contracts, and database schemas."
	result := executeCommand("init", "backend", "--description", desc)
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")
	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := project.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, desc, parsedManifest.Description)
}

func projectClockForCLI(uuids ...string) project.Clock {
	return projectClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: uuids,
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

type registryProjectRow struct {
	projectID string
	slug      string
}

func listRegistryProjects(t *testing.T) []registryProjectRow {
	t.Helper()

	container, err := mustAppContainer()
	require.NoError(t, err)
	t.Cleanup(closeAppContainer)

	entries, _, err := registry.Scan(container.Paths.MemoriesHome)
	require.NoError(t, err)

	var out []registryProjectRow
	for _, e := range entries {
		out = append(out, registryProjectRow{
			projectID: e.ProjectID,
			slug:      e.Slug,
		})
	}

	return out
}
