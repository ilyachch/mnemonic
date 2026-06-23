package cli

import (
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCompleteProjectNamesReturnsActiveSlugMatches(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	for _, slug := range []string{"backend", "personal"} {
		writeCentralProjectFixture(t, memoriesHome, slug, "550e8400-e29b-41d4-a716-4466554400"+slug[:1], time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC))
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
