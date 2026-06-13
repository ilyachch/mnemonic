package registry

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestApplySchemaCreatesExpectedTablesAndConstraints(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = ApplySchema(db)
	require.NoError(t, err)

	for _, table := range []string{"registry_meta", "projects", "project_locations", "project_status"} {
		require.True(t, tableExists(t, db, table), "table %q does not exist", table)
	}

	require.True(t, columnIsPrimaryKey(t, db, "projects", "project_id"), "projects.project_id is not primary key")

	_, err = db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p1", "backend", "backend", "local", "2026-06-02T10:00:00Z", "2026-06-02T10:00:00Z")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p2", "frontend", "backend", "local", "2026-06-02T11:00:00Z", "2026-06-02T11:00:00Z")
	require.Error(t, err, "expected active slug uniqueness violation")

	_, err = db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at, removed_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"p3", "archived", "backend", "detached", "2026-06-02T12:00:00Z", "2026-06-02T12:00:00Z", "2026-06-02T13:00:00Z")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"p4", "invalid", "invalid", "broken", "2026-06-02T14:00:00Z", "2026-06-02T14:00:00Z")
	require.Error(t, err, "expected kind constraint violation")

	_, err = db.Exec(`INSERT INTO project_status (project_id, last_seen_at) VALUES (?, ?)`, "p1", "2026-06-02T10:00:00Z")
	require.NoError(t, err)

	var needsReindex int
	err = db.QueryRow(`SELECT needs_reindex FROM project_status WHERE project_id = ?`, "p1").Scan(&needsReindex)
	require.NoError(t, err)
	require.Equal(t, 1, needsReindex)

	_, err = os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "mnemonic", "registry.sqlite"))
	require.NoError(t, err, "registry file missing")

	var userVersion int
	err = db.QueryRow(`PRAGMA user_version`).Scan(&userVersion)
	require.NoError(t, err)
	require.Equal(t, 1, userVersion)
}

func TestApplySchemaIsIdempotent(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = ApplySchema(db)
	require.NoError(t, err)
	err = ApplySchema(db)
	require.NoError(t, err)

	var userVersion int
	err = db.QueryRow(`PRAGMA user_version`).Scan(&userVersion)
	require.NoError(t, err)
	require.Equal(t, 1, userVersion)
}

func TestApplySchemaRejectsUnknownHigherVersion(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`PRAGMA user_version = 999`)
	require.NoError(t, err)

	err = ApplySchema(db)
	require.Error(t, err)
	require.EqualError(t, err, "registry schema version 999 is newer than supported version 1")
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
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		require.NoError(t, err)
		if name == column {
			return pk == 1
		}
	}
	require.NoError(t, rows.Err())
	return false
}
