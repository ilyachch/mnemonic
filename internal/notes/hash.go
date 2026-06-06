package notes

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashBytes returns a deterministic SHA-256 content hash for note bytes.
func HashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
