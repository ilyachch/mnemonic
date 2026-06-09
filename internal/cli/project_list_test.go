package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

type projectListJSON struct {
	Projects []struct {
		ProjectID    string `json:"project_id"`
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Kind         string `json:"kind"`
		MemoriesPath string `json:"memories_path"`
		StatePath    string `json:"state_path"`
		NeedsReindex bool   `json:"needs_reindex"`
		IndexPresent bool   `json:"index_present"`
	} `json:"projects"`
}

func TestProjectListCommandEmptyRegistry(t *testing.T) {
	testutil.CleanEnvForTest(t)

	result := executeCommand("project", "list", "--json")
	if result.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectListJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Projects == nil {
		t.Fatal("projects is nil, want empty array")
	}
	if len(got.Projects) != 0 {
		t.Fatalf("len(projects) = %d, want 0", len(got.Projects))
	}
}

func TestProjectListCommandReturnsSeededProject(t *testing.T) {
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

	result := executeCommand("project", "list", "--json")
	if result.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectListJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Projects == nil {
		t.Fatal("projects is nil, want populated array")
	}
	if len(got.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(got.Projects))
	}

	project := got.Projects[0]
	if project.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("project_id = %q, want %q", project.ProjectID, "550e8400-e29b-41d4-a716-446655440000")
	}
	if project.Name != "backend" {
		t.Fatalf("name = %q, want %q", project.Name, "backend")
	}
	if project.Slug != "backend" {
		t.Fatalf("slug = %q, want %q", project.Slug, "backend")
	}
	if project.Kind != string(registry.ProjectKindLocal) {
		t.Fatalf("kind = %q, want %q", project.Kind, registry.ProjectKindLocal)
	}
	if project.MemoriesPath != filepath.Join(cwd, ".mnemonic-memories", "backend") {
		t.Fatalf("memories_path = %q, want %q", project.MemoriesPath, filepath.Join(cwd, ".mnemonic-memories", "backend"))
	}
	if project.StatePath != filepath.Join(stateHome, "mnemonic", "projects", project.ProjectID, "state.toml") {
		t.Fatalf("state_path = %q, want %q", project.StatePath, filepath.Join(stateHome, "mnemonic", "projects", project.ProjectID, "state.toml"))
	}
	if !project.NeedsReindex {
		t.Fatal("needs_reindex = false, want true")
	}
	if project.IndexPresent {
		t.Fatal("index_present = true, want false")
	}
}

func TestProjectListCommandHumanOutputIncludesProjectDetails(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

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

	result := executeCommand("project", "list")
	if result.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "1 projects") {
		t.Fatalf("stdout missing project count: %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "NAME") || !strings.Contains(result.Stdout, "SLUG") || !strings.Contains(result.Stdout, "TYPE") {
		t.Fatalf("stdout missing table headers: %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "backend") {
		t.Fatalf("stdout missing project row: %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "present=false needs_reindex=true") {
		t.Fatalf("stdout missing index status: %q", result.Stdout)
	}
}
