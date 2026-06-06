package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindNearestMnemonicFileInCwd(t *testing.T) {
	cwd := t.TempDir()
	writeTestMnemonicFile(t, filepath.Join(cwd, ".mnemonic"), "backend")

	got, err := FindNearestMnemonicFile(cwd)
	if err != nil {
		t.Fatalf("FindNearestMnemonicFile() error = %v", err)
	}
	if got != filepath.Join(cwd, ".mnemonic") {
		t.Fatalf("FindNearestMnemonicFile() = %q, want %q", got, filepath.Join(cwd, ".mnemonic"))
	}
}

func TestFindNearestMnemonicFileInParent(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeTestMnemonicFile(t, filepath.Join(root, ".mnemonic"), "backend")

	got, err := FindNearestMnemonicFile(child)
	if err != nil {
		t.Fatalf("FindNearestMnemonicFile() error = %v", err)
	}
	if got != filepath.Join(root, ".mnemonic") {
		t.Fatalf("FindNearestMnemonicFile() = %q, want %q", got, filepath.Join(root, ".mnemonic"))
	}
}

func TestFindNearestMnemonicFileStopsAtRoot(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "repo", "sub", "dir")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, err := FindNearestMnemonicFile(cwd)
	if err == nil {
		t.Fatalf("FindNearestMnemonicFile() error = nil, want not found path %q", got)
	}
}

func writeTestMnemonicFile(t *testing.T, path string, slug string) {
	t.Helper()

	file := &MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Projects: []MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  slug,
				Slug:                  slug,
				Kind:                  ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", slug),
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	if err := WriteMnemonicFile(path, file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}
}
