package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCompleteProjectNamesReturnsActiveSlugMatches(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	// Create central projects for "backend" and "personal"
	for _, slug := range []string{"backend", "personal"} {
		projectDir := filepath.Join(memoriesHome, slug)
		require.NoError(t, os.MkdirAll(projectDir, 0o755))

		manifest := project.NewMnemonicManifest()
		manifest.ProjectID = "550e8400-e29b-41d4-a716-4466554400" + slug[:1]
		manifest.Name = slug
		manifest.Slug = slug
		manifest.MarkdownFormatVersion = 1
		manifest.CreatedAt = project.NowUTC()
		manifest.UpdatedAt = project.NowUTC()
		manifest.Generator.App = "mnemonic"
		require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))
	}

	root := newTestRoot(t)
	got, directive := completeProjectNames(root, nil, "ba")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.Equal(t, []string{"backend"}, got)
}

func TestProjectCommandsHaveProjectCompletion(t *testing.T) {
	require.NotNil(t, projectShowCmd.ValidArgsFunction, "project show missing ValidArgsFunction")
	require.NotNil(t, projectRemoveCmd.ValidArgsFunction, "project remove missing ValidArgsFunction")
	require.NotNil(t, projectReindexCmd.ValidArgsFunction, "project reindex missing ValidArgsFunction")
	require.NotNil(t, projectDoctorCmd.ValidArgsFunction, "project doctor missing ValidArgsFunction")
}
