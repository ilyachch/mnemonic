package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTrashPathBuildsExpectedPath(t *testing.T) {
	root := t.TempDir()

	got, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: "projects/auth.md",
		TrashDirName: ".trash",
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	require.NoError(t, err)

	want := filepath.Join(root, ".trash", "projects", "auth.deleted-20260602T150405Z.md")
	assert.Equal(t, want, got)

	info, err := os.Stat(filepath.Dir(got))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestResolveTrashPathUsesConfiguredTrashDir(t *testing.T) {
	root := t.TempDir()

	got, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: "note.md",
		TrashDirName: ".deleted",
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	require.NoError(t, err)

	want := filepath.Join(root, ".deleted", "note.deleted-20260602T150405Z.md")
	assert.Equal(t, want, got)
}

func TestResolveTrashPathAddsSuffixOnCollision(t *testing.T) {
	root := t.TempDir()
	fixedNow := func() time.Time {
		return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
	}

	first, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: "projects/auth.md",
		TrashDirName: ".trash",
		Now:          fixedNow,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(first, []byte("collision"), 0o644))

	second, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: "projects/auth.md",
		TrashDirName: ".trash",
		Now:          fixedNow,
	})
	require.NoError(t, err)

	want := filepath.Join(root, ".trash", "projects", "auth.deleted-20260602T150405Z-1.md")
	assert.Equal(t, want, second)
}

func TestResolveTrashPathRejectsTraversal(t *testing.T) {
	_, err := ResolveTrashPath(TrashPathInput{
		RootDir:      t.TempDir(),
		OriginalPath: "../auth.md",
		TrashDirName: ".trash",
		Now:          time.Now,
	})
	require.Error(t, err)
}