package graph

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/stretchr/testify/require"
)

func TestBacklinksReturnsResolvedLinksWithMetadata(t *testing.T) {
	db := mustGraphDB(t)

	_, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES
			('target', 'p1', 'target-note', 'target.md', 'Target Note', 'h-target', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
			('src1', 'p1', 'source-one', 'source1.md', 'Source One', 'h1', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z'),
			('src2', 'p1', 'source-two', 'source2.md', 'Source Two', 'h2', '2026-06-02T10:00:00Z', '2026-06-02T10:00:00Z')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, relation_type, source_line) VALUES
			('l1', 'src1', 'target', 'target-note', '', 3),
			('l2', 'src2', 'target', 'Target Note', 'depends_on', 8),
			('l3', 'src2', NULL, 'missing-note', 'relates_to', 12)`)
	require.NoError(t, err)

	links, err := Backlinks(db, "target")
	require.NoError(t, err)
	require.Len(t, links, 2)
	require.Equal(t, "src1", links[0].NoteID)
	require.Empty(t, links[0].RelationType)
	require.Equal(t, 3, links[0].SourceLine)
	require.Equal(t, "src2", links[1].NoteID)
	require.Equal(t, "depends_on", links[1].RelationType)
	require.Equal(t, 8, links[1].SourceLine)
}

func TestBacklinksRejectsNilDBAndEmptyTarget(t *testing.T) {
	_, err := Backlinks(nil, "target")
	require.Error(t, err)

	db := mustGraphDB(t)
	_, err = Backlinks(db, "")
	require.Error(t, err)
}

func mustGraphDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, index.ApplySchema(db))
	return db
}
