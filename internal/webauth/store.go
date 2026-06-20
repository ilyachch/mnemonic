package webauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// Store owns the web auth SQLite database.
type Store struct {
	db *sql.DB
}

// User is an auth database user record.
type User struct {
	UserID    string
	Username  string
	CreatedAt time.Time
}

// Permission is an access grant for a user to a project selector.
type Permission struct {
	UserID          string
	ProjectSelector string
	AccessLevel     string
}

// ResolveDBPath resolves the auth database path using explicit > env > default.
func ResolveDBPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if envPath := strings.TrimSpace(os.Getenv("MNEMONIC_WEB_AUTH_DB")); envPath != "" {
		return envPath, nil
	}

	effective, err := paths.GetMnemonicPaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(effective.StateHome, "mnemonic", "web_auth.sqlite"), nil
}

// NewStore initializes the DB file if it doesn't exist and applies the schema.
func NewStore(dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, app.NewCLIUsageError("auth database path is required", nil)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create auth database directory: %w", err)
	}

	db, err := sql.Open(sqliteDriverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open auth database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping auth database: %w", err)
	}

	db.SetMaxOpenConns(1)

	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}

	if err := applySchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply auth schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// CreateUser inserts a new user and stores only the SHA-256 digest of the token.
func (s *Store) CreateUser(ctx context.Context, username string, rawToken string) (*User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if strings.TrimSpace(rawToken) == "" {
		return nil, app.NewCLIUsageError("raw token is required", nil)
	}

	user := User{
		UserID:    project.NewUUID(),
		Username:  username,
		CreatedAt: project.NowUTC().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (user_id, username, token_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, user.UserID, user.Username, hashToken(rawToken), user.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("create user %q: %w", username, err)
	}

	return &user, nil
}

// ListUsers returns all users ordered by username.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, username, created_at
		FROM users
		ORDER BY username ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var (
			user      User
			createdAt string
		)
		if err := rows.Scan(&user.UserID, &user.Username, &createdAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at for %q: %w", user.Username, err)
		}
		user.CreatedAt = parsed.UTC()
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

// RevokeUser deletes a user by username and cascades permission cleanup.
func (s *Store) RevokeUser(ctx context.Context, username string) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return fmt.Errorf("revoke user %q: %w", username, err)
	}
	return nil
}

// ValidateToken returns the user that owns the provided raw token.
func (s *Store) ValidateToken(ctx context.Context, rawToken string) (*User, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, app.NewCLIUsageError("raw token is required", nil)
	}

	user, err := s.lookupUser(ctx, `
		SELECT user_id, username, created_at
		FROM users
		WHERE token_hash = ?
	`, hashToken(rawToken))
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}
	return user, nil
}

// GetUserByUsername looks up a user by exact username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}

	user, err := s.lookupUser(ctx, `
		SELECT user_id, username, created_at
		FROM users
		WHERE username = ?
	`, username)
	if err != nil {
		return nil, fmt.Errorf("get user %q: %w", username, err)
	}
	return user, nil
}

// GrantPermission creates or updates a permission grant.
func (s *Store) GrantPermission(ctx context.Context, userID string, projectSelector string, level string) error {
	if err := validatePermissionInputs(userID, projectSelector, level); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO permissions (user_id, project_selector, access_level)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, project_selector) DO UPDATE SET access_level = excluded.access_level
	`, userID, projectSelector, level)
	if err != nil {
		return fmt.Errorf("grant permission for %q/%q: %w", userID, projectSelector, err)
	}
	return nil
}

// RevokePermission removes a project grant for a user.
func (s *Store) RevokePermission(ctx context.Context, userID string, projectSelector string) error {
	if strings.TrimSpace(userID) == "" {
		return app.NewCLIUsageError("user ID is required", nil)
	}
	if strings.TrimSpace(projectSelector) == "" {
		return app.NewCLIUsageError("project selector is required", nil)
	}

	_, err := s.db.ExecContext(ctx, `
		DELETE FROM permissions
		WHERE user_id = ? AND project_selector = ?
	`, userID, projectSelector)
	if err != nil {
		return fmt.Errorf("revoke permission for %q/%q: %w", userID, projectSelector, err)
	}
	return nil
}

// GetPermissions returns the permissions for a user.
func (s *Store) GetPermissions(ctx context.Context, userID string) ([]Permission, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, app.NewCLIUsageError("user ID is required", nil)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, project_selector, access_level
		FROM permissions
		WHERE user_id = ?
		ORDER BY project_selector ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get permissions for %q: %w", userID, err)
	}
	defer rows.Close()

	perms := make([]Permission, 0)
	for rows.Next() {
		var perm Permission
		if err := rows.Scan(&perm.UserID, &perm.ProjectSelector, &perm.AccessLevel); err != nil {
			return nil, fmt.Errorf("scan permission for %q: %w", userID, err)
		}
		perms = append(perms, perm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions for %q: %w", userID, err)
	}
	return perms, nil
}

func applySchema(db *sql.DB) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			token_hash TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS permissions (
			user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			project_selector TEXT NOT NULL,
			access_level TEXT NOT NULL,
			PRIMARY KEY (user_id, project_selector)
		)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func validateUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return app.NewCLIUsageError("username is required", nil)
	}
	return nil
}

func validatePermissionInputs(userID string, projectSelector string, level string) error {
	if strings.TrimSpace(userID) == "" {
		return app.NewCLIUsageError("user ID is required", nil)
	}
	if strings.TrimSpace(projectSelector) == "" {
		return app.NewCLIUsageError("project selector is required", nil)
	}
	if level != "ro" && level != "rw" {
		return app.NewCLIUsageError("access level must be ro or rw", nil)
	}
	return nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func (s *Store) lookupUser(ctx context.Context, query string, arg any) (*User, error) {
	row := s.db.QueryRowContext(ctx, query, arg)

	var user User
	var createdAt string
	if err := row.Scan(&user.UserID, &user.Username, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.NewNotFoundError("user not found", nil)
		}
		return nil, err
	}

	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at for %q: %w", user.Username, err)
	}
	user.CreatedAt = parsed.UTC()
	return &user, nil
}
