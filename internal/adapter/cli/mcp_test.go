package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/stretchr/testify/require"
)

func TestMCPCommandHelpShowsProjectFlag(t *testing.T) {
	setWritableMCPEnv(t)
	res := executeCommand("mcp", "--help")
	require.NoError(t, res.Err)
	require.Contains(t, res.Stdout, "--project")
	require.Contains(t, res.Stdout, "--read-only")
}

func TestMCPCommandReadOnlyFlagAndEnvAreAdditive(t *testing.T) {
	setWritableMCPEnv(t)
	root := newTestRoot(t)
	cmd, _, err := root.Find([]string{"mcp"})
	require.NoError(t, err)

	t.Setenv("MNEMONIC_READ_ONLY", "yes")
	require.True(t, mcpReadOnlyEnabled(cmd))

	t.Setenv("MNEMONIC_READ_ONLY", "no")
	require.NoError(t, cmd.Flags().Set("read-only", "true"))
	require.True(t, mcpReadOnlyEnabled(cmd))
}

func TestMCPCommandReturnsClearErrorWithoutProjectContext(t *testing.T) {
	setWritableMCPEnv(t)
	cwd := t.TempDir()
	prevWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	res := executeCommand("mcp")
	require.Empty(t, res.Stdout)
	require.Error(t, res.Err, "expected error, got nil")
	require.Contains(t, res.Err.Error(), "no project selected")
	require.Contains(t, res.Stderr, "no project selected")
}

func TestMCPCommandRejectsBadEnvironmentProjectBeforeServing(t *testing.T) {
	setWritableMCPEnv(t)
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

	res := executeCommand("mcp")
	require.Error(t, res.Err, "expected error, got nil")
	require.Contains(t, res.Err.Error(), `project "missing" not found`)
	require.Empty(t, res.Stdout)
}

// registerRegistryProject inserts a local-mode project row for tests so the
// registry-first resolution has something to look up.
func registerRegistryProject(t *testing.T, cwd, slug string) {
	t.Helper()

	memoriesDir := filepath.Join(cwd, ".mnemonic-memories", slug)
	require.NoError(t, os.MkdirAll(memoriesDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.Type = project.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(memoriesDir, "mnemonic.toml"), manifest))

	// Write pointer file for file-based registry
	memHome, err := resolveMemoriesHome()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(memHome, 0o755))
	require.NoError(t, project.WritePointerFile(filepath.Join(memHome, slug+".toml"), &project.PointerFile{
		ManifestPath: filepath.Join(memoriesDir, "mnemonic.toml"),
	}))
}

func setWritableMCPEnv(t *testing.T) {
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
