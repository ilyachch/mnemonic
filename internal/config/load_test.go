package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid toml", func(t *testing.T) {
		path := writeConfigFile(t, `version = 1

[paths]
memories_home = "/tmp/mnemonic"

[notes]
delete_behavior = "delete"
trash_dir_name = ".deleted"

[index]
fts = false
wal = false
busy_timeout_ms = 2500

[output]
json_pretty = true

[logging]
level = "debug"
`)

		cfg, err := LoadConfig(path)
		require.NoError(t, err)

		require.Equal(t, 1, cfg.Version)
		require.Equal(t, "/tmp/mnemonic", cfg.Paths.MemoriesHome)
		require.Equal(t, "delete", cfg.Notes.DeleteBehavior)
		require.Equal(t, ".deleted", cfg.Notes.TrashDirName)
		require.False(t, cfg.Index.FTS)
		require.False(t, cfg.Index.WAL)
		require.Equal(t, 2500, cfg.Index.BusyTimeoutMs)
		require.True(t, cfg.Output.JSONPretty)
		require.Equal(t, "debug", cfg.Logging.Level)
	})

	t.Run("requires version one when file exists", func(t *testing.T) {
		path := writeConfigFile(t, `
[notes]
delete_behavior = "delete"
`)

		_, err := LoadConfig(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "version = 1")
	})

	t.Run("rejects unsupported version with clear error", func(t *testing.T) {
		path := writeConfigFile(t, `version = 999
`)

		_, err := LoadConfig(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported config version 999")
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		path := writeConfigFile(t, `version = 1
unknown = "value"
`)

		_, err := LoadConfig(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown key")
	})

	t.Run("returns clear error for invalid toml", func(t *testing.T) {
		path := writeConfigFile(t, `version = 1
[notes
delete_behavior = "delete"
`)

		_, err := LoadConfig(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "config syntax error")
	})
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))

	return path
}
