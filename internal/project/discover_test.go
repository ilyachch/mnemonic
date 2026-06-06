package project

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestDiscoverProjectsRegistersDirectChildrenOnly(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	if err := os.MkdirAll(memoriesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverManifest(t, filepath.Join(memoriesHome, "backend"), "backend", ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverManifest() error = %v", err)
	}
	if err := writeDiscoverManifest(t, filepath.Join(memoriesHome, "personal"), "personal", ManifestKindDetached); err != nil {
		t.Fatalf("writeDiscoverManifest() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(memoriesHome, "ignored", "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverManifest(t, filepath.Join(memoriesHome, "ignored", "nested"), "nested", ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverManifest() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(memoriesHome, "empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome})
	if err != nil {
		t.Fatalf("DiscoverProjects() error = %v", err)
	}
	if result.MemoriesHome != memoriesHome {
		t.Fatalf("DiscoverProjects() memories_home = %q, want %q", result.MemoriesHome, memoriesHome)
	}
	if result.Discovered != 2 {
		t.Fatalf("DiscoverProjects() discovered = %d, want 2", result.Discovered)
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
	if count != 2 {
		t.Fatalf("project count = %d, want 2", count)
	}

	assertDiscoverProjectRow(t, db, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(memoriesHome, "backend"), filepath.Join(memoriesHome, "backend", "mnemonic.toml"), string(ProjectKindRegular))
	assertDiscoverProjectRow(t, db, "550e8400-e29b-41d4-a716-446655440001", filepath.Join(memoriesHome, "personal"), filepath.Join(memoriesHome, "personal", "mnemonic.toml"), string(registry.ProjectKindDetached))

	if _, err := os.Stat(filepath.Join(memoriesHome, "backend", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("backend index.sqlite exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(memoriesHome, "personal", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("personal index.sqlite exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(memoriesHome, "ignored", "nested", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("nested index.sqlite exists or stat failed unexpectedly: %v", err)
	}
}

func TestDiscoverProjectsDryRunReportsInvalidManifests(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	if err := os.MkdirAll(memoriesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverManifest(t, filepath.Join(memoriesHome, "backend"), "backend", ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverManifest() error = %v", err)
	}
	if err := writeInvalidDiscoverManifest(filepath.Join(memoriesHome, "broken"), "broken"); err != nil {
		t.Fatalf("writeInvalidDiscoverManifest() error = %v", err)
	}

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome, DryRun: true})
	if err != nil {
		t.Fatalf("DiscoverProjects() error = %v", err)
	}
	if result.Discovered != 1 {
		t.Fatalf("DiscoverProjects() discovered = %d, want 1", result.Discovered)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("DiscoverProjects() errors = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Path != filepath.Join(memoriesHome, "broken", "mnemonic.toml") {
		t.Fatalf("DiscoverProjects() error path = %q, want %q", result.Errors[0].Path, filepath.Join(memoriesHome, "broken", "mnemonic.toml"))
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

func TestDiscoverProjectsDryRunRejectsUnsupportedManifestVersionWithoutTouchingMarkdown(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := filepath.Join(cwd, "memories")

	projectRoot := filepath.Join(memoriesHome, "broken")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	notePath := filepath.Join(projectRoot, "note.md")
	noteBody := []byte("# Note\n\nUnchanged.\n")
	if err := os.WriteFile(notePath, noteBody, 0o644); err != nil {
		t.Fatalf("WriteFile(note) error = %v", err)
	}
	beforeInfo, err := os.Stat(notePath)
	if err != nil {
		t.Fatalf("Stat(note before) error = %v", err)
	}

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
	if err := os.WriteFile(filepath.Join(projectRoot, "mnemonic.toml"), manifest, 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}

	result, err := DiscoverProjects(DiscoverInput{MemoriesHome: memoriesHome, DryRun: true})
	if err != nil {
		t.Fatalf("DiscoverProjects() error = %v", err)
	}
	if result.Discovered != 0 {
		t.Fatalf("DiscoverProjects() discovered = %d, want 0", result.Discovered)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("DiscoverProjects() errors = %d, want 1", len(result.Errors))
	}
	if !strings.Contains(result.Errors[0].Error, "version 999 is unsupported; expected 1") {
		t.Fatalf("DiscoverProjects() error = %q, want unsupported version rejection", result.Errors[0].Error)
	}

	afterInfo, err := os.Stat(notePath)
	if err != nil {
		t.Fatalf("Stat(note after) error = %v", err)
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("note.md modtime changed: before %v after %v", beforeInfo.ModTime(), afterInfo.ModTime())
	}

	afterBody, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("ReadFile(note after) error = %v", err)
	}
	if string(afterBody) != string(noteBody) {
		t.Fatalf("note.md content changed:\n%s", string(afterBody))
	}
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
	if err := db.QueryRow(
		`SELECT l.memories_abs, COALESCE(l.manifest_abs, ''), l.source_kind, p.kind
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 WHERE p.project_id = ? AND p.removed_at IS NULL`,
		projectID,
	).Scan(&gotMemoriesAbs, &gotManifestAbs, &gotSourceKind, &gotKind); err != nil {
		t.Fatalf("query project row failed: %v", err)
	}
	if gotMemoriesAbs != memoriesAbs {
		t.Fatalf("memories_abs = %q, want %q", gotMemoriesAbs, memoriesAbs)
	}
	if gotManifestAbs != manifestAbs {
		t.Fatalf("manifest_abs = %q, want %q", gotManifestAbs, manifestAbs)
	}
	if gotSourceKind != string(registry.ProjectSourceKindDiscover) {
		t.Fatalf("source_kind = %q, want %q", gotSourceKind, registry.ProjectSourceKindDiscover)
	}
	if gotKind != kind {
		t.Fatalf("kind = %q, want %q", gotKind, kind)
	}
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
