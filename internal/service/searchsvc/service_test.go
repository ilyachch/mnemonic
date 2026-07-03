package searchsvc

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchServiceSearch(t *testing.T) {
	svc := newSearchService(t)

	hits, err := svc.Search(context.Background(), SearchInput{Query: "queryterm!", Limit: 1, Tag: "django"})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "alpha", hits[0].Slug)

	allHits, err := svc.Search(context.Background(), SearchInput{Query: "queryterm!", Limit: 10})
	require.NoError(t, err)
	require.Len(t, allHits, 2)
}

func TestSearchServiceListTags(t *testing.T) {
	svc := newSearchService(t)

	out, err := svc.ListTags(context.Background())
	require.NoError(t, err)
	require.Equal(t, []ListTagsItem{
		{Tag: "go", Count: 2},
		{Tag: "django", Count: 1},
	}, out.Tags)
}

func TestSearchServiceBacklinks(t *testing.T) {
	svc := newSearchService(t)

	links, err := svc.Backlinks(context.Background(), BacklinksInput{Identifier: "gamma", Limit: 1})
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, "alpha", links[0].Slug)
}

func newSearchService(t *testing.T) Service {
	t.Helper()

	dir := t.TempDir()
	stateDir := t.TempDir()
	store := sqliteindex.Store{
		IndexPath: filepath.Join(dir, "index.sqlite"),
		RootDir:   dir,
		StateDir:  stateDir,
		KBID:      "kb-1",
	}

	db, err := store.Open()
	require.NoError(t, err)
	require.NoError(t, sqliteindex.ApplySchema(db))

	now := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	insertSearchNote(t, db, "alpha-id", "alpha", "Alpha", "alpha.md", "queryterm in alpha body", now, []string{"frontmatter:django", "frontmatter:go"})
	insertSearchNote(t, db, "beta-id", "beta", "Beta", "beta.md", "queryterm in beta body", now, []string{"frontmatter:go"})
	insertSearchNote(t, db, "gamma-id", "gamma", "Gamma", "gamma.md", "target body", now, nil)

	insertSearchLink(t, db, "link-alpha", "alpha-id", "gamma-id", "gamma", "wikilink", 12)
	insertSearchLink(t, db, "link-beta", "beta-id", "gamma-id", "gamma", "wikilink", 8)

	require.NoError(t, db.Close())

	svc := New(kb.KnowledgeBase{ID: "kb-1", RootDir: dir, StateDir: stateDir, IndexPath: filepath.Join(dir, "index.sqlite")})
	require.Equal(t, stateDir, svc.Index.StateDir)
	return *svc
}

func insertSearchNote(t *testing.T, db *sql.DB, noteID, slug, title, relPath, body string, now time.Time, tags []string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		noteID, "kb-1", slug, relPath, title, "hash-"+noteID, "", now.Unix(), now.Unix(),
	)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO notes_fts(rowid, note_id, title, summary, tags, aliases, body)
		 VALUES ((SELECT rowid FROM notes WHERE note_id = ?), ?, ?, ?, ?, ?, ?)`,
		noteID, noteID, title, "", "", "", body,
	)
	require.NoError(t, err)

	for _, tag := range tags {
		_, err := db.Exec(`INSERT INTO note_tags(note_id, tag) VALUES (?, ?)`, noteID, tag)
		require.NoError(t, err)
	}
}

func insertSearchLink(t *testing.T, db *sql.DB, linkID, noteID, toNoteID, target, sourceKind string, sourceLine int) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, is_resolved, is_ambiguous, source_line)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		linkID, noteID, toNoteID, target, "", "wiki", sourceKind, 1, 0, sourceLine,
	)
	require.NoError(t, err)
}

func TestAdvancedSearchMultiQuery(t *testing.T) {
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	restore := clock.SetClock(frozenClock{now: now})
	defer restore()

	svc := newSearchService(t)

	results, err := svc.AdvancedSearch(context.Background(), AdvancedSearchInput{
		Queries: []string{"queryterm!", "alpha"},
		Limit:   10,
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	slugs := make([]string, len(results))
	for i, r := range results {
		slugs[i] = r.Slug
	}
	assert.Contains(t, slugs, "alpha")
	assert.Contains(t, slugs, "beta")
}

func TestAdvancedSearchTimeFilter(t *testing.T) {
	now := time.Date(2026, time.June, 21, 13, 0, 0, 0, time.UTC)
	restore := clock.SetClock(frozenClock{now: now})
	defer restore()

	svc := newSearchService(t)

	results, err := svc.AdvancedSearch(context.Background(), AdvancedSearchInput{
		CreatedSince: "2h",
		Limit:        10,
	})
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestAdvancedSearchRelated(t *testing.T) {
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	restore := clock.SetClock(frozenClock{now: now})
	defer restore()

	svc := newSearchService(t)

	results, err := svc.AdvancedSearch(context.Background(), AdvancedSearchInput{
		Queries:        []string{"queryterm!"},
		Limit:          10,
		IncludeRelated: true,
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	for _, r := range results {
		if r.Slug == "alpha" {
			require.Len(t, r.RelatedNotes, 1)
			assert.Equal(t, "gamma-id", r.RelatedNotes[0].NoteID)
		}
	}
}

func TestAdvancedSearchInvalidDuration(t *testing.T) {
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	restore := clock.SetClock(frozenClock{now: now})
	defer restore()

	svc := newSearchService(t)

	_, err := svc.AdvancedSearch(context.Background(), AdvancedSearchInput{
		UpdatedSince: "10s",
		Limit:        10,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updated_since")
}

type frozenClock struct {
	now time.Time
}

func (f frozenClock) Now() time.Time { return f.now.UTC() }
func (f frozenClock) UUID() string   { return "00000000-0000-0000-0000-000000000000" }
