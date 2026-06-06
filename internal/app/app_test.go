package app

import (
	"path/filepath"
	"testing"

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
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close()
	})

	if container.Config.Paths.MemoriesHome != "" {
		t.Fatalf("config memories_home = %q, want empty default", container.Config.Paths.MemoriesHome)
	}
	if container.Paths.DataHome != filepath.Clean(dataHome) {
		t.Fatalf("data_home = %q, want %q", container.Paths.DataHome, filepath.Clean(dataHome))
	}
	if container.Paths.StateHome != filepath.Clean(stateHome) {
		t.Fatalf("state_home = %q, want %q", container.Paths.StateHome, filepath.Clean(stateHome))
	}
	if container.Paths.ConfigHome != filepath.Clean(configHome) {
		t.Fatalf("config_home = %q, want %q", container.Paths.ConfigHome, filepath.Clean(configHome))
	}
	if container.Paths.CacheHome != filepath.Clean(cacheHome) {
		t.Fatalf("cache_home = %q, want %q", container.Paths.CacheHome, filepath.Clean(cacheHome))
	}
	if container.Paths.MemoriesHome != filepath.Clean(memoriesHome) {
		t.Fatalf("memories_home = %q, want %q", container.Paths.MemoriesHome, filepath.Clean(memoriesHome))
	}
	if container.Services.Registry == nil {
		t.Fatal("registry service is nil")
	}
}
