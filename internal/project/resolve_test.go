package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestResolveProjectFileBased(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	// Create a central project
	slug := "demo"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = NowUTC()
	manifest.UpdatedAt = NowUTC()
	manifest.Generator.App = "mnemonic"
	require.NoError(t, WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	projectID, err := ResolveProject(memoriesHome, slug)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", projectID)
}

func TestResolveProjectNotFound(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()

	_, err := ResolveProject(memoriesHome, "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), `project "missing" not found`)
}

func TestMemoriesHome(t *testing.T) {
	testutil.CleanEnvForTest(t)
	tempDir := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", tempDir)

	home, err := MemoriesHome()
	require.NoError(t, err)
	require.Equal(t, tempDir, home)
}
