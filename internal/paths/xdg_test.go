package paths

import (
	"os"
	"testing"
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
	if paths.ConfigHome != "/myhome/.config" {
		t.Errorf("expected config_home to be /myhome/.config, got %q", paths.ConfigHome)
	}
	if paths.DataHome != "/myhome/.local/share" {
		t.Errorf("expected data_home to be /myhome/.local/share, got %q", paths.DataHome)
	}
	if paths.StateHome != "/myhome/.local/state" {
		t.Errorf("expected state_home to be /myhome/.local/state, got %q", paths.StateHome)
	}
	if paths.CacheHome != "/myhome/.cache" {
		t.Errorf("expected cache_home to be /myhome/.cache, got %q", paths.CacheHome)
	}

	// Test 2: Absolute env variables are respected
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	os.Setenv("XDG_STATE_HOME", "/custom/state")
	os.Setenv("XDG_CACHE_HOME", "/custom/cache")

	paths = GetXDGPaths()
	if paths.ConfigHome != "/custom/config" {
		t.Errorf("expected /custom/config, got %q", paths.ConfigHome)
	}
	if paths.DataHome != "/custom/data" {
		t.Errorf("expected /custom/data, got %q", paths.DataHome)
	}
	if paths.StateHome != "/custom/state" {
		t.Errorf("expected /custom/state, got %q", paths.StateHome)
	}
	if paths.CacheHome != "/custom/cache" {
		t.Errorf("expected /custom/cache, got %q", paths.CacheHome)
	}

	// Test 3: Relative env variables are ignored and fallback to default
	os.Setenv("XDG_CONFIG_HOME", "relative/config")
	os.Setenv("XDG_DATA_HOME", "./relative/data")
	os.Setenv("XDG_STATE_HOME", "../relative/state")
	os.Setenv("XDG_CACHE_HOME", "relative/cache")

	paths = GetXDGPaths()
	if paths.ConfigHome != "/myhome/.config" {
		t.Errorf("expected fallback /myhome/.config, got %q", paths.ConfigHome)
	}
	if paths.DataHome != "/myhome/.local/share" {
		t.Errorf("expected fallback /myhome/.local/share, got %q", paths.DataHome)
	}
	if paths.StateHome != "/myhome/.local/state" {
		t.Errorf("expected fallback /myhome/.local/state, got %q", paths.StateHome)
	}
	if paths.CacheHome != "/myhome/.cache" {
		t.Errorf("expected fallback /myhome/.cache, got %q", paths.CacheHome)
	}
}
