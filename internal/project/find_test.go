package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFindNearestMnemonicFileInCwd(t *testing.T) {
	cwd := t.TempDir()
	writeTestMnemonicFile(t, filepath.Join(cwd, ".mnemonic"), "backend")

	got, err := FindNearestMnemonicFile(cwd)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(cwd, ".mnemonic"), got)
}

func TestFindNearestMnemonicFileInParent(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "sub", "dir")
	require.NoError(t, os.MkdirAll(child, 0o755))
	writeTestMnemonicFile(t, filepath.Join(root, ".mnemonic"), "backend")

	got, err := FindNearestMnemonicFile(child)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, ".mnemonic"), got)
}

func TestFindNearestMnemonicFileStopsAtRoot(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "repo", "sub", "dir")
	require.NoError(t, os.MkdirAll(cwd, 0o755))

	got, err := FindNearestMnemonicFile(cwd)
	require.Error(t, err)
	_ = got
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
	require.NoError(t, WriteMnemonicFile(path, file))
}