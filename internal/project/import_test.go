package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestResolveImportPathDefaultsToDot(t *testing.T) {
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

	got, err := ResolveImportPath(ImportInput{})
	if err != nil {
		t.Fatalf("ResolveImportPath() error = %v", err)
	}

	want := cwd
	if got != want {
		t.Fatalf("ResolveImportPath() = %q, want %q", got, want)
	}
}

func TestResolveImportPathNormalizesRelativePath(t *testing.T) {
	cwd := t.TempDir()
	target := filepath.Join(cwd, "notes")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

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

	got, err := ResolveImportPath(ImportInput{Path: "notes"})
	if err != nil {
		t.Fatalf("ResolveImportPath() error = %v", err)
	}

	if got != target {
		t.Fatalf("ResolveImportPath() = %q, want %q", got, target)
	}
}

func TestResolveImportPathReturnsNotFoundForMissingPath(t *testing.T) {
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

	_, err = ResolveImportPath(ImportInput{Path: "missing"})
	if err == nil {
		t.Fatal("ResolveImportPath() error = nil, want not found")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("ResolveImportPath() error = %v, want app error", err)
	}
	if appErr.Code != app.CodeNotFound {
		t.Fatalf("exit code = %d, want %d", appErr.Code, app.CodeNotFound)
	}
}

func TestImportProjectFindsNearestMnemonicFileFromSubdirectory(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	writeImportMnemonicFile(t, filepath.Join(repoRoot, ".mnemonic"), "backend")

	result, err := ImportProject(ImportInput{Path: subdir})
	if err != nil {
		t.Fatalf("ImportProject() error = %v", err)
	}

	if result.Path != subdir {
		t.Fatalf("ImportProject() path = %q, want %q", result.Path, subdir)
	}
	if result.Imported != 1 {
		t.Fatalf("ImportProject() imported = %d, want 1", result.Imported)
	}
	if result.CopiedFiles != 0 {
		t.Fatalf("ImportProject() copied_files = %d, want 0", result.CopiedFiles)
	}
	if result.Indexed != 0 {
		t.Fatalf("ImportProject() indexed = %d, want 0", result.Indexed)
	}

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("project count = %d, want 1", count)
	}
}

func TestImportProjectDryRunReturnsCandidatesWithoutWritingRegistry(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	writeImportMnemonicFile(t, filepath.Join(repoRoot, ".mnemonic"), "backend")

	result, err := ImportProject(ImportInput{Path: subdir, DryRun: true})
	if err != nil {
		t.Fatalf("ImportProject() error = %v", err)
	}

	if result.Imported != 1 {
		t.Fatalf("ImportProject() imported = %d, want 1", result.Imported)
	}
	if result.CopiedFiles != 0 {
		t.Fatalf("ImportProject() copied_files = %d, want 0", result.CopiedFiles)
	}
	if result.Indexed != 0 {
		t.Fatalf("ImportProject() indexed = %d, want 0", result.Indexed)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("ImportProject() candidates = %d, want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if candidate.Slug != "backend" {
		t.Fatalf("candidate slug = %q, want backend", candidate.Slug)
	}
	if candidate.MnemonicFileAbs != filepath.Join(repoRoot, ".mnemonic") {
		t.Fatalf("candidate mnemonic_file_abs = %q, want %q", candidate.MnemonicFileAbs, filepath.Join(repoRoot, ".mnemonic"))
	}

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("project count = %d, want 0", count)
	}
}

func TestImportProjectSkipsNonLocalProjects(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	mixed := &MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Projects: []MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  "backend",
				Slug:                  "backend",
				Kind:                  ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", "backend"),
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440001",
				Name:                  "infra",
				Slug:                  "infra",
				Kind:                  ProjectKindRegular,
				MemoriesPath:          "infra",
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	if err := WriteMnemonicFile(filepath.Join(repoRoot, ".mnemonic"), mixed); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}

	result, err := ImportProject(ImportInput{Path: subdir})
	if err != nil {
		t.Fatalf("ImportProject() error = %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("ImportProject() imported = %d, want 1", result.Imported)
	}

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("project count = %d, want 1", count)
	}
}

func writeImportMnemonicFile(t *testing.T, path string, slug string) {
	t.Helper()

	file := &MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Projects: []MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  slug,
				Slug:                  slug,
				Kind:                  ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", slug),
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	if err := WriteMnemonicFile(path, file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}
}
