package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "trash", cfg.Notes.DeleteBehavior)
	assert.Equal(t, ".trash", cfg.Notes.TrashDirName)
	assert.Equal(t, 5000, cfg.Index.BusyTimeoutMs)
	assert.Equal(t, 1, cfg.Version)
}
