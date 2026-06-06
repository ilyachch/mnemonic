package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHashBytesDeterministic(t *testing.T) {
	content := []byte("mnemonic note bytes")
	sum := sha256.Sum256(content)

	got1 := HashBytes(content)
	got2 := HashBytes(content)
	want := "sha256:" + hex.EncodeToString(sum[:])

	if got1 != want {
		t.Fatalf("HashBytes() = %q, want %q", got1, want)
	}
	if got2 != want {
		t.Fatalf("HashBytes() second call = %q, want %q", got2, want)
	}
}

func TestHashBytesChangesWhenInputChanges(t *testing.T) {
	base := HashBytes([]byte("mnemonic note bytes"))
	changed := HashBytes([]byte("mnemonic note bytez"))

	if base == changed {
		t.Fatalf("HashBytes() = %q for changed input, want different hashes", base)
	}
}

func TestHashBytesUsesSha256Prefix(t *testing.T) {
	got := HashBytes([]byte("mnemonic"))

	if len(got) <= len("sha256:") {
		t.Fatalf("HashBytes() = %q, want prefixed hex output", got)
	}
	if got[:7] != "sha256:" {
		t.Fatalf("HashBytes() prefix = %q, want %q", got[:7], "sha256:")
	}
}
