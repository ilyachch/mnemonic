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
	"github.com/stretchr/testify/require"
)

func TestResolveImportPathDefaultsToDot(t *testing.T) {
	cwd := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	got, err := ResolveImportPath(ImportInput{})
	require.NoError(t, err)

	require.Equal(t, cwd, got)
}

func TestResolveImportPathNormalizesRelativePath(t *testing.T) {
	cwd := t.TempDir()
	target := filepath.Join(cwd, "notes")
	require.NoError(t, os.MkdirAll(target, 0o755))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	got, err := ResolveImportPath(ImportInput{Path: "notes"})
	require.NoError(t, err)

	require.Equal(t, target, got)
}

func TestResolveImportPathReturnsNotFoundForMissingPath(t *testing.T) {
	cwd := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	_, err = ResolveImportPath(ImportInput{Path: "missing"})
	require.Error(t, err)
	var appErr *app.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, app.CodeNotFound, appErr.Code)
}

func TestImportProjectFindsNearestMnemonicFileFromSubdirectory(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	writeImportMnemonicFile(t, filepath.Join(repoRoot, ".mnemonic"), "backend")

	result, err := ImportProject(ImportInput{Path: subdir})
	require.NoError(t, err)

	require.Equal(t, subdir, result.Path)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.CopiedFiles)
	require.Equal(t, 0, result.Indexed)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()
	require.NoError(t, registry.ApplySchema(db))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestImportProjectDryRunReturnsCandidatesWithoutWritingRegistry(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub", "dir")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	writeImportMnemonicFile(t, filepath.Join(repoRoot, ".mnemonic"), "backend")

	result, err := ImportProject(ImportInput{Path: subdir, DryRun: true})
	require.NoError(t, err)

	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.CopiedFiles)
	require.Equal(t, 0, result.Indexed)
	require.Len(t, result.Candidates, 1)
	candidate := result.Candidates[0]
	require.Equal(t, "backend", candidate.Slug)
	require.Equal(t, filepath.Join(repoRoot, ".mnemonic"), candidate.MnemonicFileAbs)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()
	require.NoError(t, registry.ApplySchema(db))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count))
	require.Equal(t, 0, count)
}

func TestImportProjectSkipsNonLocalProjects(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	repoRoot := filepath.Join(cwd, "repo")
	subdir := filepath.Join(repoRoot, "sub")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

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
	require.NoError(t, WriteMnemonicFile(filepath.Join(repoRoot, ".mnemonic"), mixed))

	result, err := ImportProject(ImportInput{Path: subdir})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()
	require.NoError(t, registry.ApplySchema(db))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count))
	require.Equal(t, 1, count)
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
	require.NoError(t, WriteMnemonicFile(path, file))
}
