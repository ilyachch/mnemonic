package cli

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCompleteProjectNamesReturnsActiveSlugMatches(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

	db, err := registry.OpenDB()
	require.NoError(t, err)
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
		err := registry.RegisterProject(db, p)
		require.NoError(t, err, "RegisterProject(%s) error = %v", p.Slug, err)
	}

	_, err = db.Exec(`UPDATE projects SET removed_at = ? WHERE slug = ?`, now.UTC().Format(time.RFC3339), "personal")
	require.NoError(t, err, "remove personal project")

	got, directive := completeProjectNames(nil, nil, "ba")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.Equal(t, []string{"backend"}, got)
}

func TestProjectCommandsHaveProjectCompletion(t *testing.T) {
	require.NotNil(t, projectShowCmd.ValidArgsFunction, "project show missing ValidArgsFunction")
	require.NotNil(t, projectRemoveCmd.ValidArgsFunction, "project remove missing ValidArgsFunction")
	require.NotNil(t, projectReindexCmd.ValidArgsFunction, "project reindex missing ValidArgsFunction")
	require.NotNil(t, projectDoctorCmd.ValidArgsFunction, "project doctor missing ValidArgsFunction")
}