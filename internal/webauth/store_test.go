package webauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestResolveDBPathPrecedence(t *testing.T) {
	root := testutil.CleanEnvForTest(t)

	t.Setenv("MNEMONIC_WEB_AUTH_DB", "/env/auth.sqlite")
	got, err := ResolveDBPath("/flag/auth.sqlite")
	require.NoError(t, err)
	require.Equal(t, "/env/auth.sqlite", got)

	t.Setenv("MNEMONIC_WEB_AUTH_DB", "")
	got, err = ResolveDBPath("/flag/auth.sqlite")
	require.NoError(t, err)
	require.Equal(t, "/flag/auth.sqlite", got)

	got, err = ResolveDBPath("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "state", "mnemonic", "web_auth.sqlite"), got)
}

func TestNewStoreAppliesSchema(t *testing.T) {
	testutil.CleanEnvForTest(t)

	path := filepath.Join(t.TempDir(), "nested", "web_auth.sqlite")
	store, err := NewStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	db := store.db
	require.True(t, tableExists(t, db, "users"))
	require.True(t, tableExists(t, db, "permissions"))

	usersSQL := sqliteTableSQL(t, db, "users")
	require.Contains(t, usersSQL, "username TEXT NOT NULL UNIQUE")
	require.Contains(t, usersSQL, "token_hash TEXT NOT NULL UNIQUE")

	fk := foreignKeyList(t, db, "permissions")
	require.Contains(t, fk, "users.user_id")
}

func TestCreateUserHashesTokenAndValidatesToken(t *testing.T) {
	testutil.CleanEnvForTest(t)
	restore := project.SetClock(testutil.NewClock(time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC), "11111111-1111-1111-1111-111111111111"))
	t.Cleanup(restore)

	store := mustNewStore(t)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	ctx := context.Background()
	user, err := store.CreateUser(ctx, "alice", "raw-token")
	require.NoError(t, err)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", user.UserID)
	require.Equal(t, "alice", user.Username)
	require.Equal(t, time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC), user.CreatedAt)

	var tokenHash string
	var createdAt string
	err = store.db.QueryRowContext(ctx, `SELECT token_hash, created_at FROM users WHERE username = ?`, "alice").Scan(&tokenHash, &createdAt)
	require.NoError(t, err)

	sum := sha256.Sum256([]byte("raw-token"))
	require.Equal(t, hex.EncodeToString(sum[:]), tokenHash)
	require.Equal(t, "2026-06-20T10:11:12Z", createdAt)

	validated, err := store.ValidateToken(ctx, "raw-token")
	require.NoError(t, err)
	require.Equal(t, user.UserID, validated.UserID)
	require.Equal(t, user.Username, validated.Username)

	_, err = store.ValidateToken(ctx, "raw-token-mutated")
	require.Error(t, err)
}

func TestPermissionCascadeOnUserRevoke(t *testing.T) {
	testutil.CleanEnvForTest(t)
	restore := project.SetClock(testutil.NewClock(time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC), "22222222-2222-2222-2222-222222222222"))
	t.Cleanup(restore)

	store := mustNewStore(t)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	ctx := context.Background()
	user, err := store.CreateUser(ctx, "bob", "another-token")
	require.NoError(t, err)

	require.NoError(t, store.GrantPermission(ctx, user.UserID, "project-a", "ro"))
	require.NoError(t, store.GrantPermission(ctx, user.UserID, "project-b", "rw"))

	perms, err := store.GetPermissions(ctx, user.UserID)
	require.NoError(t, err)
	require.Len(t, perms, 2)

	require.NoError(t, store.RevokeUser(ctx, "bob"))

	perms, err = store.GetPermissions(ctx, user.UserID)
	require.NoError(t, err)
	require.Empty(t, perms)
}

func TestGrantPermissionRejectsInvalidAccessLevel(t *testing.T) {
	testutil.CleanEnvForTest(t)

	store := mustNewStore(t)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	err := store.GrantPermission(context.Background(), "user-id", "project-a", "admin")
	require.Error(t, err)
}

func mustNewStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "auth.sqlite")
	store, err := NewStore(path)
	require.NoError(t, err)
	return store
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name)
	return err == nil && name == table
}

func sqliteTableSQL(t *testing.T, db *sql.DB, table string) string {
	t.Helper()

	var sqlText string
	require.NoError(t, db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&sqlText))
	return sqlText
}

func foreignKeyList(t *testing.T, db *sql.DB, table string) string {
	t.Helper()

	rows, err := db.Query(`PRAGMA foreign_key_list(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var (
			id, seq                                              int
			tableName, fromCol, toCol, onUpdate, onDelete, match string
		)
		require.NoError(t, rows.Scan(&id, &seq, &tableName, &fromCol, &toCol, &onUpdate, &onDelete, &match))
		parts = append(parts, tableName+"."+toCol)
	}
	require.NoError(t, rows.Err())
	return strings.Join(parts, ",")
}
