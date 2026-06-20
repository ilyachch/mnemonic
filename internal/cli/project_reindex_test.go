package cli

import (
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

	// Create notes
	_, err := notes.Create(notes.CreateInput{
		RootDir: projectDir,
		Title:   "My Note",
		Body:    []byte("body\n"),
		UUID:    func() string { return "550e8400-e29b-41d4-a716-446655440010" },
	})
	require.NoError(t, err)

	result := executeCommand("project", "reindex", "personal")
	require.NoError(t, result.Err, "project reindex returned error\nstderr: %s", result.Stderr)

	// Verify index was created
	idxPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(idxPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectReindexAllProjects(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	))
	defer restoreClock()

	for _, slug := range []string{"personal", "work"} {
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
	}

	result := executeCommand("project", "reindex", "--all")
	require.NoError(t, result.Err, "project reindex --all returned error\nstderr: %s", result.Stderr)
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

	_, err := notes.Create(notes.CreateInput{
		RootDir: projectDir,
		Title:   "Healthy Note",
		Body:    []byte("body\n"),
		UUID:    func() string { return "550e8400-e29b-41d4-a716-446655440001" },
	})
	require.NoError(t, err)

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
