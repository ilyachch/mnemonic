package sqliteindex

import (
	"testing"

	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashBytes(t *testing.T) {
	got := markdownstore.HashBytes([]byte("hello"))
	require.NotEmpty(t, got)
	assert.True(t, len(got) > 7, "expected sha256: prefix and hex")
	assert.True(t, got[:7] == "sha256:", "expected sha256: prefix, got %q", got[:7])

	// Deterministic
	got2 := markdownstore.HashBytes([]byte("hello"))
	assert.Equal(t, got, got2)

	// Different inputs produce different hashes
	got3 := markdownstore.HashBytes([]byte("world"))
	assert.NotEqual(t, got, got3)
}

func TestResolveLinkTarget_NotFound(t *testing.T) {
	// Empty docs should return false
	target, ok := resolveLinkTarget(nil, nil, "nonexistent")
	assert.False(t, ok)
	assert.Empty(t, target)
}
