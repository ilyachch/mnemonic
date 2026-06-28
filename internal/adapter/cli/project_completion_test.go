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
	root := newTestRoot(t)

	for _, path := range [][]string{
		{"project", "show"},
		{"project", "remove"},
		{"project", "reindex"},
		{"project", "doctor"},
	} {
		cmd, _, err := root.Find(path)
		require.NoErrorf(t, err, "missing command %v", path)
		require.NotNilf(t, cmd.ValidArgsFunction, "%s missing ValidArgsFunction", path)
	}
}
