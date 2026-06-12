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
	require.Contains(t, res.Err.Error(), ".mnemonic not found")
	require.Contains(t, res.Stderr, ".mnemonic not found")
}

func TestMCPCommandRejectsBadEnvironmentProjectBeforeServing(t *testing.T) {
	setWritableMCPEnv(t)
	cwd := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(cwd, ".mnemonic"))

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

func writeMCPMnemonicFile(t *testing.T, path string) {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	file := project.NewMnemonicFile()
	file.CreatedAt = now
	file.UpdatedAt = now
	file.Projects = []project.MnemonicProject{
		{
			ID:                    "550e8400-e29b-41d4-a716-446655440000",
			Name:                  "personal",
			Slug:                  "personal",
			Kind:                  project.ProjectKindLocal,
			MemoriesPath:          ".mnemonic-memories/personal",
			MarkdownFormatVersion: 1,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
	err := project.WriteMnemonicFile(path, file)
	require.NoError(t, err)
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
