package notes

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	_ "modernc.org/sqlite"
)

func TestResolveFollowsSelectorPrecedence(t *testing.T) {
	root := t.TempDir()

	writeResolvedNote(t, root, "uuid.md", markdown.Note{
		MnemonicNoteID: "11111111-1111-1111-1111-111111111111",
		Title:          "UUID Note",
		Slug:           "uuid-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "slug.md", markdown.Note{
		MnemonicNoteID: "22222222-2222-2222-2222-222222222222",
		Title:          "Slug Note",
		Slug:           "slug-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, filepath.Join("folder", "path.md"), markdown.Note{
		MnemonicNoteID: "33333333-3333-3333-3333-333333333333",
		Title:          "Path Note",
		Slug:           "path-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "title.md", markdown.Note{
		MnemonicNoteID: "44444444-4444-4444-4444-444444444444",
		Title:          "Exact Title",
		Slug:           "exact-title-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "normalized.md", markdown.Note{
		MnemonicNoteID: "55555555-5555-5555-5555-555555555555",
		Title:          "Normalized Title",
		Slug:           "custom-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	cases := []struct {
		name     string
		selector string
		wantPath string
	}{
		{name: "uuid", selector: "11111111-1111-1111-1111-111111111111", wantPath: "uuid.md"},
		{name: "slug", selector: "slug-note", wantPath: "slug.md"},
		{name: "path", selector: filepath.ToSlash(filepath.Join("folder", "path.md")), wantPath: filepath.ToSlash(filepath.Join("folder", "path.md"))},
		{name: "title", selector: "Exact Title", wantPath: "title.md"},
		{name: "normalized title", selector: "normalized-title", wantPath: "normalized.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(root, tc.selector)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got.Path != tc.wantPath {
				t.Fatalf("Resolve().Path = %q, want %q", got.Path, tc.wantPath)
			}
		})
	}
}

func TestResolveReturnsAmbiguousError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "a.md", markdown.Note{
		MnemonicNoteID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:          "Duplicate Title",
		Slug:           "dup-a",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "b.md", markdown.Note{
		MnemonicNoteID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title:          "Duplicate Title",
		Slug:           "dup-b",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "Duplicate Title")
	if err == nil {
		t.Fatal("Resolve() error = nil, want ambiguous error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeAmbiguous {
		t.Fatalf("Resolve() error = %v, want ambiguous error", err)
	}
}

func TestResolveReturnsNotFoundError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "note.md", markdown.Note{
		MnemonicNoteID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Title:          "Existing",
		Slug:           "existing",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "missing")
	if err == nil {
		t.Fatal("Resolve() error = nil, want not found error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeNotFound {
		t.Fatalf("Resolve() error = %v, want not found error", err)
	}
}

func TestResolveUsesProjectIndexWhenAvailable(t *testing.T) {
	projectRoot := filepath.Join(testutil.CleanEnvForTest(t), "project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	writeResolvedNote(t, projectRoot, "indexed.md", markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Indexed Note",
		Slug:           "indexed-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}
	if err := registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "project",
		Slug:      "project",
		Kind:      registry.ProjectKindLocal,
		CreatedAt: noteTime(),
		UpdatedAt: noteTime(),
		SeenAt:    noteTime(),
		Location: registry.ProjectLocationInput{
			MemoriesAbs: projectRoot,
			SourceKind:  registry.ProjectSourceKindInit,
		},
	}); err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}

	mnemonicPaths, err := paths.GetMnemonicPaths()
	if err != nil {
		t.Fatalf("GetMnemonicPaths() error = %v", err)
	}
	indexPath := filepath.Join(mnemonicPaths.StateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	indexDB, err := sql.Open("sqlite", indexPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() { _ = indexDB.Close() }()
	if _, err := indexDB.Exec(`CREATE TABLE notes (
		note_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		rel_path TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("CREATE TABLE notes error = %v", err)
	}
	if _, err := indexDB.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"66666666-6666-6666-6666-666666666666",
		"550e8400-e29b-41d4-a716-446655440000",
		"indexed-note",
		"indexed.md",
		"Indexed Note",
		"content-hash",
		noteTime().UTC().Format(time.RFC3339),
		noteTime().UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatalf("INSERT note error = %v", err)
	}

	got, err := Resolve(projectRoot, "indexed-note")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Path != "indexed.md" {
		t.Fatalf("Resolve().Path = %q, want indexed.md", got.Path)
	}
}

func TestResolveFallsBackToDiskWhenIndexIsStale(t *testing.T) {
	projectRoot := filepath.Join(testutil.CleanEnvForTest(t), "project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	writeResolvedNote(t, projectRoot, "indexed.md", markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Indexed Note",
		Slug:           "indexed-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, projectRoot, "fresh.md", markdown.Note{
		MnemonicNoteID: "77777777-7777-7777-7777-777777777777",
		Title:          "Fresh Note",
		Slug:           "fresh-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}
	if err := registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "project",
		Slug:      "project",
		Kind:      registry.ProjectKindLocal,
		CreatedAt: noteTime(),
		UpdatedAt: noteTime(),
		SeenAt:    noteTime(),
		Location: registry.ProjectLocationInput{
			MemoriesAbs: projectRoot,
			SourceKind:  registry.ProjectSourceKindInit,
		},
	}); err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}

	mnemonicPaths, err := paths.GetMnemonicPaths()
	if err != nil {
		t.Fatalf("GetMnemonicPaths() error = %v", err)
	}
	indexPath := filepath.Join(mnemonicPaths.StateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	indexDB, err := sql.Open("sqlite", indexPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() { _ = indexDB.Close() }()
	if _, err := indexDB.Exec(`CREATE TABLE notes (
		note_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		rel_path TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("CREATE TABLE notes error = %v", err)
	}
	if _, err := indexDB.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"66666666-6666-6666-6666-666666666666",
		"550e8400-e29b-41d4-a716-446655440000",
		"indexed-note",
		"indexed.md",
		"Indexed Note",
		"content-hash",
		noteTime().UTC().Format(time.RFC3339),
		noteTime().UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatalf("INSERT note error = %v", err)
	}

	got, err := Resolve(projectRoot, "77777777-7777-7777-7777-777777777777")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Path != "fresh.md" {
		t.Fatalf("Resolve().Path = %q, want fresh.md", got.Path)
	}
}

func writeResolvedNote(t *testing.T, root, relPath string, note markdown.Note) {
	t.Helper()

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func noteTime() time.Time {
	return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
}
