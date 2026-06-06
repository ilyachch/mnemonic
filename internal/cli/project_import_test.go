package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestProjectImportCommandDefaultsToDot(t *testing.T) {
	_, subdir := seedImportProject(t)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--json")
	if result.Err != nil {
		t.Fatalf("project import returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectImportOutput
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Path != subdir {
		t.Fatalf("path = %q, want %q", got.Path, subdir)
	}
	if got.Imported != 1 {
		t.Fatalf("imported = %d, want 1", got.Imported)
	}
	if got.CopiedFiles != 0 {
		t.Fatalf("copied_files = %d, want 0", got.CopiedFiles)
	}
	if got.Indexed != 0 {
		t.Fatalf("indexed = %d, want 0", got.Indexed)
	}
}

func TestProjectImportCommandDryRunReturnsCandidatesAndDoesNotWriteRegistry(t *testing.T) {
	repoRoot, subdir := seedImportProject(t)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", ".", "--dry-run", "--json")
	if result.Err != nil {
		t.Fatalf("project import returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectImportOutput
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Path != subdir {
		t.Fatalf("path = %q, want %q", got.Path, subdir)
	}
	if got.Imported != 1 {
		t.Fatalf("imported = %d, want 1", got.Imported)
	}
	if got.CopiedFiles != 0 {
		t.Fatalf("copied_files = %d, want 0", got.CopiedFiles)
	}
	if got.Indexed != 0 {
		t.Fatalf("indexed = %d, want 0", got.Indexed)
	}
	if len(got.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(got.Candidates))
	}
	candidate := got.Candidates[0]
	if candidate.Slug != "backend" {
		t.Fatalf("candidate slug = %q, want backend", candidate.Slug)
	}
	if candidate.RepoRootAbs != repoRoot {
		t.Fatalf("candidate repo_root_abs = %q, want %q", candidate.RepoRootAbs, repoRoot)
	}

	listResult := executeCommand("project", "list", "--json")
	if listResult.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", listResult.Err, listResult.Stderr)
	}
	var listOutput projectListOutput
	if err := json.Unmarshal([]byte(listResult.Stdout), &listOutput); err != nil {
		t.Fatalf("failed to decode list JSON: %v\nstdout: %s", err, listResult.Stdout)
	}
	if len(listOutput.Projects) != 0 {
		t.Fatalf("project list length = %d, want 0", len(listOutput.Projects))
	}
}

func TestProjectImportCommandReturnsNotFoundForMissingPath(t *testing.T) {
	cwd := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", "missing")
	if result.Err == nil {
		t.Fatal("project import error = nil, want not found")
	}
	if got := ExitCodeForError(result.Err); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func TestProjectImportCommandNormalizesRelativePath(t *testing.T) {
	_, subdir := seedImportProject(t)
	target := filepath.Join(subdir, "notes")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	result := executeCommand("project", "import", "notes", "--json")
	if result.Err != nil {
		t.Fatalf("project import returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectImportOutput
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Path != target {
		t.Fatalf("path = %q, want %q", got.Path, target)
	}
	if got.Imported != 1 {
		t.Fatalf("imported = %d, want 1", got.Imported)
	}
	if got.CopiedFiles != 0 {
		t.Fatalf("copied_files = %d, want 0", got.CopiedFiles)
	}
	if got.Indexed != 0 {
		t.Fatalf("indexed = %d, want 0", got.Indexed)
	}
}

func seedImportProject(t *testing.T) (string, string) {
	t.Helper()

	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

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
	if err := project.WriteMnemonicFile(filepath.Join(repoRoot, ".mnemonic"), file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}

	return repoRoot, subdir
}
