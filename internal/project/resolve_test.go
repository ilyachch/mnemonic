package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/registry"
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

func TestErrProjectNotFoundError(t *testing.T) {
	require.Equal(t, `project "missing" not found`, ErrProjectNotFound{Selector: "missing"}.Error())
}

func TestResolveProjectCentralManifestFallback(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
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
	require.NoError(t, WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	origManifestParser := registry.DefaultManifestParser
	origPointerParser := registry.DefaultPointerParser
	registry.DefaultManifestParser = nil
	registry.DefaultPointerParser = nil
	t.Cleanup(func() {
		registry.DefaultManifestParser = origManifestParser
		registry.DefaultPointerParser = origPointerParser
	})

	projectID, err := ResolveProject(memoriesHome, slug)
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, projectID)
}

func TestResolveProjectLocalPointerFallback(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	workspaceRoot := t.TempDir()
	slug := "personal"
	manifestPath := filepath.Join(workspaceRoot, slug, "mnemonic.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0o755))

	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = NowUTC()
	manifest.UpdatedAt = NowUTC()
	require.NoError(t, WriteMnemonicManifest(manifestPath, manifest))
	require.NoError(t, WritePointerFile(filepath.Join(memoriesHome, slug+".toml"), &PointerFile{ManifestPath: manifestPath}))

	origManifestParser := registry.DefaultManifestParser
	origPointerParser := registry.DefaultPointerParser
	registry.DefaultManifestParser = nil
	registry.DefaultPointerParser = nil
	t.Cleanup(func() {
		registry.DefaultManifestParser = origManifestParser
		registry.DefaultPointerParser = origPointerParser
	})

	projectID, err := ResolveProject(memoriesHome, slug)
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, projectID)
}

func TestMemoriesHome(t *testing.T) {
	testutil.CleanEnvForTest(t)
	tempDir := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", tempDir)

	home, err := MemoriesHome()
	require.NoError(t, err)
	require.Equal(t, tempDir, home)
}
