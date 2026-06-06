package registry

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestApplySchemaCreatesExpectedTablesAndConstraints(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	for _, table := range []string{"registry_meta", "projects", "project_locations", "project_status"} {
		if !tableExists(t, db, table) {
			t.Fatalf("table %q does not exist", table)
		}
	}

	if !columnIsPrimaryKey(t, db, "projects", "project_id") {
		t.Fatalf("projects.project_id is not primary key")
	}

	if _, err := db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p1", "backend", "backend", "local", "2026-06-02T10:00:00Z", "2026-06-02T10:00:00Z"); err != nil {
		t.Fatalf("insert project 1 failed: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p2", "frontend", "backend", "local", "2026-06-02T11:00:00Z", "2026-06-02T11:00:00Z"); err == nil {
		t.Fatal("expected active slug uniqueness violation")
	}

	if _, err := db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at, removed_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"p3", "archived", "backend", "detached", "2026-06-02T12:00:00Z", "2026-06-02T12:00:00Z", "2026-06-02T13:00:00Z"); err != nil {
		t.Fatalf("insert removed project with duplicate slug failed: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p4", "invalid", "invalid", "broken", "2026-06-02T14:00:00Z", "2026-06-02T14:00:00Z"); err == nil {
		t.Fatal("expected kind constraint violation")
	}

	if _, err := db.Exec(`INSERT INTO project_status (project_id, last_seen_at) VALUES (?, ?)`, "p1", "2026-06-02T10:00:00Z"); err != nil {
		t.Fatalf("insert status failed: %v", err)
	}

	var needsReindex int
	if err := db.QueryRow(`SELECT needs_reindex FROM project_status WHERE project_id = ?`, "p1").Scan(&needsReindex); err != nil {
		t.Fatalf("query needs_reindex failed: %v", err)
	}
	if needsReindex != 1 {
		t.Fatalf("needs_reindex = %d, want 1", needsReindex)
	}

	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "mnemonic", "registry.sqlite")); err != nil {
		t.Fatalf("registry file missing: %v", err)
	}

	var userVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatalf("PRAGMA user_version query failed: %v", err)
	}
	if userVersion != 1 {
		t.Fatalf("PRAGMA user_version = %d, want 1", userVersion)
	}
}

func TestApplySchemaIsIdempotent(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := ApplySchema(db); err != nil {
		t.Fatalf("first ApplySchema() error = %v", err)
	}
	if err := ApplySchema(db); err != nil {
		t.Fatalf("second ApplySchema() error = %v", err)
	}

	var userVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatalf("PRAGMA user_version query failed: %v", err)
	}
	if userVersion != 1 {
		t.Fatalf("PRAGMA user_version = %d, want 1", userVersion)
	}
}

func TestApplySchemaRejectsUnknownHigherVersion(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`PRAGMA user_version = 999`); err != nil {
		t.Fatalf("set user_version=999: %v", err)
	}

	err = ApplySchema(db)
	if err == nil {
		t.Fatal("ApplySchema() error = nil, want version rejection")
	}
	if err.Error() != "registry schema version 999 is newer than supported version 1" {
		t.Fatalf("ApplySchema() error = %q", err)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	return err == nil && name == table
}

func columnIsPrimaryKey(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) failed: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s) row failed: %v", table, err)
		}
		if name == column {
			return pk == 1
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s) failed: %v", table, err)
	}
	return false
}
