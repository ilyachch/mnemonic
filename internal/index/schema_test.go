package index

import (
	"database/sql"
	"strings"
	"testing"
)

func TestApplySchemaCreatesExpectedTables(t *testing.T) {
	db := mustTestDB(t)
	if err := ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("PRAGMA user_version query failed: %v", err)
	}
	if version != 1 {
		t.Fatalf("PRAGMA user_version = %d, want 1", version)
	}

	for _, table := range []string{"meta", "notes", "note_tags", "observations", "links", "index_runs", "notes_fts"} {
		if !tableExists(t, db, table) {
			t.Fatalf("table %s missing", table)
		}
	}
	if !ftsTableUsesTokenizer(t, db, "notes_fts", "unicode61") {
		t.Fatal("notes_fts does not use unicode61 tokenizer")
	}

	if !columnIsUnique(t, db, "notes", "slug") {
		t.Fatal("notes.slug is not unique")
	}
	if !columnIsUnique(t, db, "notes", "rel_path") {
		t.Fatal("notes.rel_path is not unique")
	}
	if !columnAllowsNull(t, db, "links", "to_note_id") {
		t.Fatal("links.to_note_id is not nullable")
	}
}

func TestCheckSchemaStatus(t *testing.T) {
	db := mustTestDB(t)
	if _, err := db.Exec(`PRAGMA user_version = 0`); err != nil {
		t.Fatalf("set user_version=0: %v", err)
	}

	status, err := CheckSchemaStatus(db)
	if err != nil {
		t.Fatalf("CheckSchemaStatus() error = %v", err)
	}
	if status != SchemaStatusNeedsRebuild {
		t.Fatalf("status = %q, want needs_rebuild", status)
	}
}

func mustTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(sqliteDriverName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
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
	if err != nil {
		t.Fatalf("PRAGMA index_list(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index_list: %v", err)
		}
		if unique != 1 {
			continue
		}
		cols, err := db.Query(`PRAGMA index_info(` + name + `)`)
		if err != nil {
			t.Fatalf("PRAGMA index_info(%s): %v", name, err)
		}
		for cols.Next() {
			var seqno, cid int
			var colName string
			if err := cols.Scan(&seqno, &cid, &colName); err != nil {
				t.Fatalf("scan index_info: %v", err)
			}
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
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
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
	if err != nil {
		t.Fatalf("sqlite_master lookup for %s: %v", table, err)
	}
	return sqlText != "" && strings.Contains(sqlText, "tokenize = '"+tokenizer+"'")
}
