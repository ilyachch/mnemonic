package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

type configShowJSON struct {
	ConfigHome   string `json:"config_home"`
	DataHome     string `json:"data_home"`
	StateHome    string `json:"state_home"`
	CacheHome    string `json:"cache_home"`
	MemoriesHome string `json:"memories_home"`
	ConfigFile   string `json:"config_file"`
}

func TestConfigShowCommandJSONDefaults(t *testing.T) {
	tmpRoot := testutil.CleanEnvForTest(t)
	tmpHome := tmpRoot
	tmpConfig := filepath.Join(tmpRoot, "config")
	tmpData := filepath.Join(tmpRoot, "data")
	tmpState := filepath.Join(tmpRoot, "state")
	tmpCache := filepath.Join(tmpRoot, "cache")

	result := executeCommand("config", "show", "--json")
	if result.Err != nil {
		t.Fatalf("config show returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got configShowJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}

	assertConfigShowJSON(t, got, configShowJSON{
		ConfigHome:   tmpConfig,
		DataHome:     tmpData,
		StateHome:    tmpState,
		CacheHome:    tmpCache,
		MemoriesHome: filepath.Join(tmpHome, ".mnemonic"),
		ConfigFile:   filepath.Join(tmpConfig, "mnemonic", "config.toml"),
	})
}

func TestConfigShowCommandLoadsConfigFile(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("version = 1\n[paths]\nmemories_home = \"~/from-config\"\n"), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	result := executeCommand("config", "show", "--json")
	if result.Err != nil {
		t.Fatalf("config show returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got configShowJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}

	assertConfigShowJSON(t, got, configShowJSON{
		ConfigHome:   tmpConfig,
		DataHome:     filepath.Join(tmpHome, ".local", "share"),
		StateHome:    filepath.Join(tmpHome, ".local", "state"),
		CacheHome:    filepath.Join(tmpHome, ".cache"),
		MemoriesHome: filepath.Join(tmpHome, "from-config"),
		ConfigFile:   configPath,
	})
}

func TestConfigShowCommandRejectsMissingVersionWithExitTwo(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("[paths]\nmemories_home = \"~/from-config\"\n"), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	result := executeCommand("config", "show", "--json")
	if result.Err == nil {
		t.Fatal("config show error = nil, want missing version error")
	}
	if ExitCodeForError(result.Err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCodeForError(result.Err))
	}
	if !strings.Contains(result.Err.Error(), "version = 1") {
		t.Fatalf("error = %q, want mention of version = 1", result.Err)
	}
}

func TestConfigShowCommandRejectsUnsupportedVersionWithExitTwo(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("version = 999\n"), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	result := executeCommand("config", "show", "--json")
	if result.Err == nil {
		t.Fatal("config show error = nil, want unsupported version error")
	}
	if ExitCodeForError(result.Err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCodeForError(result.Err))
	}
	if !strings.Contains(result.Err.Error(), "unsupported config version 999") {
		t.Fatalf("error = %q, want unsupported config version 999", result.Err)
	}
}

func assertConfigShowJSON(t *testing.T, got, want configShowJSON) {
	t.Helper()

	if got.ConfigHome != want.ConfigHome {
		t.Fatalf("config_home = %q, want %q", got.ConfigHome, want.ConfigHome)
	}
	if got.DataHome != want.DataHome {
		t.Fatalf("data_home = %q, want %q", got.DataHome, want.DataHome)
	}
	if got.StateHome != want.StateHome {
		t.Fatalf("state_home = %q, want %q", got.StateHome, want.StateHome)
	}
	if got.CacheHome != want.CacheHome {
		t.Fatalf("cache_home = %q, want %q", got.CacheHome, want.CacheHome)
	}
	if got.MemoriesHome != want.MemoriesHome {
		t.Fatalf("memories_home = %q, want %q", got.MemoriesHome, want.MemoriesHome)
	}
	if got.ConfigFile != want.ConfigFile {
		t.Fatalf("config_file = %q, want %q", got.ConfigFile, want.ConfigFile)
	}
}
