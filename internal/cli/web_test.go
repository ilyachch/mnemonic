package cli

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/ilyachch/mnemonic/internal/webauth"
	"github.com/stretchr/testify/require"
)

func TestWebHelpShowsSubcommands(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("web", "--help")
	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, "serve")
	require.Contains(t, result.Stdout, "users")
	require.Contains(t, result.Stdout, "perms")
}

func TestWebServeUsesPortProjectsAndAuthDBFlags(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	require.NoError(t, os.MkdirAll(filepath.Join(memoriesHome, "demo"), 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(memoriesHome, "demo", "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(memoriesHome, "demo", "demo.md"), []byte("# Demo\n"), 0o644))
	_, err := index.RebuildProjectIndex(manifest.ProjectID, filepath.Join(memoriesHome, "demo"))
	require.NoError(t, err)

	explicitAuthDB := filepath.Join(root, "state", "explicit", "web_auth.sqlite")
	envAuthDB := filepath.Join(root, "state", "env", "web_auth.sqlite")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_SERVE_PROJECTS", "demo")
	t.Setenv("MNEMONIC_WEB_AUTH_DB", envAuthDB)

	result := executeCommand(
		"web",
		"--auth-db", explicitAuthDB,
		"serve",
		"--projects", "demo",
		"--port", "not-a-port",
	)
	require.Error(t, result.Err)
	require.Equal(t, int(app.CodeInternal), ExitCodeForError(result.Err))
	require.Contains(t, result.Err.Error(), "serve web MCP")

	_, err = os.Stat(explicitAuthDB)
	require.NoError(t, err)
	_, err = os.Stat(envAuthDB)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestWebUsersAndPermsMutateAuthDB(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	require.NoError(t, os.MkdirAll(filepath.Join(memoriesHome, "demo"), 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	manifest.Name = "Demo"
	manifest.Slug = "demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, 6, 20, 10, 11, 12, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(memoriesHome, "demo", "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(memoriesHome, "demo", "demo.md"), []byte("# Demo\n"), 0o644))
	_, err := index.RebuildProjectIndex(manifest.ProjectID, filepath.Join(memoriesHome, "demo"))
	require.NoError(t, err)

	authDB := filepath.Join(root, "state", "web", "web_auth.sqlite")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_WEB_AUTH_DB", authDB)

	addResult := executeCommand("web", "--auth-db", authDB, "users", "add", "alice", "--json")
	require.NoError(t, addResult.Err)

	var addOutput struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(addResult.Stdout), &addOutput))
	require.Equal(t, "alice", addOutput.Username)
	require.NotEmpty(t, addOutput.Token)

	store, err := webauth.NewStore(authDB)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	db, err := sql.Open("sqlite", authDB)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	user, err := store.GetUserByUsername(context.Background(), "alice")
	require.NoError(t, err)

	var tokenHash string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT token_hash FROM users WHERE username = ?`, "alice").Scan(&tokenHash))
	sum := sha256.Sum256([]byte(addOutput.Token))
	require.Equal(t, hex.EncodeToString(sum[:]), tokenHash)

	listResult := executeCommand("web", "--auth-db", authDB, "users", "list", "--json")
	require.NoError(t, listResult.Err)

	var listOutput struct {
		Users []string `json:"users"`
	}
	require.NoError(t, json.Unmarshal([]byte(listResult.Stdout), &listOutput))
	require.Equal(t, []string{"alice"}, listOutput.Users)

	require.NoError(t, store.GrantPermission(context.Background(), user.UserID, "demo", "rw"))

	var level string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT access_level FROM permissions WHERE user_id = ? AND project_selector = ?`, user.UserID, "demo").Scan(&level))
	require.Equal(t, "rw", level)

	revokePermResult := executeCommand("web", "--auth-db", authDB, "perms", "revoke", "alice", "demo")
	require.NoError(t, revokePermResult.Err)

	err = db.QueryRowContext(context.Background(), `SELECT access_level FROM permissions WHERE user_id = ? AND project_selector = ?`, user.UserID, "demo").Scan(&level)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	require.NoError(t, store.GrantPermission(context.Background(), user.UserID, "demo", "ro"))
	revokeUserResult := executeCommand("web", "--auth-db", authDB, "users", "revoke", "alice")
	require.NoError(t, revokeUserResult.Err)

	var username string
	err = db.QueryRowContext(context.Background(), `SELECT username FROM users WHERE username = ?`, "alice").Scan(&username)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = db.QueryRowContext(context.Background(), `SELECT access_level FROM permissions WHERE user_id = ?`, user.UserID).Scan(&level)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	permsGrantResult := executeCommand("web", "--auth-db", authDB, "perms", "grant", "alice", "demo", "--level", "ro")
	require.Error(t, permsGrantResult.Err)
	require.Equal(t, int(app.CodeNotFound), ExitCodeForError(permsGrantResult.Err))
}

func TestWebUsersListHumanOutput(t *testing.T) {
	testutil.CleanEnvForTest(t)

	authDB := filepath.Join(t.TempDir(), "web_auth.sqlite")
	t.Setenv("MNEMONIC_WEB_AUTH_DB", authDB)

	store, err := webauth.NewStore(authDB)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	_, err = store.CreateUser(context.Background(), "alice", "token-a")
	require.NoError(t, err)
	_, err = store.CreateUser(context.Background(), "bob", "token-b")
	require.NoError(t, err)

	result := executeCommand("web", "--auth-db", authDB, "users", "list")
	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, "USERNAME")
	require.Contains(t, result.Stdout, "alice")
	require.Contains(t, result.Stdout, "bob")
}
