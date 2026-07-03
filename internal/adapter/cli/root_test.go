package cli

import (
	"os"
	"testing"
)

func TestResolveLogLevelPrecedence(t *testing.T) {
	t.Run("flag takes precedence over everything", func(t *testing.T) {
		t.Setenv("MNEMONIC_LOG_LEVEL", "warn")
		result := resolveLogLevel("debug", "error")
		if result != "debug" {
			t.Errorf("expected debug, got %s", result)
		}
	})

	t.Run("env takes precedence over config", func(t *testing.T) {
		t.Setenv("MNEMONIC_LOG_LEVEL", "warn")
		result := resolveLogLevel("", "error")
		if result != "warn" {
			t.Errorf("expected warn, got %s", result)
		}
	})

	t.Run("config is used when flag and env are empty", func(t *testing.T) {
		result := resolveLogLevel("", "debug")
		if result != "debug" {
			t.Errorf("expected debug, got %s", result)
		}
	})

	t.Run("default is info when everything is empty", func(t *testing.T) {
		result := resolveLogLevel("", "")
		if result != "info" {
			t.Errorf("expected info, got %s", result)
		}
	})

	t.Run("empty env does not override config", func(t *testing.T) {
		os.Unsetenv("MNEMONIC_LOG_LEVEL")
		result := resolveLogLevel("", "error")
		if result != "error" {
			t.Errorf("expected error, got %s", result)
		}
	})
}

func TestResolveLogFormatPrecedence(t *testing.T) {
	t.Run("flag takes precedence over everything", func(t *testing.T) {
		t.Setenv("MNEMONIC_LOG_FORMAT", "text")
		result := resolveLogFormat("json", "text")
		if result != "json" {
			t.Errorf("expected json, got %s", result)
		}
	})

	t.Run("env takes precedence over config", func(t *testing.T) {
		t.Setenv("MNEMONIC_LOG_FORMAT", "json")
		result := resolveLogFormat("", "text")
		if result != "json" {
			t.Errorf("expected json, got %s", result)
		}
	})

	t.Run("config is used when flag and env are empty", func(t *testing.T) {
		result := resolveLogFormat("", "json")
		if result != "json" {
			t.Errorf("expected json, got %s", result)
		}
	})

	t.Run("default is text when everything is empty", func(t *testing.T) {
		result := resolveLogFormat("", "")
		if result != "text" {
			t.Errorf("expected text, got %s", result)
		}
	})

	t.Run("empty env does not override config", func(t *testing.T) {
		os.Unsetenv("MNEMONIC_LOG_FORMAT")
		result := resolveLogFormat("", "json")
		if result != "json" {
			t.Errorf("expected json, got %s", result)
		}
	})
}
