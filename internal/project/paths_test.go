package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMemoriesRootForRegularProject(t *testing.T) {
	base := t.TempDir()

	got, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindRegular),
		MemoriesHome: base,
		MemoriesPath: "research",
	})
	if err != nil {
		t.Fatalf("ResolveMemoriesRoot() error = %v", err)
	}

	want := filepath.Join(base, "research")
	if got != want {
		t.Fatalf("ResolveMemoriesRoot() = %q, want %q", got, want)
	}
}

func TestResolveMemoriesRootForDetachedProject(t *testing.T) {
	base := t.TempDir()

	got, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ManifestKindDetached),
		MemoriesHome: base,
		Slug:         "backend",
	})
	if err != nil {
		t.Fatalf("ResolveMemoriesRoot() error = %v", err)
	}

	want := filepath.Join(base, "backend")
	if got != want {
		t.Fatalf("ResolveMemoriesRoot() = %q, want %q", got, want)
	}
}

func TestResolveMemoriesRootForLocalProject(t *testing.T) {
	repoRoot := t.TempDir()

	got, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindLocal),
		RepoRoot:     repoRoot,
		MemoriesPath: ".mnemonic-memories/backend",
	})
	if err != nil {
		t.Fatalf("ResolveMemoriesRoot() error = %v", err)
	}

	want := filepath.Join(repoRoot, ".mnemonic-memories", "backend")
	if got != want {
		t.Fatalf("ResolveMemoriesRoot() = %q, want %q", got, want)
	}
}

func TestResolveMemoriesRootRejectsTraversal(t *testing.T) {
	repoRoot := t.TempDir()

	_, err := ResolveMemoriesRoot(MemoriesRootInput{
		Kind:         string(ProjectKindLocal),
		RepoRoot:     repoRoot,
		MemoriesPath: "../escape",
	})
	if err == nil {
		t.Fatal("ResolveMemoriesRoot() error = nil, want traversal rejection")
	}
	if !strings.Contains(err.Error(), "escapes base directory") {
		t.Fatalf("ResolveMemoriesRoot() error = %v, want traversal rejection", err)
	}
}
