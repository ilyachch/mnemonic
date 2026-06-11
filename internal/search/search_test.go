package search

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/index"
)

func TestSearchFindsTitleBodyAndObservationMatches(t *testing.T) {
	db := mustSearchDB(t)

	_, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
			('n1', 'p1', 'alpha', 'alpha.md', 'Alpha Service', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
			('n2', 'p1', 'beta', 'beta.md', 'Beta Note', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
			('n3', 'p1', 'gamma', 'gamma.md', 'Gamma Note', 'h3', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO note_tags(note_id, tag) VALUES
			('n1', 'frontmatter:django'),
			('n2', 'inline:auth'),
			('n3', 'observation:django')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO notes_fts(rowid, note_id, title, body) VALUES
			((SELECT rowid FROM notes WHERE note_id='n1'),'n1','Alpha Service','searchable body text'),
			((SELECT rowid FROM notes WHERE note_id='n2'),'n2','Beta Note','body mentions queryterm'),
			((SELECT rowid FROM notes WHERE note_id='n3'),'n3','Gamma Note','observation category queryterm and value')`)
	require.NoError(t, err)

	titleHits, err := Search(db, "Alpha", 10, "")
	require.NoError(t, err)
	require.Len(t, titleHits, 1)
	require.Equal(t, "n1", titleHits[0].NoteID)

	bodyHits, err := Search(db, "queryterm", 10, "")
	require.NoError(t, err)
	require.Len(t, bodyHits, 2)

	observationHits, err := Search(db, "observation", 10, "")
	require.NoError(t, err)
	require.Len(t, observationHits, 1)
	require.Equal(t, "n3", observationHits[0].NoteID)

	tagHits, err := Search(db, "queryterm", 10, "django")
	require.NoError(t, err)
	require.Len(t, tagHits, 1)
	require.Equal(t, "n3", tagHits[0].NoteID)

	emptyHits, err := Search(db, "queryterm", 10, "missing")
	require.NoError(t, err)
	require.Empty(t, emptyHits)
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
	require.NoError(t, err)
	require.Len(t, hits, 1)
}

func TestSearchSanitizesFTSOperators(t *testing.T) {
	db := mustSearchDB(t)

	_, _ = db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
			('n1', 'p1', 'alpha', 'alpha.md', 'Alpha Service', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
			('n2', 'p1', 'unicode', 'unicode.md', 'Заметка поиска', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO notes_fts(rowid, note_id, title, body) VALUES
			((SELECT rowid FROM notes WHERE note_id='n1'),'n1','Alpha Service','token with punctuation'),
			((SELECT rowid FROM notes WHERE note_id='n2'),'n2','Заметка поиска','токен с пунктуацией')`)

	hits, err := Search(db, `Alpha-Service:token*"punctuation"`, 10, "")
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "n1", hits[0].NoteID)

	unicodeHits, err := Search(db, `Заметка:"токен"`, 10, "")
	require.NoError(t, err)
	require.Len(t, unicodeHits, 1)
	require.Equal(t, "n2", unicodeHits[0].NoteID)
}

func mustSearchDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, index.ApplySchema(db))
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	return db
}