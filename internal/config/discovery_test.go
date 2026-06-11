package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestDiscoverConfigFile(t *testing.T) {
	t.Run("missing config uses default config", func(t *testing.T) {
		clearConfigDiscoveryEnv(t)

		got, err := DiscoverConfigFile("")
		require.NoError(t, err)
		require.Equal(t, "", got)
	})

	t.Run("precedence order", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		homeConfigPath := mustConfigFile(t, filepath.Join(homeDir, ".config", "mnemonic", configFileName))

		t.Run("level 5 home default", func(t *testing.T) {
			clearDiscoverySelectionEnv(t)

			got, err := DiscoverConfigFile("")
			require.NoError(t, err)
			require.Equal(t, homeConfigPath, got)
		})

		t.Run("level 4 xdg config home beats home default", func(t *testing.T) {
			clearDiscoverySelectionEnv(t)
			xdgRoot := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgRoot)
			xdgPath := mustConfigFile(t, filepath.Join(xdgRoot, "mnemonic", configFileName))

			got, err := DiscoverConfigFile("")
			require.NoError(t, err)
			require.Equal(t, xdgPath, got)
		})

		t.Run("level 3 mnemonic config home beats xdg", func(t *testing.T) {
			clearDiscoverySelectionEnv(t)
			mnemonicRoot := t.TempDir()
			xdgRoot := t.TempDir()
			t.Setenv("MNEMONIC_CONFIG_HOME", mnemonicRoot)
			t.Setenv("XDG_CONFIG_HOME", xdgRoot)
			mnemonicPath := mustConfigFile(t, filepath.Join(mnemonicRoot, "mnemonic", configFileName))
			mustConfigFile(t, filepath.Join(xdgRoot, "mnemonic", configFileName))

			got, err := DiscoverConfigFile("")
			require.NoError(t, err)
			require.Equal(t, mnemonicPath, got)
		})

		t.Run("level 2 mnemonic config file beats config home", func(t *testing.T) {
			clearDiscoverySelectionEnv(t)
			envPath := filepath.Join(t.TempDir(), "env-config.toml")
			mnemonicRoot := t.TempDir()
			xdgRoot := t.TempDir()
			t.Setenv("MNEMONIC_CONFIG_FILE", envPath)
			t.Setenv("MNEMONIC_CONFIG_HOME", mnemonicRoot)
			t.Setenv("XDG_CONFIG_HOME", xdgRoot)
			envPath = mustConfigFile(t, envPath)
			mustConfigFile(t, filepath.Join(mnemonicRoot, "mnemonic", configFileName))
			mustConfigFile(t, filepath.Join(xdgRoot, "mnemonic", configFileName))

			got, err := DiscoverConfigFile("")
			require.NoError(t, err)
			require.Equal(t, envPath, got)
		})

		t.Run("level 1 explicit config beats env", func(t *testing.T) {
			clearDiscoverySelectionEnv(t)
			explicitPath := mustConfigFile(t, filepath.Join(t.TempDir(), "explicit-config.toml"))
			envPath := mustConfigFile(t, filepath.Join(t.TempDir(), "env-config.toml"))
			mnemonicRoot := t.TempDir()
			xdgRoot := t.TempDir()
			t.Setenv("MNEMONIC_CONFIG_FILE", envPath)
			t.Setenv("MNEMONIC_CONFIG_HOME", mnemonicRoot)
			t.Setenv("XDG_CONFIG_HOME", xdgRoot)
			mustConfigFile(t, filepath.Join(mnemonicRoot, "mnemonic", configFileName))
			mustConfigFile(t, filepath.Join(xdgRoot, "mnemonic", configFileName))

			got, err := DiscoverConfigFile(explicitPath)
			require.NoError(t, err)
			require.Equal(t, explicitPath, got)
		})
	})
}

func clearConfigDiscoveryEnv(t *testing.T) {
	t.Helper()

	testutil.CleanEnvForTest(t)
	clearDiscoverySelectionEnv(t)
}

func clearDiscoverySelectionEnv(t *testing.T) {
	t.Helper()

	t.Setenv("MNEMONIC_CONFIG_FILE", "")
	t.Setenv("MNEMONIC_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
}

func mustConfigFile(t *testing.T, path string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("version = 1\n"), 0o644))

	return path
}