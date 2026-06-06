package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Notes.DeleteBehavior != "trash" {
		t.Errorf("expected DeleteBehavior 'trash', got %q", cfg.Notes.DeleteBehavior)
	}
	if cfg.Notes.TrashDirName != ".trash" {
		t.Errorf("expected TrashDirName '.trash', got %q", cfg.Notes.TrashDirName)
	}
	if cfg.Index.BusyTimeoutMs != 5000 {
		t.Errorf("expected BusyTimeoutMs 5000, got %d", cfg.Index.BusyTimeoutMs)
	}
	if cfg.Version != 1 {
		t.Errorf("expected Version 1, got %d", cfg.Version)
	}
}
