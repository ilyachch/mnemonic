package paths

import "testing"

func TestGetMnemonicPaths(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		setDefaultEnv(t)

		paths, err := GetMnemonicPaths()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertPaths(t, paths, wantPaths{
			configHome:   "/myhome/.config",
			dataHome:     "/myhome/.local/share",
			stateHome:    "/myhome/.local/state",
			cacheHome:    "/myhome/.cache",
			memoriesHome: "/myhome/.mnemonic",
		})
	})

	t.Run("absolute overrides", func(t *testing.T) {
		tests := []struct {
			name      string
			envKey    string
			envValue  string
			assertion func(t *testing.T, got MnemonicPaths)
		}{
			{
				name:     "config",
				envKey:   "MNEMONIC_CONFIG_HOME",
				envValue: "/mconfig",
				assertion: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/mconfig",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:     "data",
				envKey:   "MNEMONIC_DATA_HOME",
				envValue: "/mdata",
				assertion: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/mdata",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:     "state",
				envKey:   "MNEMONIC_STATE_HOME",
				envValue: "/mstate",
				assertion: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/mstate",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:     "cache",
				envKey:   "MNEMONIC_CACHE_HOME",
				envValue: "/mcache",
				assertion: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/mcache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:     "memories",
				envKey:   "MNEMONIC_MEMORIES_HOME",
				envValue: "/mmemories",
				assertion: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/mmemories",
					})
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				setXDGEnv(t)
				t.Setenv(tt.envKey, tt.envValue)

				paths, err := GetMnemonicPaths()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				tt.assertion(t, paths)
			})
		}
	})

	t.Run("relative memories path returns validation error", func(t *testing.T) {
		setXDGEnv(t)
		t.Setenv("MNEMONIC_MEMORIES_HOME", "relative/memories")

		_, err := GetMnemonicPaths()
		if err == nil {
			t.Fatal("expected validation error for relative MNEMONIC_MEMORIES_HOME")
		}
	})

	t.Run("relative overrides fall back to xdg", func(t *testing.T) {
		tests := []struct {
			name   string
			envKey string
			want   func(t *testing.T, got MnemonicPaths)
		}{
			{
				name:   "config",
				envKey: "MNEMONIC_CONFIG_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:   "data",
				envKey: "MNEMONIC_DATA_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:   "state",
				envKey: "MNEMONIC_STATE_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
			{
				name:   "cache",
				envKey: "MNEMONIC_CACHE_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					assertPaths(t, got, wantPaths{
						configHome:   "/xdg/config",
						dataHome:     "/xdg/data",
						stateHome:    "/xdg/state",
						cacheHome:    "/xdg/cache",
						memoriesHome: "/myhome/.mnemonic",
					})
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				setXDGEnv(t)
				t.Setenv(tt.envKey, "relative/"+tt.name)

				paths, err := GetMnemonicPaths()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				tt.want(t, paths)
			})
		}
	})
}

type wantPaths struct {
	configHome   string
	dataHome     string
	stateHome    string
	cacheHome    string
	memoriesHome string
}

func setDefaultEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", "/myhome")
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

func setXDGEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", "/myhome")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
	t.Setenv("MNEMONIC_CONFIG_HOME", "")
	t.Setenv("MNEMONIC_DATA_HOME", "")
	t.Setenv("MNEMONIC_STATE_HOME", "")
	t.Setenv("MNEMONIC_CACHE_HOME", "")
	t.Setenv("MNEMONIC_MEMORIES_HOME", "")
}

func assertPaths(t *testing.T, got MnemonicPaths, want wantPaths) {
	t.Helper()

	if got.ConfigHome != want.configHome {
		t.Fatalf("config_home: want %q, got %q", want.configHome, got.ConfigHome)
	}
	if got.DataHome != want.dataHome {
		t.Fatalf("data_home: want %q, got %q", want.dataHome, got.DataHome)
	}
	if got.StateHome != want.stateHome {
		t.Fatalf("state_home: want %q, got %q", want.stateHome, got.StateHome)
	}
	if got.CacheHome != want.cacheHome {
		t.Fatalf("cache_home: want %q, got %q", want.cacheHome, got.CacheHome)
	}
	if got.MemoriesHome != want.memoriesHome {
		t.Fatalf("memories_home: want %q, got %q", want.memoriesHome, got.MemoriesHome)
	}
}
