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

func TestProjectReindexSingleProject(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	// Create a central project in memories home
	slug := "personal"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	writeTaggedNote(t, filepath.Join(projectDir, "my-note.md"), "550e8400-e29b-41d4-a716-446655440010", "My Note", "my-note", nil, "body\n")

	result := executeCommand("project", "reindex", "personal")
	require.NoError(t, result.Err, "project reindex returned error\nstderr: %s", result.Stderr)

	// Verify index was created
	idxPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(idxPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectReindexUsesEnvironmentSelector(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_PROJECT", "personal")

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	projectDir := filepath.Join(memoriesHome, "personal")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "personal"
	manifest.Slug = "personal"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	writeTaggedNote(t, filepath.Join(projectDir, "my-note.md"), "550e8400-e29b-41d4-a716-446655440010", "My Note", "my-note", nil, "body\n")

	result := executeCommand("project", "reindex", "--json")
	require.NoError(t, result.Err, "project reindex returned error\nstderr: %s", result.Stderr)

	idxPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(idxPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectReindexPositionalProjectWinsOverFlag(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	projectDir := filepath.Join(memoriesHome, "personal")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "personal"
	manifest.Slug = "personal"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	writeTaggedNote(t, filepath.Join(projectDir, "my-note.md"), "550e8400-e29b-41d4-a716-446655440010", "My Note", "my-note", nil, "body\n")

	result := executeCommand("project", "reindex", "personal", "--project", "work", "--json")
	require.NoError(t, result.Err, "project reindex returned error\nstderr: %s", result.Stderr)

	idxPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(idxPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectReindexAllProjects(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_PROJECT", "missing")

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	))
	defer restoreClock()

	for _, item := range []struct {
		slug      string
		projectID string
	}{
		{slug: "personal", projectID: "550e8400-e29b-41d4-a716-446655440000"},
		{slug: "work", projectID: "550e8400-e29b-41d4-a716-446655440001"},
	} {
		projectDir := filepath.Join(memoriesHome, item.slug)
		require.NoError(t, os.MkdirAll(projectDir, 0o755))
		manifest := project.NewMnemonicManifest()
		manifest.ProjectID = item.projectID
		manifest.Name = item.slug
		manifest.Slug = item.slug
		manifest.MarkdownFormatVersion = 1
		manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
		manifest.UpdatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
		manifest.Generator.App = "mnemonic"
		require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))
		writeTaggedNote(t, filepath.Join(projectDir, "note.md"), item.projectID, "Note", "note", nil, "body\n")
	}

	result := executeCommand("project", "reindex", "--all")
	require.NoError(t, result.Err, "project reindex --all returned error\nstderr: %s", result.Stderr)

	for _, projectID := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	} {
		idxPath, err := index.Path(projectID)
		require.NoError(t, err)
		_, err = os.Stat(idxPath)
		require.NoError(t, err, "index file missing")
	}
}

func TestProjectReindexRequiresSelector(t *testing.T) {
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", t.TempDir())

	result := executeCommand("project", "reindex", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 2, ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "project selector is required")
}

func TestProjectReindexAllRejectsExplicitSelector(t *testing.T) {
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", t.TempDir())

	result := executeCommand("project", "reindex", "--all", "personal", "--json")
	require.Error(t, result.Err)
	require.Equal(t, 2, ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "--all cannot be combined with a project selector")
}

func TestProjectReindexRebuildsIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	slug := "personal"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	writeTaggedNote(t, filepath.Join(projectDir, "healthy-note.md"), "550e8400-e29b-41d4-a716-446655440001", "Healthy Note", "healthy-note", nil, "body\n")

	// Initial index
	result := executeCommand("project", "reindex", "personal")
	require.NoError(t, result.Err, "initial reindex returned error\nstderr: %s", result.Stderr)

	// Reindex again
	result2 := executeCommand("project", "reindex", "personal")
	require.NoError(t, result2.Err, "second reindex returned error\nstderr: %s", result2.Stderr)

	idxPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(idxPath)
	require.NoError(t, err, "index file missing")
}
