package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("ResolveTrashPath() error = %v", err)
	}

	want := filepath.Join(root, ".trash", "projects", "auth.deleted-20260602T150405Z.md")
	if got != want {
		t.Fatalf("ResolveTrashPath() = %q, want %q", got, want)
	}

	if info, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("stat trash parent = %v", err)
	} else if !info.IsDir() {
		t.Fatalf("trash parent is not a directory: %s", info.Mode())
	}
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
	if err != nil {
		t.Fatalf("ResolveTrashPath() error = %v", err)
	}

	want := filepath.Join(root, ".deleted", "note.deleted-20260602T150405Z.md")
	if got != want {
		t.Fatalf("ResolveTrashPath() = %q, want %q", got, want)
	}
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
	if err != nil {
		t.Fatalf("first ResolveTrashPath() error = %v", err)
	}
	if err := os.WriteFile(first, []byte("collision"), 0o644); err != nil {
		t.Fatalf("write first trash file = %v", err)
	}

	second, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: "projects/auth.md",
		TrashDirName: ".trash",
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("second ResolveTrashPath() error = %v", err)
	}

	want := filepath.Join(root, ".trash", "projects", "auth.deleted-20260602T150405Z-1.md")
	if second != want {
		t.Fatalf("ResolveTrashPath() = %q, want %q", second, want)
	}
}

func TestResolveTrashPathRejectsTraversal(t *testing.T) {
	_, err := ResolveTrashPath(TrashPathInput{
		RootDir:      t.TempDir(),
		OriginalPath: "../auth.md",
		TrashDirName: ".trash",
		Now:          time.Now,
	})
	if err == nil {
		t.Fatal("ResolveTrashPath() error = nil, want traversal rejection")
	}
}
