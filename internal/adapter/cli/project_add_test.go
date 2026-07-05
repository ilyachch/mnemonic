package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectAddCommandRegistersStructuredProject(t *testing.T) {
	cwd, importRoot := seedAddProject(t)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(importRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "add", ".", "--json")
	require.NoError(t, result.Err, "project add returned error\nstderr: %s", result.Stderr)

	var got projectAddOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "backend", got.Slug)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.ProjectID)
	require.Equal(t, 1, got.Indexed)
	require.Equal(t, "ok", got.IndexStatus)

	indexPath := testIndexPath(cwd, "550e8400-e29b-41d4-a716-446655440000")
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectAddCommandRejectsMissingManifest(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	importRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(importRoot, 0o755))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(importRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "add", ".")
	require.Error(t, result.Err, "project add error = nil, want not found")
	require.Equal(t, 3, ExitCodeForError(result.Err))
}

func seedAddProject(t *testing.T) (string, string) {
	t.Helper()

	cwd := testutil.CleanEnvForTest(t)
	importRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(importRoot, 0o755))

	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Backend"
	manifest.Slug = "backend"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC).Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(importRoot, "mnemonic.toml"), manifest))

	return cwd, importRoot
}
