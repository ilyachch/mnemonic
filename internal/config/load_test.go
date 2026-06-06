package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		if err != nil {
			t.Fatalf("LoadConfig() returned error: %v", err)
		}

		if cfg.Version != 1 {
			t.Fatalf("Version = %d, want 1", cfg.Version)
		}
		if cfg.Paths.MemoriesHome != "/tmp/mnemonic" {
			t.Fatalf("Paths.MemoriesHome = %q, want %q", cfg.Paths.MemoriesHome, "/tmp/mnemonic")
		}
		if cfg.Notes.DeleteBehavior != "delete" {
			t.Fatalf("Notes.DeleteBehavior = %q, want %q", cfg.Notes.DeleteBehavior, "delete")
		}
		if cfg.Notes.TrashDirName != ".deleted" {
			t.Fatalf("Notes.TrashDirName = %q, want %q", cfg.Notes.TrashDirName, ".deleted")
		}
		if cfg.Index.FTS {
			t.Fatalf("Index.FTS = true, want false")
		}
		if cfg.Index.WAL {
			t.Fatalf("Index.WAL = true, want false")
		}
		if cfg.Index.BusyTimeoutMs != 2500 {
			t.Fatalf("Index.BusyTimeoutMs = %d, want 2500", cfg.Index.BusyTimeoutMs)
		}
		if !cfg.Output.JSONPretty {
			t.Fatalf("Output.JSONPretty = false, want true")
		}
		if cfg.Logging.Level != "debug" {
			t.Fatalf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
		}
	})

	t.Run("requires version one when file exists", func(t *testing.T) {
		path := writeConfigFile(t, `
[notes]
delete_behavior = "delete"
`)

		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("LoadConfig() error = nil, want missing version error")
		}
		if !strings.Contains(err.Error(), "version = 1") {
			t.Fatalf("LoadConfig() error = %q, want mention of version = 1", err)
		}
	})

	t.Run("rejects unsupported version with clear error", func(t *testing.T) {
		path := writeConfigFile(t, `version = 999
`)

		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("LoadConfig() error = nil, want unsupported version error")
		}
		if !strings.Contains(err.Error(), "unsupported config version 999") {
			t.Fatalf("LoadConfig() error = %q, want unsupported config version 999", err)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		path := writeConfigFile(t, `version = 1
unknown = "value"
`)

		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("LoadConfig() error = nil, want unknown-field error")
		}
		if !strings.Contains(err.Error(), "unknown key") {
			t.Fatalf("LoadConfig() error = %q, want unknown key message", err)
		}
	})

	t.Run("returns clear error for invalid toml", func(t *testing.T) {
		path := writeConfigFile(t, `version = 1
[notes
delete_behavior = "delete"
`)

		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("LoadConfig() error = nil, want syntax error")
		}
		if !strings.Contains(err.Error(), "config syntax error") {
			t.Fatalf("LoadConfig() error = %q, want config syntax error", err)
		}
	})
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	return path
}
