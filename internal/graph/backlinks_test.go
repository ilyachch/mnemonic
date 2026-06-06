package graph

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ilyachch/mnemonic/internal/index"
)

func TestBacklinksReturnsResolvedLinksWithMetadata(t *testing.T) {
	db := mustGraphDB(t)

	if _, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
		('target', 'p1', 'target-note', 'target.md', 'Target Note', 'h-target', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
		('src1', 'p1', 'source-one', 'source1.md', 'Source One', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
		('src2', 'p1', 'source-two', 'source2.md', 'Source Two', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`); err != nil {
		t.Fatalf("insert notes: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, relation_type, source_line) VALUES
		('l1', 'src1', 'target', 'target-note', '', 3),
		('l2', 'src2', 'target', 'Target Note', 'depends_on', 8),
		('l3', 'src2', NULL, 'missing-note', 'relates_to', 12)`); err != nil {
		t.Fatalf("insert links: %v", err)
	}

	links, err := Backlinks(db, "target")
	if err != nil {
		t.Fatalf("Backlinks() error = %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(links))
	}
	if links[0].NoteID != "src1" || links[0].RelationType != "" || links[0].SourceLine != 3 {
		t.Fatalf("links[0] = %#v", links[0])
	}
	if links[1].NoteID != "src2" || links[1].RelationType != "depends_on" || links[1].SourceLine != 8 {
		t.Fatalf("links[1] = %#v", links[1])
	}
}

func TestBacklinksRejectsNilDBAndEmptyTarget(t *testing.T) {
	if _, err := Backlinks(nil, "target"); err == nil {
		t.Fatal("Backlinks(nil, target) error = nil")
	}

	db := mustGraphDB(t)
	if _, err := Backlinks(db, ""); err == nil {
		t.Fatal("Backlinks(db, empty) error = nil")
	}
}

func mustGraphDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := index.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}
	return db
}
