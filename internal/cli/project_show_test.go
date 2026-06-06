package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

type projectShowJSON struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Kind      string `json:"kind"`
	StatePath string `json:"state_path"`
	Location  struct {
		MnemonicFileAbs string `json:"mnemonic_file_abs"`
		RepoRootAbs     string `json:"repo_root_abs"`
		MemoriesAbs     string `json:"memories_abs"`
		ManifestAbs     string `json:"manifest_abs"`
		SourceKind      string `json:"source_kind"`
		LastSeenAt      string `json:"last_seen_at"`
	} `json:"location"`
	Status struct {
		IndexSchemaVersion int    `json:"index_schema_version"`
		LastSeenAt         string `json:"last_seen_at"`
		IndexPresent       bool   `json:"index_present"`
		NeedsReindex       bool   `json:"needs_reindex"`
	} `json:"status"`
}

func TestProjectShowCommandFindsProjectBySlugAndUUID(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	stateHome := filepath.Join(root, "state")
	cwd := t.TempDir()

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	if err := registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend",
		Slug:      "backend",
		Kind:      registry.ProjectKindLocal,
		CreatedAt: now,
		UpdatedAt: now,
		SeenAt:    now,
		Location: registry.ProjectLocationInput{
			MnemonicFileAbs: filepath.Join(cwd, ".mnemonic"),
			RepoRootAbs:     cwd,
			MemoriesAbs:     filepath.Join(cwd, ".mnemonic-memories", "backend"),
			SourceKind:      registry.ProjectSourceKindInit,
		},
	}); err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}

	bySlug := executeCommand("project", "show", "backend", "--json")
	if bySlug.Err != nil {
		t.Fatalf("project show by slug returned error: %v\nstderr: %s", bySlug.Err, bySlug.Stderr)
	}
	var got projectShowJSON
	if err := json.Unmarshal([]byte(bySlug.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON by slug: %v\nstdout: %s", err, bySlug.Stdout)
	}
	assertProjectShowJSON(t, got, projectShowJSON{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "backend",
		Slug:      "backend",
		Kind:      string(registry.ProjectKindLocal),
		StatePath: filepath.Join(stateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "state.toml"),
		Location: struct {
			MnemonicFileAbs string `json:"mnemonic_file_abs"`
			RepoRootAbs     string `json:"repo_root_abs"`
			MemoriesAbs     string `json:"memories_abs"`
			ManifestAbs     string `json:"manifest_abs"`
			SourceKind      string `json:"source_kind"`
			LastSeenAt      string `json:"last_seen_at"`
		}{
			MnemonicFileAbs: filepath.Join(cwd, ".mnemonic"),
			RepoRootAbs:     cwd,
			MemoriesAbs:     filepath.Join(cwd, ".mnemonic-memories", "backend"),
			ManifestAbs:     "",
			SourceKind:      string(registry.ProjectSourceKindInit),
			LastSeenAt:      now.UTC().Format(time.RFC3339),
		},
		Status: struct {
			IndexSchemaVersion int    `json:"index_schema_version"`
			LastSeenAt         string `json:"last_seen_at"`
			IndexPresent       bool   `json:"index_present"`
			NeedsReindex       bool   `json:"needs_reindex"`
		}{
			IndexSchemaVersion: 0,
			LastSeenAt:         now.UTC().Format(time.RFC3339),
			IndexPresent:       false,
			NeedsReindex:       true,
		},
	})

	byUUID := executeCommand("project", "show", "550e8400-e29b-41d4-a716-446655440000", "--json")
	if byUUID.Err != nil {
		t.Fatalf("project show by UUID returned error: %v\nstderr: %s", byUUID.Err, byUUID.Stderr)
	}
	if err := json.Unmarshal([]byte(byUUID.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON by UUID: %v\nstdout: %s", err, byUUID.Stdout)
	}
	if got.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("project_id = %q, want %q", got.ProjectID, "550e8400-e29b-41d4-a716-446655440000")
	}
}

func TestProjectShowCommandMissingProject(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("project", "show", "missing", "--json")
	if result.Err == nil {
		t.Fatal("project show error = nil, want not-found")
	}
	if got := ExitCodeForError(result.Err); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func assertProjectShowJSON(t *testing.T, got, want projectShowJSON) {
	t.Helper()

	if got.ProjectID != want.ProjectID {
		t.Fatalf("project_id = %q, want %q", got.ProjectID, want.ProjectID)
	}
	if got.Name != want.Name {
		t.Fatalf("name = %q, want %q", got.Name, want.Name)
	}
	if got.Slug != want.Slug {
		t.Fatalf("slug = %q, want %q", got.Slug, want.Slug)
	}
	if got.Kind != want.Kind {
		t.Fatalf("kind = %q, want %q", got.Kind, want.Kind)
	}
	if got.StatePath != want.StatePath {
		t.Fatalf("state_path = %q, want %q", got.StatePath, want.StatePath)
	}
	if got.Location != want.Location {
		t.Fatalf("location = %#v, want %#v", got.Location, want.Location)
	}
	if got.Status != want.Status {
		t.Fatalf("status = %#v, want %#v", got.Status, want.Status)
	}
}
