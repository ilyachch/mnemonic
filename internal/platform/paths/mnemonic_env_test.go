package paths

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestGetMnemonicPaths(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		setDefaultEnv(t)

		paths, err := GetMnemonicPaths()
		require.NoError(t, err)

		want := MnemonicPaths{
			ConfigHome:   "/myhome/.config",
			DataHome:     "/myhome/.local/share",
			StateHome:    "/myhome/.local/state",
			CacheHome:    "/myhome/.cache",
			MemoriesHome: "/myhome/.mnemonic",
		}
		require.Empty(t, cmp.Diff(want, paths))
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
					want := MnemonicPaths{
						ConfigHome:   "/mconfig",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:     "data",
				envKey:   "MNEMONIC_DATA_HOME",
				envValue: "/mdata",
				assertion: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/mdata",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:     "state",
				envKey:   "MNEMONIC_STATE_HOME",
				envValue: "/mstate",
				assertion: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/mstate",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:     "cache",
				envKey:   "MNEMONIC_CACHE_HOME",
				envValue: "/mcache",
				assertion: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/mcache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:     "memories",
				envKey:   "MNEMONIC_MEMORIES_HOME",
				envValue: "/mmemories",
				assertion: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/mmemories",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				setXDGEnv(t)
				t.Setenv(tt.envKey, tt.envValue)

				paths, err := GetMnemonicPaths()
				require.NoError(t, err)

				tt.assertion(t, paths)
			})
		}
	})

	t.Run("relative memories path returns validation error", func(t *testing.T) {
		setXDGEnv(t)
		t.Setenv("MNEMONIC_MEMORIES_HOME", "relative/memories")

		_, err := GetMnemonicPaths()
		require.Error(t, err, "expected validation error for relative MNEMONIC_MEMORIES_HOME")
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
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:   "data",
				envKey: "MNEMONIC_DATA_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:   "state",
				envKey: "MNEMONIC_STATE_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
			{
				name:   "cache",
				envKey: "MNEMONIC_CACHE_HOME",
				want: func(t *testing.T, got MnemonicPaths) {
					want := MnemonicPaths{
						ConfigHome:   "/xdg/config",
						DataHome:     "/xdg/data",
						StateHome:    "/xdg/state",
						CacheHome:    "/xdg/cache",
						MemoriesHome: "/myhome/.mnemonic",
					}
					require.Empty(t, cmp.Diff(want, got))
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				setXDGEnv(t)
				t.Setenv(tt.envKey, "relative/"+tt.name)

				paths, err := GetMnemonicPaths()
				require.NoError(t, err)

				tt.want(t, paths)
			})
		}
	})
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
