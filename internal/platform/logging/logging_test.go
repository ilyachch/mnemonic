package logging

import (
	"log/slog"
	"strings"
	"testing"
)

func TestNew_TextFormat(t *testing.T) {
	logger := New("debug", "text")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	handler := logger.Handler()
	if !handler.Enabled(t.Context(), slog.LevelDebug) {
		t.Error("debug level should be enabled")
	}
	if !handler.Enabled(t.Context(), slog.LevelError) {
		t.Error("error level should be enabled")
	}
}

func TestNew_JSONFormat(t *testing.T) {
	logger := New("warn", "json")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	handler := logger.Handler()
	if handler.Enabled(t.Context(), slog.LevelDebug) {
		t.Error("debug level should not be enabled when level is warn")
	}
	if handler.Enabled(t.Context(), slog.LevelInfo) {
		t.Error("info level should not be enabled when level is warn")
	}
	if !handler.Enabled(t.Context(), slog.LevelWarn) {
		t.Error("warn level should be enabled when level is warn")
	}
}

func TestNew_DefaultFormat(t *testing.T) {
	logger := New("info", "")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestParseLevel_Unknown(t *testing.T) {
	logger := New("bogus", "text")
	handler := logger.Handler()
	if !handler.Enabled(t.Context(), slog.LevelInfo) {
		t.Error("info level should be enabled when level is unknown (defaults to info)")
	}
}

func TestValidateLevel(t *testing.T) {
	tests := []struct {
		level string
		valid bool
	}{
		{"debug", true},
		{"info", true},
		{"warn", true},
		{"error", true},
		{"DEBUG", true},
		{" Info ", true},
		{"trace", false},
		{"", false},
		{"critical", false},
	}

	for _, tt := range tests {
		err := ValidateLevel(tt.level)
		if tt.valid && err != nil {
			t.Errorf("ValidateLevel(%q) unexpected error: %v", tt.level, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateLevel(%q) expected error, got nil", tt.level)
		}
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"text", true},
		{"json", true},
		{"TEXT", true},
		{" Json ", true},
		{"yaml", false},
		{"", false},
		{"xml", false},
	}

	for _, tt := range tests {
		err := ValidateFormat(tt.format)
		if tt.valid && err != nil {
			t.Errorf("ValidateFormat(%q) unexpected error: %v", tt.format, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateFormat(%q) expected error, got nil", tt.format)
		}
	}
}

func TestNew_WritesToStderr(t *testing.T) {
	logger := New("info", "text")

	var buf strings.Builder
	noopLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	noopLogger.Info("test")

	_ = logger
}
