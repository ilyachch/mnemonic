package registry

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestRegisterProjectCreatesProjectLocationAndStatus(t *testing.T) {
	db := openTestRegistry(t)

	seenAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	err := RegisterProject(db, RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend",
		Slug:      "backend",
		Kind:      ProjectKindLocal,
		CreatedAt: seenAt,
		UpdatedAt: seenAt,
		SeenAt:    seenAt,
		Location: ProjectLocationInput{
			MnemonicFileAbs: "/tmp/repo/.mnemonic",
			RepoRootAbs:     "/tmp/repo",
			MemoriesAbs:     "/tmp/repo/.mnemonic-memories/backend",
			ManifestAbs:     "",
			SourceKind:      ProjectSourceKindInit,
		},
	})
	require.NoError(t, err)

	require.True(t, rowExists(t, db, `SELECT 1 FROM projects WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000"), "projects row missing")
	require.True(t, rowExists(t, db, `SELECT 1 FROM project_locations WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000"), "project_locations row missing")
	require.True(t, rowExists(t, db, `SELECT 1 FROM project_status WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000"), "project_status row missing")

	var needsReindex int
	err = db.QueryRow(`SELECT needs_reindex FROM project_status WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000").Scan(&needsReindex)
	require.NoError(t, err)
	require.Equal(t, 1, needsReindex)
}

func TestRegisterProjectRejectsDuplicateActiveSlug(t *testing.T) {
	db := openTestRegistry(t)

	seenAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	err := RegisterProject(db, RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend",
		Slug:      "backend",
		Kind:      ProjectKindLocal,
		CreatedAt: seenAt,
		UpdatedAt: seenAt,
		SeenAt:    seenAt,
		Location: ProjectLocationInput{
			MemoriesAbs: "/tmp/repo/.mnemonic-memories/backend",
			SourceKind:  ProjectSourceKindInit,
		},
	})
	require.NoError(t, err)

	err = RegisterProject(db, RegisterProjectInput{
		ProjectID: "7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
		Name:      "frontend",
		Slug:      "backend",
		Kind:      ProjectKindLocal,
		CreatedAt: seenAt.Add(time.Hour),
		UpdatedAt: seenAt.Add(time.Hour),
		SeenAt:    seenAt.Add(time.Hour),
		Location: ProjectLocationInput{
			MemoriesAbs: "/tmp/repo/.mnemonic-memories/frontend",
			SourceKind:  ProjectSourceKindInit,
		},
	})
	require.Error(t, err)
	require.EqualError(t, err, `project slug "backend" already exists`)
}

func TestRegisterProjectNilDB(t *testing.T) {
	err := RegisterProject(nil, RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Slug:      "test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "registry database is nil")
}

func TestRegisterProjectReRegistersSameProjectID(t *testing.T) {
	db := openTestRegistry(t)

	firstSeenAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	err := RegisterProject(db, RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend",
		Slug:      "backend",
		Kind:      ProjectKindLocal,
		CreatedAt: firstSeenAt,
		UpdatedAt: firstSeenAt,
		SeenAt:    firstSeenAt,
		Location: ProjectLocationInput{
			MnemonicFileAbs: "/tmp/repo/.mnemonic",
			MemoriesAbs:     "/tmp/repo/.mnemonic-memories/backend",
			SourceKind:      ProjectSourceKindInit,
		},
	})
	require.NoError(t, err)

	secondSeenAt := firstSeenAt.Add(time.Hour)
	err = RegisterProject(db, RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend renamed",
		Slug:      "backend",
		Kind:      ProjectKindLocal,
		CreatedAt: firstSeenAt,
		UpdatedAt: secondSeenAt,
		SeenAt:    secondSeenAt,
		Location: ProjectLocationInput{
			MnemonicFileAbs: "/tmp/other/.mnemonic",
			MemoriesAbs:     "/tmp/other/.mnemonic-memories/backend",
			SourceKind:      ProjectSourceKindDiscover,
		},
	})
	require.NoError(t, err)

	var mnemonicFileAbs, memoriesAbs, sourceKind, lastSeenAt string
	err = db.QueryRow(
		`SELECT mnemonic_file_abs, memories_abs, source_kind, last_seen_at FROM project_locations WHERE project_id = ?`,
		"550e8400-e29b-41d4-a716-446655440000",
	).Scan(&mnemonicFileAbs, &memoriesAbs, &sourceKind, &lastSeenAt)
	require.NoError(t, err)
	require.Equal(t, "/tmp/other/.mnemonic", mnemonicFileAbs)
	require.Equal(t, "/tmp/other/.mnemonic-memories/backend", memoriesAbs)
	require.Equal(t, string(ProjectSourceKindDiscover), sourceKind)
	require.Equal(t, secondSeenAt.UTC().Format(time.RFC3339), lastSeenAt)
}

func openTestRegistry(t *testing.T) *sql.DB {
	t.Helper()

	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	require.NoError(t, err)
	err = ApplySchema(db)
	if err != nil {
		_ = db.Close()
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func rowExists(t *testing.T, db *sql.DB, query string, args ...any) bool {
	t.Helper()

	var one int
	err := db.QueryRow(query, args...).Scan(&one)
	return err == nil
}
