package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetXDGPaths(t *testing.T) {
	// Backup environment variables
	origConfig := os.Getenv("XDG_CONFIG_HOME")
	origData := os.Getenv("XDG_DATA_HOME")
	origState := os.Getenv("XDG_STATE_HOME")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")

	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origConfig)
		os.Setenv("XDG_DATA_HOME", origData)
		os.Setenv("XDG_STATE_HOME", origState)
		os.Setenv("XDG_CACHE_HOME", origCache)
		os.Setenv("HOME", origHome)
	}()

	// Test 1: Empty environment variables fall back to platform-native defaults.
	// adrg/xdg resolves native locations per OS (XDG dirs on Linux, Library on
	// macOS, Known Folders on Windows), so expectations are platform-specific.
	t.Run("defaults are platform-native", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Unsetenv("XDG_DATA_HOME")
		os.Unsetenv("XDG_STATE_HOME")
		os.Unsetenv("XDG_CACHE_HOME")
		os.Setenv("HOME", "/myhome")

		paths := GetXDGPaths()

		want := expectedDefaultXDGPaths(t, "/myhome")
		require.NotEmpty(t, want.ConfigHome, "no default expectation for GOOS=%s", runtime.GOOS)
		assert.Equal(t, want.ConfigHome, paths.ConfigHome)
		assert.Equal(t, want.DataHome, paths.DataHome)
		assert.Equal(t, want.StateHome, paths.StateHome)
		assert.Equal(t, want.CacheHome, paths.CacheHome)
	})

	// Test 2: Absolute env variables are respected
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	os.Setenv("XDG_STATE_HOME", "/custom/state")
	os.Setenv("XDG_CACHE_HOME", "/custom/cache")

	paths := GetXDGPaths()
	assert.Equal(t, "/custom/config", paths.ConfigHome)
	assert.Equal(t, "/custom/data", paths.DataHome)
	assert.Equal(t, "/custom/state", paths.StateHome)
	assert.Equal(t, "/custom/cache", paths.CacheHome)

	// Test 3: Relative env variables are ignored and fallback to default
	os.Setenv("XDG_CONFIG_HOME", "relative/config")
	os.Setenv("XDG_DATA_HOME", "./relative/data")
	os.Setenv("XDG_STATE_HOME", "../relative/state")
	os.Setenv("XDG_CACHE_HOME", "relative/cache")
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_STATE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	os.Setenv("HOME", "/myhome")

	paths = GetXDGPaths()
	want := expectedDefaultXDGPaths(t, "/myhome")
	assert.Equal(t, want.ConfigHome, paths.ConfigHome)
	assert.Equal(t, want.DataHome, paths.DataHome)
	assert.Equal(t, want.StateHome, paths.StateHome)
	assert.Equal(t, want.CacheHome, paths.CacheHome)
}

// expectedDefaultXDGPaths returns the platform-native XDG fallback locations
// for the given home directory. Unsupported platforms skip the test.
func expectedDefaultXDGPaths(t *testing.T, home string) XDGPaths {
	t.Helper()

	switch runtime.GOOS {
	case "linux", "freebsd", "netbsd", "openbsd", "dragonfly":
		return XDGPaths{
			ConfigHome: filepath.Join(home, ".config"),
			DataHome:   filepath.Join(home, ".local", "share"),
			StateHome:  filepath.Join(home, ".local", "state"),
			CacheHome:  filepath.Join(home, ".cache"),
		}
	case "darwin":
		appSupport := filepath.Join(home, "Library", "Application Support")
		return XDGPaths{
			ConfigHome: appSupport,
			DataHome:   appSupport,
			StateHome:  appSupport,
			CacheHome:  filepath.Join(home, "Library", "Caches"),
		}
	default:
		t.Skipf("no default XDG expectation for GOOS=%s", runtime.GOOS)
		return XDGPaths{}
	}
}
