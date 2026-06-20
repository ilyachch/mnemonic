package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
	require.Contains(t, err.Error(), "not found")
}

func TestImportProjectRegistersManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()

	root := t.TempDir()
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, WriteMnemonicManifest(filepath.Join(root, "mnemonic.toml"), manifest))

	result, err := ImportProject(ImportInput{Path: root}, memoriesHome)
	require.NoError(t, err)
	require.Equal(t, root, result.Path)
	require.Equal(t, 1, result.Imported)
	require.Len(t, result.Candidates, 1)

	// Verify pointer file was created
	pointerPath := filepath.Join(memoriesHome, "demo.toml")
	_, err = os.Stat(pointerPath)
	require.NoError(t, err, "pointer file should exist")

	data, err := os.ReadFile(pointerPath)
	require.NoError(t, err)
	pf, err := ParsePointerFile(data)
	require.NoError(t, err)
	require.Contains(t, pf.ManifestPath, "mnemonic.toml")
}

func TestImportProjectDryRunDoesNotCreatePointer(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()

	root := t.TempDir()
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, WriteMnemonicManifest(filepath.Join(root, "mnemonic.toml"), manifest))

	result, err := ImportProject(ImportInput{Path: root, DryRun: true}, memoriesHome)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	// Verify no pointer file was created
	pointerPath := filepath.Join(memoriesHome, "demo.toml")
	_, err = os.Stat(pointerPath)
	require.True(t, os.IsNotExist(err), "pointer file should not exist on dry run")
}

func TestImportProjectReturnsNotFoundWithoutManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	root := t.TempDir()

	_, err := ImportProject(ImportInput{Path: root}, memoriesHome)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mnemonic.toml not found")
}

func TestImportProjectRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()

	root := t.TempDir()
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, WriteMnemonicManifest(filepath.Join(root, "mnemonic.toml"), manifest))

	_, err := ImportProject(ImportInput{Path: root}, memoriesHome)
	require.NoError(t, err)

	// Try importing the same project again
	_, err = ImportProject(ImportInput{Path: root}, memoriesHome)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}
