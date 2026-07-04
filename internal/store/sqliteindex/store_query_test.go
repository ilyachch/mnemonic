package sqliteindex

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/stretchr/testify/assert"
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

func TestCountUnresolvedLinks(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	_, err := db.Exec(
		`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, relation_type, is_resolved, is_ambiguous, source_line)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"link-unresolved", "gamma-id", nil, "missing-target", "", "", "wikilink", "", 0, 0, 4,
	)
	require.NoError(t, err)

	count, err := store.CountUnresolvedLinks(db)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestParseRelativeDurationValid(t *testing.T) {
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		input    string
		expected int64
	}{
		{"15m", now.Add(-15 * time.Minute).Unix()},
		{"2h", now.Add(-2 * time.Hour).Unix()},
		{"30d", now.Add(-30 * 24 * time.Hour).Unix()},
		{"1m", now.Add(-1 * time.Minute).Unix()},
		{"24h", now.Add(-24 * time.Hour).Unix()},
		{"1d", now.Add(-24 * time.Hour).Unix()},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ts, err := parseRelativeDuration(tt.input, now)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, ts)
		})
	}
}

func TestParseRelativeDurationInvalid(t *testing.T) {
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)

	invalid := []string{"abc", "10s", "1y", "-5m", "0d", "m", "5"}
	for _, input := range invalid {
		t.Run("invalid_"+input, func(t *testing.T) {
			_, err := parseRelativeDuration(input, now)
			require.Error(t, err)
			var appErr *apperr.Error
			require.True(t, errors.As(err, &appErr))
		})
	}
}

func TestSearchAdvancedMultiQuery(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"queryterm!", "alpha"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 2)

	alphaIdx := 0
	betaIdx := 1
	if results[0].Slug != "alpha" {
		alphaIdx, betaIdx = 1, 0
	}
	assert.Equal(t, "alpha", results[alphaIdx].Slug)
	assert.True(t, results[alphaIdx].MatchCount >= 1)
	assert.Equal(t, "beta", results[betaIdx].Slug)
	assert.Equal(t, 1, results[betaIdx].MatchCount)
}

func TestSearchAdvancedTimeFilterNoQueries(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	createdAt := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	now := createdAt.Add(1 * time.Hour)

	results, err := store.SearchAdvanced(db, SearchOptions{
		CreatedSince: "2h",
		Limit:        10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 3)
}

func TestSearchAdvancedTimeFilterNoResults(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	createdAt := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	now := createdAt.Add(48 * time.Hour)

	results, err := store.SearchAdvanced(db, SearchOptions{
		CreatedBefore: ptr(createdAt.Unix()),
		Limit:         10,
	}, now)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestSearchAdvancedGraphRerank(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"queryterm!"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.GreaterOrEqual(t, results[0].Score, results[1].Score)
}

func TestSearchAdvancedRelatedNotes(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries:        []string{"queryterm!"},
		Limit:          10,
		IncludeRelated: true,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, r := range results {
		if r.Slug == "alpha" {
			require.Len(t, r.RelatedNotes, 1)
			assert.Equal(t, "gamma-id", r.RelatedNotes[0].NoteID)
			assert.Equal(t, "wikilink", r.RelatedNotes[0].SourceKind)
			assert.Equal(t, "outgoing", r.RelatedNotes[0].Direction)
		}
	}
}

func TestSearchAdvancedRelatedNotesFalse(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries:        []string{"queryterm!"},
		Limit:          10,
		IncludeRelated: false,
	}, now)
	require.NoError(t, err)
	for _, r := range results {
		assert.Nil(t, r.RelatedNotes)
	}
}

func TestSearchAdvancedNoFilterError(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	_, err := store.SearchAdvanced(db, SearchOptions{Limit: 10}, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one query")
}

func TestSearchAdvancedInvalidDuration(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	_, err := store.SearchAdvanced(db, SearchOptions{
		CreatedSince: "abc",
		Limit:        10,
	}, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "created_since")
}

func TestSearchAdvancedTagFilter(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"queryterm!"},
		Tags:    []string{"go", "django"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "alpha", results[0].Slug)
}

func TestSearchAdvancedAbsoluteTimeBounds(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	createdAt := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)

	after := createdAt.Add(-1 * time.Hour).Unix()
	before := createdAt.Add(1 * time.Hour).Unix()

	results, err := store.SearchAdvanced(db, SearchOptions{
		CreatedAfter:  &after,
		CreatedBefore: &before,
		Limit:         10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 3)
}

func TestSearchAdvancedUpdatedBounds(t *testing.T) {
	store, db := seedQueryStore(t)
	t.Cleanup(func() { _ = db.Close() })

	updatedAt := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	now := updatedAt.Add(48 * time.Hour)

	after := updatedAt.Add(-1 * time.Hour).Unix()
	before := updatedAt.Add(1 * time.Hour).Unix()

	results, err := store.SearchAdvanced(db, SearchOptions{
		UpdatedAfter:  &after,
		UpdatedBefore: &before,
		Limit:         10,
	}, now)
	require.NoError(t, err)
	require.Len(t, results, 3)
}

func ptr[T any](v T) *T {
	return &v
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
	require.NoError(t, ApplySchema(db))

	now := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)
	insertQueryNote(t, db, "alpha-id", "alpha", "Alpha", "alpha.md", "queryterm in alpha body", now, []string{"frontmatter:django", "frontmatter:go"})
	insertQueryNote(t, db, "beta-id", "beta", "Beta", "beta.md", "queryterm in beta body", now, []string{"frontmatter:go"})
	insertQueryNote(t, db, "gamma-id", "gamma", "Gamma", "gamma.md", "target body", now, nil)

	insertQueryLink(t, db, "link-alpha", "alpha-id", "gamma-id", "gamma", "wikilink", "", 12)
	insertQueryLink(t, db, "link-beta", "beta-id", "gamma-id", "gamma", "wikilink", "", 8)

	return store, db
}

func insertQueryNote(t *testing.T, db *sql.DB, noteID, slug, title, relPath, body string, now time.Time, tags []string) {
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

func insertQueryLink(t *testing.T, db *sql.DB, linkID, noteID, toNoteID, target, sourceKind, relationType string, sourceLine int) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, relation_type, is_resolved, is_ambiguous, source_line)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		linkID, noteID, toNoteID, target, "", "wiki", sourceKind, relationType, 1, 0, sourceLine,
	)
	require.NoError(t, err)
}
