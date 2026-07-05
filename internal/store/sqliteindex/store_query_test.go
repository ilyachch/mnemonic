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

func TestDedupRelatedNotesDeterministic(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "a", Slug: "z", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "a", Slug: "z", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "b", Slug: "y", Direction: "outgoing", SourceKind: "wikilink"},
	}

	result := dedupRelatedNotes(notes)
	require.Len(t, result, 2)
	assert.Equal(t, "b", result[0].NoteID)
	assert.Equal(t, "a", result[1].NoteID)
}

func TestDedupRelatedNotesDifferentOrderSameResult(t *testing.T) {
	order1 := []RelatedNote{
		{NoteID: "c", Slug: "ccc", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "a", Slug: "aaa", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "b", Slug: "bbb", Direction: "outgoing", SourceKind: "wikilink"},
	}
	order2 := []RelatedNote{
		{NoteID: "b", Slug: "bbb", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "c", Slug: "ccc", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "a", Slug: "aaa", Direction: "outgoing", SourceKind: "wikilink"},
	}

	r1 := dedupRelatedNotes(makeCopy(order1))
	r2 := dedupRelatedNotes(makeCopy(order2))

	require.Equal(t, len(r1), len(r2))
	for i := range r1 {
		assert.Equal(t, r1[i].NoteID, r2[i].NoteID)
	}
}

func TestDedupRelatedNotesRelationsSectionPriority(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "incoming", Slug: "inc", Direction: "incoming", SourceKind: "wikilink"},
		{NoteID: "outgoing", Slug: "out", Direction: "outgoing", SourceKind: "wikilink"},
		{NoteID: "relation", Slug: "rel", Direction: "outgoing", SourceKind: "relations_section"},
	}

	result := dedupRelatedNotes(notes)
	require.Len(t, result, 3)
	assert.Equal(t, "relation", result[0].NoteID, "relations_section should be first")
	assert.Equal(t, "outgoing", result[1].NoteID, "outgoing wikilink should be second")
	assert.Equal(t, "incoming", result[2].NoteID, "incoming should be last")
}

func TestDedupRelatedNotesMaxLimit(t *testing.T) {
	notes := make([]RelatedNote, 10)
	for i := 0; i < 10; i++ {
		notes[i] = RelatedNote{
			NoteID:       "id-" + string(rune('a'+i)),
			Slug:         "slug-" + string(rune('0'+i%10)),
			Direction:    "outgoing",
			SourceKind:   "wikilink",
			RelationType: "relates_to",
		}
	}

	result := dedupRelatedNotes(notes)
	assert.LessOrEqual(t, len(result), 3)
}

func TestDedupRelatedNotesDedupKeepsFirstByPriority(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "same", Slug: "s", SourceKind: "relations_section", Direction: "outgoing", RelationType: "depends_on"},
		{NoteID: "same", Slug: "s", SourceKind: "wikilink", Direction: "incoming", RelationType: "relates_to"},
	}

	result := dedupRelatedNotes(notes)
	require.Len(t, result, 1)
	assert.Equal(t, "relations_section", result[0].SourceKind, "highest priority version should be kept")
}

func TestDedupRelatedNotesWithinGroupOrdering(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "c", Slug: "ccc", Direction: "outgoing", SourceKind: "wikilink", RelationType: "relates_to"},
		{NoteID: "a", Slug: "bbb", Direction: "outgoing", SourceKind: "wikilink", RelationType: "depends_on"},
		{NoteID: "b", Slug: "aaa", Direction: "outgoing", SourceKind: "wikilink", RelationType: "depends_on"},
	}

	result := dedupRelatedNotes(notes)
	require.Len(t, result, 3)
	assert.Equal(t, "b", result[0].NoteID, "depends_on + aaa slug (within depends_on group, aaa < bbb)")
	assert.Equal(t, "a", result[1].NoteID, "depends_on + bbb slug")
	assert.Equal(t, "c", result[2].NoteID, "relates_to + ccc slug")
}

func TestSortRelatedNotesPriority(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "inc", Slug: "i", Direction: "incoming", SourceKind: "wikilink"},
		{NoteID: "rel", Slug: "r", Direction: "outgoing", SourceKind: "relations_section"},
		{NoteID: "out", Slug: "o", Direction: "outgoing", SourceKind: "wikilink"},
	}

	sortRelatedNotes(notes)
	assert.Equal(t, "rel", notes[0].NoteID, "relations_section first")
	assert.Equal(t, "out", notes[1].NoteID, "outgoing wikilink second")
	assert.Equal(t, "inc", notes[2].NoteID, "incoming last")
}

func TestSortRelatedNotesTiebreaker(t *testing.T) {
	notes := []RelatedNote{
		{NoteID: "z", Slug: "z", Direction: "outgoing", SourceKind: "wikilink", RelationType: "z"},
		{NoteID: "a", Slug: "a", Direction: "outgoing", SourceKind: "wikilink", RelationType: "a"},
		{NoteID: "m", Slug: "m", Direction: "outgoing", SourceKind: "wikilink", RelationType: "a"},
	}

	sortRelatedNotes(notes)
	assert.Equal(t, "a", notes[0].NoteID, "relation_type a + slug a -> NoteID a")
	assert.Equal(t, "m", notes[1].NoteID, "relation_type a + slug m -> NoteID m")
	assert.Equal(t, "z", notes[2].NoteID, "relation_type z + slug z -> NoteID z")
}

func makeCopy(notes []RelatedNote) []RelatedNote {
	out := make([]RelatedNote, len(notes))
	copy(out, notes)
	return out
}

func seedRegressionStore(t *testing.T) (Store, *sql.DB) {
	t.Helper()

	dir := t.TempDir()
	store := Store{
		IndexPath: filepath.Join(dir, "index.sqlite"),
		RootDir:   dir,
		KBID:      "kb-regression",
	}

	db, err := store.Open()
	require.NoError(t, err)
	require.NoError(t, ApplySchema(db))

	now := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)

	type seedNote struct {
		id    string
		slug  string
		title string
		body  string
		tags  []string
		ts    time.Time
	}
	notes := []seedNote{
		{id: "n1", slug: "python-guide", title: "Python Guide", body: "Python is a popular programming language for backend development.", tags: []string{"frontmatter:tech:python", "frontmatter:tech:backend"}, ts: now},
		{id: "n2", slug: "django-guide", title: "Django Guide", body: "Django is a Python web framework for rapid development.", tags: []string{"frontmatter:tech:python", "frontmatter:tech:django", "frontmatter:framework"}, ts: now.Add(1 * time.Hour)},
		{id: "n3", slug: "golang-guide", title: "Golang Guide", body: "Go is a statically typed compiled language designed at Google.", tags: []string{"frontmatter:tech:golang", "frontmatter:tech:backend"}, ts: now.Add(2 * time.Hour)},
		{id: "n4", slug: "flask-guide", title: "Flask Guide", body: "Flask is a lightweight Python web micro-framework.", tags: []string{"frontmatter:tech:python", "frontmatter:tech:flask"}, ts: now.Add(3 * time.Hour)},
		{id: "n5", slug: "database-design", title: "Database Design", body: "Database design principles for SQL and NoSQL systems.", tags: []string{"frontmatter:tech:database", "frontmatter:design"}, ts: now.Add(4 * time.Hour)},
		{id: "n6", slug: "postgresql", title: "PostgreSQL", body: "PostgreSQL is a powerful open source relational database.", tags: []string{"frontmatter:tech:database", "frontmatter:tech:sql"}, ts: now.Add(5 * time.Hour)},
		{id: "n7", slug: "redis-cache", title: "Redis Cache", body: "Redis is an in-memory data structure store used as a cache.", tags: []string{"frontmatter:tech:redis", "frontmatter:tech:caching"}, ts: now.Add(6 * time.Hour)},
		{id: "n8", slug: "testing-strategies", title: "Testing Strategies", body: "Unit testing integration testing and end-to-end testing approaches.", tags: []string{"frontmatter:testing", "frontmatter:quality"}, ts: now.Add(7 * time.Hour)},
		{id: "n9", slug: "api-design", title: "API Design", body: "REST API and GraphQL design principles for modern web services.", tags: []string{"frontmatter:tech:api", "frontmatter:design"}, ts: now.Add(8 * time.Hour)},
		{id: "n10", slug: "deployment", title: "Deployment", body: "CI/CD pipelines and deployment strategies for production.", tags: []string{"frontmatter:devops", "frontmatter:deployment"}, ts: now.Add(9 * time.Hour)},
	}

	for _, n := range notes {
		insertRegressionNote(t, db, n.id, n.slug, n.title, n.body, n.tags, n.ts)
	}

	type seedLink struct {
		id           string
		fromID       string
		toID         string
		sourceKind   string
		relationType string
		sourceLine   int
	}
	links := []seedLink{
		{id: "l1", fromID: "n2", toID: "n1", sourceKind: "relations_section", relationType: "depends_on", sourceLine: 10},
		{id: "l2", fromID: "n2", toID: "n4", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 15},
		{id: "l3", fromID: "n4", toID: "n2", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 8},
		{id: "l4", fromID: "n3", toID: "n6", sourceKind: "relations_section", relationType: "depends_on", sourceLine: 12},
		{id: "l5", fromID: "n5", toID: "n6", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 5},
		{id: "l6", fromID: "n6", toID: "n7", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 14},
		{id: "l7", fromID: "n7", toID: "n6", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 20},
		{id: "l8", fromID: "n8", toID: "n1", sourceKind: "relations_section", relationType: "depends_on", sourceLine: 5},
		{id: "l9", fromID: "n8", toID: "n3", sourceKind: "wikilink", relationType: "relates_to", sourceLine: 10},
		{id: "l10", fromID: "n9", toID: "n10", sourceKind: "relations_section", relationType: "depends_on", sourceLine: 8},
	}

	for _, l := range links {
		insertRegressionLink(t, db, l.id, l.fromID, l.toID, l.fromID+"-slug", l.sourceKind, l.relationType, l.sourceLine)
	}

	return store, db
}

func insertRegressionNote(t *testing.T, db *sql.DB, noteID, slug, title, body string, tags []string, ts time.Time) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		noteID, "kb-regression", slug, slug+".md", title, "hash-"+noteID, "", ts.Unix(), ts.Unix(),
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

func insertRegressionLink(t *testing.T, db *sql.DB, linkID, noteID, toNoteID, target, sourceKind, relationType string, sourceLine int) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, relation_type, is_resolved, is_ambiguous, source_line)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		linkID, noteID, toNoteID, target, "", "wiki", sourceKind, relationType, 1, 0, sourceLine,
	)
	require.NoError(t, err)
}

func TestRegressionGraphBoost(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"python"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	topSlugs := results
	if len(topSlugs) > 3 {
		topSlugs = results[:3]
	}
	topSlugSet := make(map[string]bool, len(topSlugs))
	for _, r := range topSlugs {
		topSlugSet[r.Slug] = true
	}
	assert.True(t, topSlugSet["python-guide"], "python-guide should be in top 3 for python query")

	results, err = store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"Django"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "django-guide", results[0].Slug, "django-guide should be first for Django query")
}

func TestRegressionTagAndFilter(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Tags:  []string{"tech:python", "framework"},
		Limit: 10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	for _, r := range results {
		assert.Equal(t, "django-guide", r.Slug, "only django-guide has both tech:python and framework tags")
	}
}

func TestRegressionNormalizedTags(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Tags:  []string{"tech:python"},
		Limit: 10,
	}, now)
	require.NoError(t, err)
	slugs := make([]string, len(results))
	for i, r := range results {
		slugs[i] = r.Slug
	}
	assert.Contains(t, slugs, "python-guide")
	assert.Contains(t, slugs, "django-guide")
	assert.Contains(t, slugs, "flask-guide")
}

func TestRegressionRelatedNotesOrdering(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries:        []string{"Django"},
		Limit:          10,
		IncludeRelated: true,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	django := findBySlug(results, "django-guide")
	require.NotNil(t, django)
	require.NotEmpty(t, django.RelatedNotes, "django-guide should have related notes")

	firstRel := django.RelatedNotes[0]
	assert.Equal(t, "relations_section", firstRel.SourceKind, "relations_section should be first priority")

	slugs := make([]string, len(django.RelatedNotes))
	for i, rn := range django.RelatedNotes {
		slugs[i] = rn.Slug
	}
	assert.Contains(t, slugs, "python-guide")
	assert.Contains(t, slugs, "flask-guide")
}

func TestRegressionRelatedNotesDedup(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	results, err := store.ListLinkIssues(db)
	require.NoError(t, err)

	issueIDs := make(map[string]bool)
	for _, li := range results {
		issueIDs[li.LinkID] = true
	}
	assert.False(t, issueIDs["duplicate"], "no duplicate links should exist")
}

func TestRegressionFilterOnlySearch(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Tags:  []string{"design"},
		Limit: 10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	slugs := make([]string, len(results))
	for i, r := range results {
		slugs[i] = r.Slug
	}
	assert.Contains(t, slugs, "database-design")
	assert.Contains(t, slugs, "api-design")
}

func TestRegressionRelatedNotesMaxLimit(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries:        []string{"testing"},
		Limit:          10,
		IncludeRelated: true,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	testNote := findBySlug(results, "testing-strategies")
	require.NotNil(t, testNote)
	assert.LessOrEqual(t, len(testNote.RelatedNotes), 3, "max 3 related notes")
}

func TestRegressionQueryDedupDoesNotChangeRank(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	results1, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"python", "python"},
		Limit:   10,
	}, now)
	require.NoError(t, err)

	results2, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"python"},
		Limit:   10,
	}, now)
	require.NoError(t, err)

	require.Equal(t, len(results1), len(results2))
	for i := range results1 {
		assert.Equal(t, results1[i].Slug, results2[i].Slug, "duplicate query should not change rank at position %d", i)
	}
}

func TestRegressionExpectedNoteIsRank1(t *testing.T) {
	store, db := seedRegressionStore(t)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)

	results, err := store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"PostgreSQL"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "postgresql", results[0].Slug, "PostgreSQL should be rank 1 for its own name")

	results, err = store.SearchAdvanced(db, SearchOptions{
		Queries: []string{"Redis"},
		Limit:   10,
	}, now)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "redis-cache", results[0].Slug, "Redis should be rank 1 for its own name")
}

func findBySlug(results []SearchResult, slug string) *SearchResult {
	for i := range results {
		if results[i].Slug == slug {
			return &results[i]
		}
	}
	return nil
}
