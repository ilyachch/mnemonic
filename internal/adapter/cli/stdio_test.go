package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStdioCommandHelpShowsProjectFlag(t *testing.T) {
	setWritableStdioEnv(t)
	res := executeCommand("stdio", "--help")
	require.NoError(t, res.Err)
	require.Contains(t, res.Stdout, "--project")
	require.Contains(t, res.Stdout, "--read-only")
}

func TestStdioCommandReadOnlyFlagAndEnvAreAdditive(t *testing.T) {
	setWritableStdioEnv(t)
	root := newTestRoot(t)
	cmd, _, err := root.Find([]string{"stdio"})
	require.NoError(t, err)

	t.Setenv("MNEMONIC_READ_ONLY", "yes")
	require.True(t, stdioReadOnlyEnabled(cmd))

	t.Setenv("MNEMONIC_READ_ONLY", "no")
	require.NoError(t, cmd.Flags().Set("read-only", "true"))
	require.True(t, stdioReadOnlyEnabled(cmd))
}

func TestStdioCommandReturnsClearErrorWithoutProjectContext(t *testing.T) {
	setWritableStdioEnv(t)
	cwd := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	res := executeCommand("stdio")
	require.Empty(t, res.Stdout)
	require.Error(t, res.Err, "expected error, got nil")
	require.Contains(t, res.Err.Error(), "no project selected")
	require.Contains(t, res.Stderr, "no project selected")
}

func TestStdioCommandRejectsBadEnvironmentProjectBeforeServing(t *testing.T) {
	setWritableStdioEnv(t)
	cwd := t.TempDir()
	registerRegistryProject(t, cwd, "personal")

	prevWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	t.Setenv("MNEMONIC_PROJECT", "missing")

	res := executeCommand("stdio")
	require.Error(t, res.Err, "expected error, got nil")
	require.Contains(t, res.Err.Error(), `project "missing" not found`)
	require.Empty(t, res.Stdout)
}

func TestHiddenMCPAliasStillResolves(t *testing.T) {
	setWritableStdioEnv(t)
	root := newTestRoot(t)
	stdioCmd, _, err := root.Find([]string{"stdio"})
	require.NoError(t, err)
	require.False(t, stdioCmd.Hidden)

	mcpCmd, _, err := root.Find([]string{"mcp"})
	require.NoError(t, err)
	require.True(t, mcpCmd.Hidden)
	require.Equal(t, "mcp", mcpCmd.Name())
}

// registerRegistryProject inserts a local-mode project row for tests so the
// registry-first resolution has something to look up.
func registerRegistryProject(t *testing.T, cwd, slug string) {
	t.Helper()

	require.NoError(t, writeLocalProjectFixture(t, cwd, slug))
}

func setWritableStdioEnv(t *testing.T) {
	t.Helper()

	base := t.TempDir()
	t.Setenv("HOME", base)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
	t.Setenv("GOCACHE", filepath.Join(base, "go-build"))
	t.Setenv("GOMODCACHE", filepath.Join(base, "go-mod"))
	t.Setenv("GOSUMDB", "off")
}
