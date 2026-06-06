package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveEffectivePaths(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		setEffectiveBaseEnv(t)

		got, err := ResolveEffectivePaths(EffectiveInput{})
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		wantHome := "/home/alice"
		assertEffectivePaths(t, got, effectiveWant{
			configHome:   filepath.Join(wantHome, ".config"),
			dataHome:     filepath.Join(wantHome, ".local", "share"),
			stateHome:    filepath.Join(wantHome, ".local", "state"),
			cacheHome:    filepath.Join(wantHome, ".cache"),
			memoriesHome: filepath.Join(wantHome, ".mnemonic"),
			configFile:   "",
			rawMemories:  "",
		})
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
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		assertEffectivePaths(t, got, effectiveWant{
			configHome:   xdgConfig,
			dataHome:     xdgData,
			stateHome:    xdgState,
			cacheHome:    xdgCache,
			memoriesHome: filepath.Join("/home/alice", ".mnemonic"),
			configFile:   "",
			rawMemories:  "",
		})
	})

	t.Run("config.toml beats xdg env for memories home", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)

		got, err := ResolveEffectivePaths(EffectiveInput{
			RawConfigMemoriesHome: "~/.custom-memories",
		})
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		assertEffectivePaths(t, got, effectiveWant{
			configHome:   xdgConfig,
			dataHome:     filepath.Join("/home/alice", ".local", "share"),
			stateHome:    filepath.Join("/home/alice", ".local", "state"),
			cacheHome:    filepath.Join("/home/alice", ".cache"),
			memoriesHome: filepath.Join("/home/alice", ".custom-memories"),
			configFile:   "",
			rawMemories:  "~/.custom-memories",
		})
	})

	t.Run("mnemonic env beats config.toml", func(t *testing.T) {
		setEffectiveBaseEnv(t)
		xdgConfig := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgConfig)
		t.Setenv("MNEMONIC_MEMORIES_HOME", "/tmp/mnemonic-memories")

		got, err := ResolveEffectivePaths(EffectiveInput{
			RawConfigMemoriesHome: "~/.custom-memories",
		})
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		assertEffectivePaths(t, got, effectiveWant{
			configHome:   xdgConfig,
			dataHome:     filepath.Join("/home/alice", ".local", "share"),
			stateHome:    filepath.Join("/home/alice", ".local", "state"),
			cacheHome:    filepath.Join("/home/alice", ".cache"),
			memoriesHome: "/tmp/mnemonic-memories",
			configFile:   "",
			rawMemories:  "",
		})
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
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		cliConfigFile, err := filepath.Abs("cli-config.toml")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}
		cliConfigHome, err := filepath.Abs("cli-config-home")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}
		cliDataHome, err := filepath.Abs("cli-data-home")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}
		cliStateHome, err := filepath.Abs("cli-state-home")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}
		cliCacheHome, err := filepath.Abs("cli-cache-home")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}
		cliMemoriesHome, err := filepath.Abs("cli-memories-home")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}

		assertEffectivePaths(t, got, effectiveWant{
			configHome:   cliConfigHome,
			dataHome:     cliDataHome,
			stateHome:    cliStateHome,
			cacheHome:    cliCacheHome,
			memoriesHome: cliMemoriesHome,
			configFile:   cliConfigFile,
			rawMemories:  "",
		})
	})

	t.Run("config file path is normalized when discovered path is relative", func(t *testing.T) {
		setEffectiveBaseEnv(t)

		got, err := ResolveEffectivePaths(EffectiveInput{
			ConfigFile: "relative/config.toml",
		})
		if err != nil {
			t.Fatalf("ResolveEffectivePaths() error = %v", err)
		}

		wantConfigFile, err := filepath.Abs("relative/config.toml")
		if err != nil {
			t.Fatalf("filepath.Abs() error = %v", err)
		}

		assertEffectivePaths(t, got, effectiveWant{
			configHome:   filepath.Join("/home/alice", ".config"),
			dataHome:     filepath.Join("/home/alice", ".local", "share"),
			stateHome:    filepath.Join("/home/alice", ".local", "state"),
			cacheHome:    filepath.Join("/home/alice", ".cache"),
			memoriesHome: filepath.Join("/home/alice", ".mnemonic"),
			configFile:   wantConfigFile,
			rawMemories:  "",
		})
	})
}

type effectiveWant struct {
	configHome   string
	dataHome     string
	stateHome    string
	cacheHome    string
	memoriesHome string
	configFile   string
	rawMemories  string
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

func assertEffectivePaths(t *testing.T, got EffectivePaths, want effectiveWant) {
	t.Helper()

	if got.ConfigHome != want.configHome {
		t.Fatalf("ConfigHome = %q, want %q", got.ConfigHome, want.configHome)
	}
	if got.DataHome != want.dataHome {
		t.Fatalf("DataHome = %q, want %q", got.DataHome, want.dataHome)
	}
	if got.StateHome != want.stateHome {
		t.Fatalf("StateHome = %q, want %q", got.StateHome, want.stateHome)
	}
	if got.CacheHome != want.cacheHome {
		t.Fatalf("CacheHome = %q, want %q", got.CacheHome, want.cacheHome)
	}
	if got.MemoriesHome != want.memoriesHome {
		t.Fatalf("MemoriesHome = %q, want %q", got.MemoriesHome, want.memoriesHome)
	}
	if got.ConfigFile != want.configFile {
		t.Fatalf("ConfigFile = %q, want %q", got.ConfigFile, want.configFile)
	}
	if got.RawConfigMemoriesHome != want.rawMemories {
		t.Fatalf("RawConfigMemoriesHome = %q, want %q", got.RawConfigMemoriesHome, want.rawMemories)
	}
}
