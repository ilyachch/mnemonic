package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// New creates a slog.Logger that writes to stderr with the specified level and format.
func New(level, format string) *slog.Logger {
	return NewWithWriter(os.Stderr, level, format)
}

// NewWithWriter creates a slog.Logger that writes to the specified writer.
func NewWithWriter(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ValidateLevel reports whether the given string is a supported log level.
func ValidateLevel(level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("unsupported log level %q: must be one of debug, info, warn, error", level)
	}
}

// ValidateFormat reports whether the given string is a supported log format.
func ValidateFormat(format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("unsupported log format %q: must be one of text, json", format)
	}
}
