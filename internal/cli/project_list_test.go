package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, result.Err, "project list returned error\nstderr: %s", result.Stderr)

	var got projectListJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.NotNil(t, got.Projects, "projects is nil, want empty array")
	require.Len(t, got.Projects, 0)
}

func TestProjectListCommandReturnsSeededProject(t *testing.T) {
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

	result := executeCommand("project", "list", "--json")
	require.NoError(t, result.Err, "project list returned error\nstderr: %s", result.Stderr)

	var got projectListJSON
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.NotNil(t, got.Projects, "projects is nil, want populated array")
	require.Len(t, got.Projects, 1)

	project := got.Projects[0]
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", project.ProjectID)
	require.Equal(t, "backend", project.Name)
	require.Equal(t, "backend", project.Slug)
	require.Equal(t, string(registry.ProjectKindLocal), project.Kind)
	require.Equal(t, filepath.Join(cwd, ".mnemonic-memories", "backend"), project.MemoriesPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", project.ProjectID, "state.toml"), project.StatePath)
	require.True(t, project.NeedsReindex, "needs_reindex = false, want true")
	require.False(t, project.IndexPresent, "index_present = true, want false")
}

func TestProjectListCommandHumanOutputIncludesProjectDetails(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

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

	result := executeCommand("project", "list")
	require.NoError(t, result.Err, "project list returned error\nstderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, "1 projects")
	require.Contains(t, result.Stdout, "NAME")
	require.Contains(t, result.Stdout, "SLUG")
	require.Contains(t, result.Stdout, "TYPE")
	require.Contains(t, result.Stdout, "backend")
	require.Contains(t, result.Stdout, "present=false needs_reindex=true")
}
