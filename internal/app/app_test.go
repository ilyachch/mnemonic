package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

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
	require.NotNil(t, container.Services.Registry)
}