package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectImportCommandDefaultsToDot(t *testing.T) {
	_, importRoot := seedImportProject(t)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(importRoot)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, importRoot, got.Path)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 0, got.CopiedFiles)
	require.Equal(t, 1, got.Indexed)
	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectImportCommandDryRunReturnsCandidatesAndDoesNotWriteRegistry(t *testing.T) {
	_, importRoot := seedImportProject(t)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(importRoot)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--dry-run", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, importRoot, got.Path)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 0, got.CopiedFiles)
	require.Equal(t, 0, got.Indexed)
	require.Len(t, got.Candidates, 1)
	candidate := got.Candidates[0]
	require.Equal(t, "backend", candidate.Slug)
	require.Equal(t, importRoot, candidate.RepoRootAbs)

	listResult := executeCommand("project", "list", "--json")
	require.NoError(t, listResult.Err, "project list returned error\nstderr: %s", listResult.Err)
	var listOutput projectListOutput
	err = json.Unmarshal([]byte(listResult.Stdout), &listOutput)
	require.NoError(t, err, "failed to decode list JSON\nstdout: %s", listResult.Stdout)
	require.Len(t, listOutput.Projects, 0)
}

func TestProjectImportCommandReturnsNotFoundForMissingPath(t *testing.T) {
	cwd := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", "missing")
	require.Error(t, result.Err, "project import error = nil, want not found")
	require.Equal(t, 3, ExitCodeForError(result.Err))
}

func TestProjectImportCommandNormalizesRelativePath(t *testing.T) {
	_, importRoot := seedImportProject(t)
	target := filepath.Join(filepath.Dir(importRoot), "notes-target")
	require.NoError(t, os.Rename(importRoot, target))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(filepath.Dir(target))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", "notes-target", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, target, got.Path)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 0, got.CopiedFiles)
	require.Equal(t, 1, got.Indexed)
}

func seedImportProject(t *testing.T) (string, string) {
	t.Helper()

	cwd := testutil.CleanEnvForTest(t)
	importRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(importRoot, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "backend"
	manifest.Slug = "backend"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(importRoot, "mnemonic.toml"), manifest))

	return cwd, importRoot
}
