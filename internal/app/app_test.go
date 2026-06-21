package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestIsNoProjectSelected(t *testing.T) {
	require.False(t, IsNoProjectSelected(nil))
	require.False(t, IsNoProjectSelected(errors.New("other")))
}

func TestNewBuildsConfigPathsRegistryAndServices(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)
	dataHome := filepath.Join(projectRoot, "data")
	stateHome := filepath.Join(projectRoot, "state")
	configHome := filepath.Join(projectRoot, "config")
	cacheHome := filepath.Join(projectRoot, "cache")
	memoriesHome := filepath.Join(projectRoot, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	container, err := New(Input{})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Close()
	})

	require.Empty(t, container.Config.Paths.MemoriesHome)
	require.Equal(t, filepath.Clean(dataHome), container.Paths.DataHome)
	require.Equal(t, filepath.Clean(stateHome), container.Paths.StateHome)
	require.Equal(t, filepath.Clean(configHome), container.Paths.ConfigHome)
	require.Equal(t, filepath.Clean(cacheHome), container.Paths.CacheHome)
	require.Equal(t, filepath.Clean(memoriesHome), container.Paths.MemoriesHome)
	require.NotNil(t, container.Services.Catalog)
	require.NotNil(t, container.Services.Maint)
	require.NotNil(t, container.Services.Maint.Catalog)
	require.NotNil(t, container.Services.Maint.RuntimeFactory)
	require.NotNil(t, container.Services.ProjectResolver)

	runtime, err := container.Services.Maint.RuntimeFactory(context.Background(), kb.KnowledgeBase{ID: "550e8400-e29b-41d4-a716-446655440999"})
	require.NoError(t, err)
	require.NotNil(t, runtime)
	require.NotNil(t, runtime.IndexService())
}

func TestMemoriesPathForEntryUsesRepoRelativePathForLocalProjects(t *testing.T) {
	entry := registry.Entry{
		Type:        "local",
		RepoRootAbs: "/abs/path/to/repo",
		MemoriesAbs: "/abs/path/to/repo/.mnemonic-memories/personal",
	}

	require.Equal(t, ".mnemonic-memories/personal", memoriesPathForEntry(entry))
}

func TestMemoriesPathForEntryFallsBackToLeafName(t *testing.T) {
	entry := registry.Entry{
		Type:        "central",
		MemoriesAbs: "/abs/path/to/memories/personal",
	}

	require.Equal(t, "personal", memoriesPathForEntry(entry))
}

func TestFileResolverResolve(t *testing.T) {
	testutil.CleanEnvForTest(t)

	t.Run("missing selector", func(t *testing.T) {
		resolver := &FileResolver{MemoriesHome: t.TempDir()}

		resolution, err := resolver.Resolve(ProjectResolveInput{})
		require.Error(t, err)
		require.Empty(t, resolution)
		require.True(t, IsNoProjectSelected(err))
	})

	t.Run("not found", func(t *testing.T) {
		resolver := &FileResolver{MemoriesHome: t.TempDir()}

		_, err := resolver.Resolve(ProjectResolveInput{ProjectSelector: "missing"})
		require.Error(t, err)
		require.Equal(t, apperr.CodeNotFound, err.(*apperr.Error).Code)
		require.Contains(t, err.Error(), `project "missing" not found`)
	})

	t.Run("central project", func(t *testing.T) {
		memoriesHome := t.TempDir()
		slug := "demo"
		projectDir := filepath.Join(memoriesHome, slug)
		require.NoError(t, os.MkdirAll(projectDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "mnemonic.toml"), []byte("project_id = \"550e8400-e29b-41d4-a716-446655440000\"\nname = \"demo\"\nslug = \"demo\"\n"), 0o644))

		origManifestParser := registry.DefaultManifestParser
		origPointerParser := registry.DefaultPointerParser
		registry.DefaultManifestParser = nil
		registry.DefaultPointerParser = nil
		t.Cleanup(func() {
			registry.DefaultManifestParser = origManifestParser
			registry.DefaultPointerParser = origPointerParser
		})

		resolver := &FileResolver{MemoriesHome: memoriesHome}
		resolution, err := resolver.Resolve(ProjectResolveInput{ProjectSelector: slug})
		require.NoError(t, err)
		require.Equal(t, projectDir, resolution.RepoRootAbs)
		require.Equal(t, filepath.Join(projectDir, "mnemonic.toml"), resolution.ManifestAbs)
		require.Equal(t, slug, resolution.Project.Slug)
		require.Equal(t, "demo", resolution.Project.MemoriesPath)
	})
}

func TestWrapRegistryErrorNil(t *testing.T) {
	require.NoError(t, wrapRegistryError(nil))
}
