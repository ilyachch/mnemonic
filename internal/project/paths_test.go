package project

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveMemoriesRootForCentralProject(t *testing.T) {
	base := t.TempDir()

	got, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindCentral),
		MemoriesHome: base,
		Slug:         "backend",
	})
	require.NoError(t, err)

	want := filepath.Join(base, "backend")
	require.Equal(t, want, got)
}

func TestResolveMemoriesRootForLocalProject(t *testing.T) {
	repoRoot := t.TempDir()

	got, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindLocal),
		RepoRoot:     repoRoot,
		MemoriesPath: ".mnemonic-memories/backend",
	})
	require.NoError(t, err)

	want := filepath.Join(repoRoot, ".mnemonic-memories", "backend")
	require.Equal(t, want, got)
}

func TestResolveMemoriesRootRejectsTraversal(t *testing.T) {
	repoRoot := t.TempDir()

	_, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindLocal),
		RepoRoot:     repoRoot,
		MemoriesPath: "../escape",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes base directory")
}
