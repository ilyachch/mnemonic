package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashBytesDeterministic(t *testing.T) {
	content := []byte("mnemonic note bytes")
	sum := sha256.Sum256(content)

	got1 := HashBytes(content)
	got2 := HashBytes(content)
	want := "sha256:" + hex.EncodeToString(sum[:])

	assert.Equal(t, want, got1)
	assert.Equal(t, want, got2)
}

func TestHashBytesChangesWhenInputChanges(t *testing.T) {
	base := HashBytes([]byte("mnemonic note bytes"))
	changed := HashBytes([]byte("mnemonic note bytez"))

	assert.NotEqual(t, base, changed)
}

func TestHashBytesUsesSha256Prefix(t *testing.T) {
	got := HashBytes([]byte("mnemonic"))

	assert.Greater(t, len(got), len("sha256:"))
	assert.Equal(t, "sha256:", got[:7])
}