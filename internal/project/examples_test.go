package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExampleMnemonicFileParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "projects", "example", ".mnemonic"))
	if err != nil {
		t.Fatalf("ReadFile(.mnemonic) error = %v", err)
	}

	parsed, err := ParseMnemonicFile(data)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}

	if len(parsed.Projects) != 1 {
		t.Fatalf("len(Projects) = %d, want 1", len(parsed.Projects))
	}
}

func TestExampleMnemonicManifestParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "projects", "example", "mnemonic.toml"))
	if err != nil {
		t.Fatalf("ReadFile(mnemonic.toml) error = %v", err)
	}

	parsed, err := ParseMnemonicManifest(data)
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}

	if parsed.ProjectID == "" {
		t.Fatal("ProjectID is empty")
	}
}
