package registry

import (
	"database/sql"
	"testing"
	"time"

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
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}

	if !rowExists(t, db, `SELECT 1 FROM projects WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("projects row missing")
	}
	if !rowExists(t, db, `SELECT 1 FROM project_locations WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("project_locations row missing")
	}
	if !rowExists(t, db, `SELECT 1 FROM project_status WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("project_status row missing")
	}

	var needsReindex int
	if err := db.QueryRow(`SELECT needs_reindex FROM project_status WHERE project_id = ?`, "550e8400-e29b-41d4-a716-446655440000").Scan(&needsReindex); err != nil {
		t.Fatalf("query needs_reindex failed: %v", err)
	}
	if needsReindex != 1 {
		t.Fatalf("needs_reindex = %d, want 1", needsReindex)
	}
}

func TestRegisterProjectRejectsDuplicateActiveSlug(t *testing.T) {
	db := openTestRegistry(t)

	seenAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	if err := RegisterProject(db, RegisterProjectInput{
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
	}); err != nil {
		t.Fatalf("first RegisterProject() error = %v", err)
	}

	err := RegisterProject(db, RegisterProjectInput{
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
	if err == nil {
		t.Fatal("RegisterProject() error = nil, want duplicate slug rejection")
	}
	if err.Error() != `project slug "backend" already exists` {
		t.Fatalf("RegisterProject() error = %q, want duplicate slug rejection", err)
	}
}

func TestRegisterProjectReRegistersSameProjectID(t *testing.T) {
	db := openTestRegistry(t)

	firstSeenAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	if err := RegisterProject(db, RegisterProjectInput{
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
	}); err != nil {
		t.Fatalf("first RegisterProject() error = %v", err)
	}

	secondSeenAt := firstSeenAt.Add(time.Hour)
	if err := RegisterProject(db, RegisterProjectInput{
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
	}); err != nil {
		t.Fatalf("second RegisterProject() error = %v", err)
	}

	var mnemonicFileAbs, memoriesAbs, sourceKind, lastSeenAt string
	if err := db.QueryRow(
		`SELECT mnemonic_file_abs, memories_abs, source_kind, last_seen_at FROM project_locations WHERE project_id = ?`,
		"550e8400-e29b-41d4-a716-446655440000",
	).Scan(&mnemonicFileAbs, &memoriesAbs, &sourceKind, &lastSeenAt); err != nil {
		t.Fatalf("query project_locations failed: %v", err)
	}
	if mnemonicFileAbs != "/tmp/other/.mnemonic" {
		t.Fatalf("mnemonic_file_abs = %q, want %q", mnemonicFileAbs, "/tmp/other/.mnemonic")
	}
	if memoriesAbs != "/tmp/other/.mnemonic-memories/backend" {
		t.Fatalf("memories_abs = %q, want %q", memoriesAbs, "/tmp/other/.mnemonic-memories/backend")
	}
	if sourceKind != string(ProjectSourceKindDiscover) {
		t.Fatalf("source_kind = %q, want %q", sourceKind, ProjectSourceKindDiscover)
	}
	if lastSeenAt != secondSeenAt.UTC().Format(time.RFC3339) {
		t.Fatalf("last_seen_at = %q, want %q", lastSeenAt, secondSeenAt.UTC().Format(time.RFC3339))
	}
}

func openTestRegistry(t *testing.T) *sql.DB {
	t.Helper()

	testutil.CleanEnvForTest(t)

	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	if err := ApplySchema(db); err != nil {
		_ = db.Close()
		t.Fatalf("ApplySchema() error = %v", err)
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
