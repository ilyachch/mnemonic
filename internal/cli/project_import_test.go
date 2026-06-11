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
	_, subdir := seedImportProject(t)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(subdir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, subdir, got.Path)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 0, got.CopiedFiles)
	require.Equal(t, 1, got.Indexed)
	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index file missing")
}

func TestProjectImportCommandDryRunReturnsCandidatesAndDoesNotWriteRegistry(t *testing.T) {
	repoRoot, subdir := seedImportProject(t)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(subdir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--dry-run", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, subdir, got.Path)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 0, got.CopiedFiles)
	require.Equal(t, 0, got.Indexed)
	require.Len(t, got.Candidates, 1)
	candidate := got.Candidates[0]
	require.Equal(t, "backend", candidate.Slug)
	require.Equal(t, repoRoot, candidate.RepoRootAbs)

	listResult := executeCommand("project", "list", "--json")
	require.NoError(t, listResult.Err, "project list returned error\nstderr: %s", listResult.Stderr)
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
	_, subdir := seedImportProject(t)
	target := filepath.Join(subdir, "notes")
	err := os.MkdirAll(target, 0o755)
	require.NoError(t, err)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(subdir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", "notes", "--json")
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
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	err := os.MkdirAll(subdir, 0o755)
	require.NoError(t, err)

	file := &project.MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Projects: []project.MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  "backend",
				Slug:                  "backend",
				Kind:                  project.ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", "backend"),
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	err = project.WriteMnemonicFile(filepath.Join(repoRoot, ".mnemonic"), file)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(repoRoot, ".mnemonic-memories", "backend"), 0o755)
	require.NoError(t, err)

	return repoRoot, subdir
}