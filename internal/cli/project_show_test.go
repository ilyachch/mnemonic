package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	err = registry.RegisterProject(db, registry.RegisterProjectInput{
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
	})
	require.NoError(t, err)

	bySlug := executeCommand("project", "show", "backend", "--json")
	require.NoError(t, bySlug.Err, "project show by slug returned error\nstderr: %s", bySlug.Stderr)
	var got projectShowJSON
	err = json.Unmarshal([]byte(bySlug.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON by slug\nstdout: %s", bySlug.Stdout)

	want := projectShowJSON{
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
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("project show by slug mismatch (-want +got):\n%s", diff)
	}

	byUUID := executeCommand("project", "show", "550e8400-e29b-41d4-a716-446655440000", "--json")
	require.NoError(t, byUUID.Err, "project show by UUID returned error\nstderr: %s", byUUID.Stderr)
	err = json.Unmarshal([]byte(byUUID.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON by UUID\nstdout: %s", byUUID.Stdout)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.ProjectID)
}

func TestProjectShowCommandMissingProject(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("project", "show", "missing", "--json")
	require.Error(t, result.Err, "project show error = nil, want not-found")
	require.Equal(t, 3, ExitCodeForError(result.Err))
}