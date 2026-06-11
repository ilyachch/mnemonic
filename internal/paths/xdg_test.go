package paths

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

	// Test 1: Empty environment variables
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_STATE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	os.Setenv("HOME", "/myhome") // Set HOME to control user home dir behavior in test

	paths := GetXDGPaths()
	assert.Equal(t, "/myhome/.config", paths.ConfigHome)
	assert.Equal(t, "/myhome/.local/share", paths.DataHome)
	assert.Equal(t, "/myhome/.local/state", paths.StateHome)
	assert.Equal(t, "/myhome/.cache", paths.CacheHome)

	// Test 2: Absolute env variables are respected
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	os.Setenv("XDG_STATE_HOME", "/custom/state")
	os.Setenv("XDG_CACHE_HOME", "/custom/cache")

	paths = GetXDGPaths()
	assert.Equal(t, "/custom/config", paths.ConfigHome)
	assert.Equal(t, "/custom/data", paths.DataHome)
	assert.Equal(t, "/custom/state", paths.StateHome)
	assert.Equal(t, "/custom/cache", paths.CacheHome)

	// Test 3: Relative env variables are ignored and fallback to default
	os.Setenv("XDG_CONFIG_HOME", "relative/config")
	os.Setenv("XDG_DATA_HOME", "./relative/data")
	os.Setenv("XDG_STATE_HOME", "../relative/state")
	os.Setenv("XDG_CACHE_HOME", "relative/cache")

	paths = GetXDGPaths()
	assert.Equal(t, "/myhome/.config", paths.ConfigHome)
	assert.Equal(t, "/myhome/.local/share", paths.DataHome)
	assert.Equal(t, "/myhome/.local/state", paths.StateHome)
	assert.Equal(t, "/myhome/.cache", paths.CacheHome)
}