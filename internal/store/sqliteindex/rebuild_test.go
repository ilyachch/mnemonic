package sqliteindex

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashBytes(t *testing.T) {
	got := markdownstore.HashBytes([]byte("hello"))
	require.NotEmpty(t, got)
	assert.True(t, len(got) > 7, "expected sha256: prefix and hex")
	assert.True(t, got[:7] == "sha256:", "expected sha256: prefix, got %q", got[:7])

	// Deterministic
	got2 := markdownstore.HashBytes([]byte("hello"))
	assert.Equal(t, got, got2)

	// Different inputs produce different hashes
	got3 := markdownstore.HashBytes([]byte("world"))
	assert.NotEqual(t, got, got3)
}

func TestResolveLinkTarget_NotFound(t *testing.T) {
	aliasMap := make(map[string]struct {
		noteID string
		count  int
	})
	target, ok, ambiguous := resolveLinkTarget(nil, nil, aliasMap, "nonexistent")
	assert.False(t, ok)
	assert.False(t, ambiguous)
	assert.Empty(t, target)
}

func TestRebuildPopulatesTimestampsAndCreatedSinceSearchWorks(t *testing.T) {
	root := t.TempDir()
	stateDir := t.TempDir()
	indexPath := filepath.Join(stateDir, "index.sqlite")

	// Write a minimal-frontmatter note whose creation/update times must be
	// derived from the filesystem.
	notePath := filepath.Join(root, "recent.md")
	require.NoError(t, os.WriteFile(notePath, []byte("---\n"+
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440070\n"+
		"---\n"+
		"# Recent Note\n"+
		"Body content for the recent note.\n"), 0o644))

	// Set the file mtime to 1 minute ago so created_since "1h" matches.
	recent := time.Now().UTC().Add(-1 * time.Minute)
	require.NoError(t, os.Chtimes(notePath, recent, recent))

	store := Store{RootDir: root, StateDir: stateDir, IndexPath: indexPath, KBID: "test-kb"}

	result, err := store.Rebuild()
	require.NoError(t, err)
	require.Equal(t, "ok", result.Status)
	require.Equal(t, 1, result.NotesIndexed)

	// Verify the notes table has non-zero timestamps derived from the file.
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var createdAt, updatedAt int64
	require.NoError(t, db.QueryRow(`SELECT created_at, updated_at FROM notes WHERE note_id = ?`,
		"550e8400-e29b-41d4-a716-446655440070").Scan(&createdAt, &updatedAt))
	assert.Greater(t, createdAt, int64(0), "created_at must be populated")
	assert.Greater(t, updatedAt, int64(0), "updated_at must be populated")

	// Search with created_since "1h" — the note was created ~1 minute ago so
	// it must be found.
	hits, err := store.SearchAdvanced(db, SearchOptions{
		Queries:      []string{"recent"},
		CreatedSince: "1h",
		Limit:        10,
	}, time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "recent-note", hits[0].Slug)
	assert.Equal(t, "Recent Note", hits[0].Title)
}

func TestRebuildPopulatesTimestampsFromLegacyFrontmatter(t *testing.T) {
	root := t.TempDir()
	stateDir := t.TempDir()
	indexPath := filepath.Join(stateDir, "index.sqlite")

	// A legacy note that still carries explicit YAML timestamps.
	notePath := filepath.Join(root, "legacy.md")
	require.NoError(t, os.WriteFile(notePath, []byte("---\n"+
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440071\n"+
		"title: Legacy Note\n"+
		"slug: legacy-note\n"+
		"created_at: 1741737600\n"+
		"updated_at: 1741824000\n"+
		"---\n"+
		"# Legacy Note\n"), 0o644))

	store := Store{RootDir: root, StateDir: stateDir, IndexPath: indexPath, KBID: "test-kb"}
	_, err := store.Rebuild()
	require.NoError(t, err)

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var createdAt, updatedAt int64
	require.NoError(t, db.QueryRow(`SELECT created_at, updated_at FROM notes WHERE slug = ?`,
		"legacy-note").Scan(&createdAt, &updatedAt))
	assert.Equal(t, int64(1741737600), createdAt, "created_at should come from YAML frontmatter")
	assert.Equal(t, int64(1741824000), updatedAt, "updated_at should come from YAML frontmatter")
}
