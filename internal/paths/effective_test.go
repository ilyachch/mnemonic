package paths

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestResolveEffectivePaths(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		setEffectiveBaseEnv(t)

		got, err := ResolveEffectivePaths(EffectiveInput{})
		require.NoError(t, err)

		wantHome := "/home/alice"
		want := EffectivePaths{
			ConfigHome:   filepath.Join(wantHome, ".config"),
			DataHome:     filepath.Join(wantHome, ".local", "share"),
			StateHome:    filepath.Join(wantHome, ".local", "state"),
			CacheHome:    filepath.Join(wantHome, ".cache"),
			MemoriesHome: filepath.Join(wantHome, ".mnemonic"),
		}
		require.Empty(t, cmp.Diff(want, got))
	})

	t.Run("xdg env beats defaults", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		xdgData := t.TempDir()
		xdgState := t.TempDir()
		xdgCache := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)
		t.Setenv("XDG_DATA_HOME", xdgData)
		t.Setenv("XDG_STATE_HOME", xdgState)
		t.Setenv("XDG_CACHE_HOME", xdgCache)

		got, err := ResolveEffectivePaths(EffectiveInput{})
		require.NoError(t, err)

		want := EffectivePaths{
			ConfigHome:   xdgConfig,
			DataHome:     xdgData,
			StateHome:    xdgState,
			CacheHome:    xdgCache,
			MemoriesHome: filepath.Join("/home/alice", ".mnemonic"),
		}
		require.Empty(t, cmp.Diff(want, got))
	})

	t.Run("config.toml beats xdg env for memories home", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)

		got, err := ResolveEffectivePaths(EffectiveInput{
			RawConfigMemoriesHome: "~/.custom-memories",
		})
		require.NoError(t, err)

		want := EffectivePaths{
			ConfigHome:            xdgConfig,
			DataHome:              filepath.Join("/home/alice", ".local", "share"),
			StateHome:             filepath.Join("/home/alice", ".local", "state"),
			CacheHome:             filepath.Join("/home/alice", ".cache"),
			MemoriesHome:          filepath.Join("/home/alice", ".custom-memories"),
			RawConfigMemoriesHome: "~/.custom-memories",
		}
		require.Empty(t, cmp.Diff(want, got))
	})

	t.Run("mnemonic env beats config.toml", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)
		t.Setenv("MNEMONIC_MEMORIES_HOME", "/tmp/mnemonic-memories")

		got, err := ResolveEffectivePaths(EffectiveInput{
			RawConfigMemoriesHome: "~/.custom-memories",
		})
		require.NoError(t, err)

		want := EffectivePaths{
			ConfigHome:   xdgConfig,
			DataHome:     filepath.Join("/home/alice", ".local", "share"),
			StateHome:    filepath.Join("/home/alice", ".local", "state"),
			CacheHome:    filepath.Join("/home/alice", ".cache"),
			MemoriesHome: "/tmp/mnemonic-memories",
		}
		require.Empty(t, cmp.Diff(want, got))
	})

	t.Run("cli flags beat mnemonic env and config.toml", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)
		t.Setenv("MNEMONIC_MEMORIES_HOME", "/tmp/mnemonic-memories")

		got, err := ResolveEffectivePaths(EffectiveInput{
			CLI: CLIOverrides{
				ConfigFile:   "cli-config.toml",
				ConfigHome:   "cli-config-home",
				DataHome:     "cli-data-home",
				StateHome:    "cli-state-home",
				CacheHome:    "cli-cache-home",
				MemoriesHome: "cli-memories-home",
			},
			ConfigFile:            "fallback-config.toml",
			RawConfigMemoriesHome: "~/.custom-memories",
		})
		require.NoError(t, err)

		cliConfigFile, err := filepath.Abs("cli-config.toml")
		require.NoError(t, err)
		cliConfigHome, err := filepath.Abs("cli-config-home")
		require.NoError(t, err)
		cliDataHome, err := filepath.Abs("cli-data-home")
		require.NoError(t, err)
		cliStateHome, err := filepath.Abs("cli-state-home")
		require.NoError(t, err)
		cliCacheHome, err := filepath.Abs("cli-cache-home")
		require.NoError(t, err)
		cliMemoriesHome, err := filepath.Abs("cli-memories-home")
		require.NoError(t, err)

		want := EffectivePaths{
			ConfigHome:   cliConfigHome,
			DataHome:     cliDataHome,
			StateHome:    cliStateHome,
			CacheHome:    cliCacheHome,
			MemoriesHome: cliMemoriesHome,
			ConfigFile:   cliConfigFile,
		}
		require.Empty(t, cmp.Diff(want, got))
	})

	t.Run("config file path is normalized when discovered path is relative", func(t *testing.T) {
		setEffectiveBaseEnv(t)

		got, err := ResolveEffectivePaths(EffectiveInput{
			ConfigFile: "relative/config.toml",
		})
		require.NoError(t, err)

		wantConfigFile, err := filepath.Abs("relative/config.toml")
		require.NoError(t, err)

		want := EffectivePaths{
			ConfigHome:   filepath.Join("/home/alice", ".config"),
			DataHome:     filepath.Join("/home/alice", ".local", "share"),
			StateHome:    filepath.Join("/home/alice", ".local", "state"),
			CacheHome:    filepath.Join("/home/alice", ".cache"),
			MemoriesHome: filepath.Join("/home/alice", ".mnemonic"),
			ConfigFile:   wantConfigFile,
		}
		require.Empty(t, cmp.Diff(want, got))
	})
}

func setEffectiveBaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", "/home/alice")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("MNEMONIC_CONFIG_HOME", "")
	t.Setenv("MNEMONIC_DATA_HOME", "")
	t.Setenv("MNEMONIC_STATE_HOME", "")
	t.Setenv("MNEMONIC_CACHE_HOME", "")
	t.Setenv("MNEMONIC_MEMORIES_HOME", "")
}