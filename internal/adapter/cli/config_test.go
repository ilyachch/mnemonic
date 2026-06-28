package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got configShowJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "stdout: %s", result.Stdout)

	want := configShowJSON{
		ConfigHome:   tmpConfig,
		DataHome:     tmpData,
		StateHome:    tmpState,
		CacheHome:    tmpCache,
		MemoriesHome: filepath.Join(tmpHome, ".mnemonic"),
		ConfigFile:   filepath.Join(tmpConfig, "mnemonic", "config.toml"),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("config show JSON mismatch (-want +got):\n%s", diff)
	}
}

func TestConfigShowCommandLoadsConfigFile(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")
	tmpData := filepath.Join(tmpHome, "data")
	tmpState := filepath.Join(tmpHome, "state")
	tmpCache := filepath.Join(tmpHome, "cache")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("version = 1\n[paths]\nmemories_home = \"~/from-config\"\n"), 0o644)
	require.NoError(t, err)

	result := executeCommand("config", "show", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got configShowJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "stdout: %s", result.Stdout)

	want := configShowJSON{
		ConfigHome:   tmpConfig,
		DataHome:     tmpData,
		StateHome:    tmpState,
		CacheHome:    tmpCache,
		MemoriesHome: filepath.Join(tmpHome, "from-config"),
		ConfigFile:   configPath,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("config show JSON mismatch (-want +got):\n%s", diff)
	}
}

func TestConfigShowCommandRejectsMissingVersionWithExitTwo(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("[paths]\nmemories_home = \"~/from-config\"\n"), 0o644)
	require.NoError(t, err)

	result := executeCommand("config", "show", "--json")
	require.Error(t, result.Err, "config show error = nil, want missing version error")
	require.Equal(t, 2, ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "version = 1")
}

func TestConfigShowCommandRejectsUnsupportedVersionWithExitTwo(t *testing.T) {
	tmpHome := testutil.CleanEnvForTest(t)
	tmpConfig := filepath.Join(tmpHome, "config")

	configPath := filepath.Join(tmpConfig, "mnemonic", "config.toml")
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("version = 999\n"), 0o644)
	require.NoError(t, err)

	result := executeCommand("config", "show", "--json")
	require.Error(t, result.Err, "config show error = nil, want unsupported version error")
	require.Equal(t, 2, ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "unsupported config version 999")
}
