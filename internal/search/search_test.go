package search

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ilyachch/mnemonic/internal/index"
)

func TestSearchFindsTitleBodyAndObservationMatches(t *testing.T) {
	db := mustSearchDB(t)

	if _, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
		('n1', 'p1', 'alpha', 'alpha.md', 'Alpha Service', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
		('n2', 'p1', 'beta', 'beta.md', 'Beta Note', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
		('n3', 'p1', 'gamma', 'gamma.md', 'Gamma Note', 'h3', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`); err != nil {
		t.Fatalf("insert notes: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO note_tags(note_id, tag) VALUES
		('n1', 'frontmatter:django'),
		('n2', 'inline:auth'),
		('n3', 'observation:django')`); err != nil {
		t.Fatalf("insert tags: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notes_fts(rowid, note_id, title, body) VALUES
		((SELECT rowid FROM notes WHERE note_id='n1'),'n1','Alpha Service','searchable body text'),
		((SELECT rowid FROM notes WHERE note_id='n2'),'n2','Beta Note','body mentions queryterm'),
		((SELECT rowid FROM notes WHERE note_id='n3'),'n3','Gamma Note','observation category queryterm and value')`); err != nil {
		t.Fatalf("insert fts: %v", err)
	}

	titleHits, err := Search(db, "Alpha", 10, "")
	if err != nil {
		t.Fatalf("Search(title) error = %v", err)
	}
	if len(titleHits) != 1 || titleHits[0].NoteID != "n1" {
		t.Fatalf("Search(title) = %#v", titleHits)
	}

	bodyHits, err := Search(db, "queryterm", 10, "")
	if err != nil {
		t.Fatalf("Search(body) error = %v", err)
	}
	if len(bodyHits) != 2 {
		t.Fatalf("Search(body) len = %d, want 2", len(bodyHits))
	}

	observationHits, err := Search(db, "observation", 10, "")
	if err != nil {
		t.Fatalf("Search(observation) error = %v", err)
	}
	if len(observationHits) != 1 || observationHits[0].NoteID != "n3" {
		t.Fatalf("Search(observation) = %#v", observationHits)
	}

	tagHits, err := Search(db, "queryterm", 10, "django")
	if err != nil {
		t.Fatalf("Search(tag) error = %v", err)
	}
	if len(tagHits) != 1 || tagHits[0].NoteID != "n3" {
		t.Fatalf("Search(tag) = %#v", tagHits)
	}

	emptyHits, err := Search(db, "queryterm", 10, "missing")
	if err != nil {
		t.Fatalf("Search(missing tag) error = %v", err)
	}
	if len(emptyHits) != 0 {
		t.Fatalf("Search(missing tag) len = %d, want 0", len(emptyHits))
	}
}

func TestSearchOrdersByScoreAndRespectsLimit(t *testing.T) {
	db := mustSearchDB(t)

	_, _ = db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
		('n1', 'p1', 'alpha', 'alpha.md', 'Query Term', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
		('n2', 'p1', 'beta', 'beta.md', 'Other', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO notes_fts(rowid, note_id, title, body) VALUES
		((SELECT rowid FROM notes WHERE note_id='n1'),'n1','Query Term','queryterm queryterm queryterm'),
		((SELECT rowid FROM notes WHERE note_id='n2'),'n2','Other','queryterm')`)

	hits, err := Search(db, "queryterm", 1, "")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(hits))
	}
}

func mustSearchDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := index.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	return db
}
