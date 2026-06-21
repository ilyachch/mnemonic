package sqliteindex

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/stretchr/testify/require"
)

func TestSearchFiltersAndLimits(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	hits, err := store.Search(db, "queryterm!", 1, "django")
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "alpha", hits[0].Slug)

	allHits, err := store.Search(db, "queryterm!", 10, "")
	require.NoError(t, err)
	require.Len(t, allHits, 2)
}

func TestListTagsAggregatesCounts(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	tags, err := store.ListTags(db)
	require.NoError(t, err)
	require.Equal(t, []TagCount{
		{Tag: "go", Count: 2},
		{Tag: "django", Count: 1},
	}, tags)
}

func TestLookupNoteAndBacklinks(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	note, err := store.LookupNoteByIdentifier(db, "gamma")
	require.NoError(t, err)
	require.Equal(t, "gamma-id", note.NoteID)
	require.Equal(t, "gamma", note.Slug)
	require.Equal(t, "Gamma", note.Title)
	require.Equal(t, "gamma.md", note.Path)

	links, err := store.Backlinks(db, note.NoteID, 1)
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, "alpha", links[0].Slug)
}

func seedQueryStore(t *testing.T) (Store, *sql.DB) {
	t.Helper()

	dir := t.TempDir()
	store := Store{
		IndexPath: filepath.Join(dir, "index.sqlite"),
		RootDir:   dir,
		KBID:      "kb-1",
	}

	db, err := store.Open()
	require.NoError(t, err)
	require.NoError(t, index.ApplySchema(db))

	now := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	insertQueryNote(t, db, "alpha-id", "alpha", "Alpha", "alpha.md", "queryterm in alpha body", now, []string{"frontmatter:django", "frontmatter:go"})
	insertQueryNote(t, db, "beta-id", "beta", "Beta", "beta.md", "queryterm in beta body", now, []string{"frontmatter:go"})
	insertQueryNote(t, db, "gamma-id", "gamma", "Gamma", "gamma.md", "target body", now, nil)

	insertQueryLink(t, db, "link-alpha", "alpha-id", "gamma-id", "gamma", "wikilink", 12)
	insertQueryLink(t, db, "link-beta", "beta-id", "gamma-id", "gamma", "wikilink", 8)

	return store, db
}

func insertQueryNote(t *testing.T, db *sql.DB, noteID, slug, title, relPath, body string, now time.Time, tags []string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		noteID, "kb-1", slug, relPath, title, "hash-"+noteID, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO notes_fts(rowid, note_id, title, body)
		 VALUES ((SELECT rowid FROM notes WHERE note_id = ?), ?, ?, ?)`,
		noteID, noteID, title, body,
	)
	require.NoError(t, err)

	for _, tag := range tags {
		_, err := db.Exec(`INSERT INTO note_tags(note_id, tag) VALUES (?, ?)`, noteID, tag)
		require.NoError(t, err)
	}
}

func insertQueryLink(t *testing.T, db *sql.DB, linkID, noteID, toNoteID, target, relationType string, sourceLine int) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO links(link_id, note_id, to_note_id, target, relation_type, source_line)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		linkID, noteID, toNoteID, target, relationType, sourceLine,
	)
	require.NoError(t, err)
}
