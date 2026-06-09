package cli

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/spf13/cobra"
)

func TestCompleteProjectNamesReturnsActiveSlugMatches(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	for _, p := range []registry.RegisterProjectInput{
		{
			ProjectID: "550e8400-e29b-41d4-a716-446655440000",
			Name:      "backend",
			Slug:      "backend",
			Kind:      registry.ProjectKindLocal,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				MnemonicFileAbs: filepath.Join(cwd, ".mnemonic"),
				RepoRootAbs:     cwd,
				MemoriesAbs:     filepath.Join(cwd, ".mnemonic-memories", "backend"),
				SourceKind:      registry.ProjectSourceKindInit,
			},
		},
		{
			ProjectID: "550e8400-e29b-41d4-a716-446655440001",
			Name:      "personal",
			Slug:      "personal",
			Kind:      registry.ProjectKindLocal,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				MnemonicFileAbs: filepath.Join(cwd, ".mnemonic"),
				RepoRootAbs:     cwd,
				MemoriesAbs:     filepath.Join(cwd, ".mnemonic-memories", "personal"),
				SourceKind:      registry.ProjectSourceKindInit,
			},
		},
	} {
		if err := registry.RegisterProject(db, p); err != nil {
			t.Fatalf("RegisterProject(%s) error = %v", p.Slug, err)
		}
	}

	if _, err := db.Exec(`UPDATE projects SET removed_at = ? WHERE slug = ?`, now.UTC().Format(time.RFC3339), "personal"); err != nil {
		t.Fatalf("remove personal project: %v", err)
	}

	got, directive := completeProjectNames(nil, nil, "ba")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %d, want %d", directive, cobra.ShellCompDirectiveNoFileComp)
	}
	if len(got) != 1 || got[0] != "backend" {
		t.Fatalf("completion = %#v, want [backend]", got)
	}
}

func TestProjectCommandsHaveProjectCompletion(t *testing.T) {
	if projectShowCmd.ValidArgsFunction == nil {
		t.Fatal("project show missing ValidArgsFunction")
	}
	if projectRemoveCmd.ValidArgsFunction == nil {
		t.Fatal("project remove missing ValidArgsFunction")
	}
	if projectReindexCmd.ValidArgsFunction == nil {
		t.Fatal("project reindex missing ValidArgsFunction")
	}
	if projectDoctorCmd.ValidArgsFunction == nil {
		t.Fatal("project doctor missing ValidArgsFunction")
	}
}
