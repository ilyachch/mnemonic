package index

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplySchemaCreatesExpectedTables(t *testing.T) {
	db := mustTestDB(t)
	require.NoError(t, ApplySchema(db))

	var version int
	require.NoError(t, db.QueryRow(`PRAGMA user_version`).Scan(&version))
	require.Equal(t, 1, version)

	for _, table := range []string{"meta", "notes", "note_tags", "observations", "links", "index_runs", "notes_fts"} {
		require.True(t, tableExists(t, db, table), "table %s missing", table)
	}
	require.True(t, ftsTableUsesTokenizer(t, db, "notes_fts", "unicode61"), "notes_fts does not use unicode61 tokenizer")

	require.True(t, columnIsUnique(t, db, "notes", "slug"), "notes.slug is not unique")
	require.True(t, columnIsUnique(t, db, "notes", "rel_path"), "notes.rel_path is not unique")
	require.True(t, columnAllowsNull(t, db, "links", "to_note_id"), "links.to_note_id is not nullable")
}

func TestCheckSchemaStatus(t *testing.T) {
	db := mustTestDB(t)
	_, err := db.Exec(`PRAGMA user_version = 0`)
	require.NoError(t, err)

	status, err := CheckSchemaStatus(db)
	require.NoError(t, err)
	require.Equal(t, SchemaStatusNeedsRebuild, status)
}

func mustTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(sqliteDriverName, "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	return db
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name)
	return err == nil && name == table
}

func columnIsUnique(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA index_list(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string
		require.NoError(t, rows.Scan(&seq, &name, &unique, &origin, &partial))
		if unique != 1 {
			continue
		}
		cols, err := db.Query(`PRAGMA index_info(` + name + `)`)
		require.NoError(t, err)
		for cols.Next() {
			var seqno, cid int
			var colName string
			require.NoError(t, cols.Scan(&seqno, &cid, &colName))
			if colName == column {
				cols.Close()
				return true
			}
		}
		cols.Close()
	}
	return false
}

func columnAllowsNull(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		require.NoError(t, rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk))
		if name == column {
			return notnull == 0
		}
	}
	return false
}

func ftsTableUsesTokenizer(t *testing.T, db *sql.DB, table, tokenizer string) bool {
	t.Helper()

	var sqlText string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&sqlText)
	require.NoError(t, err)
	return sqlText != "" && strings.Contains(sqlText, "tokenize = '"+tokenizer+"'")
}