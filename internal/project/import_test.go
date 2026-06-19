package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
	require.Contains(t, err.Error(), "not found")
}

func TestImportProjectRegistersRegularManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, registry.ApplySchema(db))

	root := t.TempDir()
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "demo"
	manifest.Slug = "demo"
	manifest.Kind = ManifestKindRegular
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, WriteMnemonicManifest(filepath.Join(root, "mnemonic.toml"), manifest))

	result, err := ImportProject(ImportInput{Path: root})
	require.NoError(t, err)
	require.Equal(t, root, result.Path)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.Indexed, "indexing requires app container; this is set by the CLI command, not by ImportProject itself")
	require.Len(t, result.Candidates, 1)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL AND slug = ?`, "demo").Scan(&count))
	require.Equal(t, 1, count)
}

func TestImportProjectDryRunDoesNotWriteRegistry(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, registry.ApplySchema(db))

	root := t.TempDir()
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "demo"
	manifest.Slug = "demo"
	manifest.Kind = ManifestKindRegular
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"

	require.NoError(t, WriteMnemonicManifest(filepath.Join(root, "mnemonic.toml"), manifest))

	result, err := ImportProject(ImportInput{Path: root, DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.Indexed)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count))
	require.Equal(t, 0, count)
}

func TestImportProjectReturnsNotFoundWithoutManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()

	_, err := ImportProject(ImportInput{Path: root})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mnemonic.toml not found")
}
