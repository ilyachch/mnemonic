package project

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDiscoverProjectsRegistersDirectChildrenOnly(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))
	require.NoError(t, writeDiscoverManifest(t, filepath.Join(memoriesHome, "backend"), "backend", ManifestKindRegular))
	require.NoError(t, writeDiscoverManifest(t, filepath.Join(memoriesHome, "personal"), "personal", ManifestKindDetached))
	require.NoError(t, os.MkdirAll(filepath.Join(memoriesHome, "ignored", "nested"), 0o755))
	require.NoError(t, writeDiscoverManifest(t, filepath.Join(memoriesHome, "ignored", "nested"), "nested", ManifestKindRegular))
	require.NoError(t, os.MkdirAll(filepath.Join(memoriesHome, "empty"), 0o755))

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome})
	require.NoError(t, err)
	require.Equal(t, memoriesHome, result.MemoriesHome)
	require.Equal(t, 2, result.Discovered)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()
	require.NoError(t, registry.ApplySchema(db))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL`).Scan(&count))
	require.Equal(t, 2, count)

	assertDiscoverProjectRow(t, db, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(memoriesHome, "backend"), filepath.Join(memoriesHome, "backend", "mnemonic.toml"), string(ProjectKindRegular))
	assertDiscoverProjectRow(t, db, "550e8400-e29b-41d4-a716-446655440001", filepath.Join(memoriesHome, "personal"), filepath.Join(memoriesHome, "personal", "mnemonic.toml"), string(registry.ProjectKindDetached))

	_, err = os.Stat(filepath.Join(memoriesHome, "backend", "index.sqlite"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(memoriesHome, "personal", "index.sqlite"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(memoriesHome, "ignored", "nested", "index.sqlite"))
	require.True(t, os.IsNotExist(err))
}

func TestDiscoverProjectsDryRunReportsInvalidManifests(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))
	require.NoError(t, writeDiscoverManifest(t, filepath.Join(memoriesHome, "backend"), "backend", ManifestKindRegular))
	require.NoError(t, writeInvalidDiscoverManifest(filepath.Join(memoriesHome, "broken"), "broken"))

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome, DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 1, result.Discovered)
	require.Len(t, result.Errors, 1)
	require.Equal(t, filepath.Join(memoriesHome, "broken", "mnemonic.toml"), result.Errors[0].Path)

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

func TestDiscoverProjectsDryRunRejectsUnsupportedManifestVersionWithoutTouchingMarkdown(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	projectRoot := filepath.Join(memoriesHome, "broken")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	notePath := filepath.Join(projectRoot, "note.md")
	noteBody := []byte("# Note\n\nUnchanged.\n")
	require.NoError(t, os.WriteFile(notePath, noteBody, 0o644))
	beforeInfo, err := os.Stat(notePath)
	require.NoError(t, err)

	manifest := []byte(`version = 999
project_id = "550e8400-e29b-41d4-a716-446655440010"
name = "broken"
slug = "broken"
kind = "regular"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[layout]
notes_glob = ["**/*.md"]
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
`)
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "mnemonic.toml"), manifest, 0o644))

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome, DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 0, result.Discovered)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0].Error, "version 999 is unsupported; expected 1")

	afterInfo, err := os.Stat(notePath)
	require.NoError(t, err)
	require.True(t, afterInfo.ModTime().Equal(beforeInfo.ModTime()))

	afterBody, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, string(noteBody), string(afterBody))
}

func writeDiscoverManifest(t *testing.T, projectRoot, name string, kind ManifestKind) error {
	t.Helper()

	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return err
	}

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest := NewMnemonicManifest()
	manifest.ProjectID = projectIDForSlug(name)
	manifest.Name = name
	manifest.Slug = name
	manifest.Kind = kind
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now
	manifest.UpdatedAt = now
	manifest.Generator.App = "mnemonic"

	return WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest)
}

func writeInvalidDiscoverManifest(projectRoot, name string) error {
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return err
	}

	manifest := []byte(`version = 1
project_id = "550e8400-e29b-41d4-a716-446655440010"
name = "` + name + `"
slug = "` + name + `"
kind = "local"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[layout]
notes_glob = ["**/*.md"]
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
`)

	return os.WriteFile(filepath.Join(projectRoot, "mnemonic.toml"), manifest, 0o644)
}

func assertDiscoverProjectRow(t *testing.T, db *sql.DB, projectID, memoriesAbs, manifestAbs, kind string) {
	t.Helper()

	var gotMemoriesAbs, gotManifestAbs, gotSourceKind, gotKind string
	require.NoError(t, db.QueryRow(
		`SELECT l.memories_abs, COALESCE(l.manifest_abs, ''), l.source_kind, p.kind
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 WHERE p.project_id = ? AND p.removed_at IS NULL`,
		projectID,
	).Scan(&gotMemoriesAbs, &gotManifestAbs, &gotSourceKind, &gotKind))
	require.Equal(t, memoriesAbs, gotMemoriesAbs)
	require.Equal(t, manifestAbs, gotManifestAbs)
	require.Equal(t, string(registry.ProjectSourceKindDiscover), gotSourceKind)
	require.Equal(t, kind, gotKind)
}

func projectIDForSlug(slug string) string {
	switch slug {
	case "backend":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "personal":
		return "550e8400-e29b-41d4-a716-446655440001"
	case "nested":
		return "550e8400-e29b-41d4-a716-446655440002"
	default:
		return "550e8400-e29b-41d4-a716-446655440099"
	}
}
