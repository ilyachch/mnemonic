package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

type projectListJSON struct {
	Projects []struct {
		ProjectID    string `json:"project_id"`
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Type         string `json:"type"`
		MemoriesPath string `json:"memories_path"`
		StatePath    string `json:"state_path"`
		Status       string `json:"status"`
		Issue        string `json:"issue,omitempty"`
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
	memoriesHome := t.TempDir()
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "backend", "550e8400-e29b-41d4-a716-446655440000", now)

	result := executeCommand("project", "list", "--json")
	require.NoError(t, result.Err, "project list returned error\nstderr: %s", result.Stderr)

	var got projectListJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.NotNil(t, got.Projects, "projects is nil, want populated array")
	require.Len(t, got.Projects, 1)

	project := got.Projects[0]
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", project.ProjectID)
	require.Equal(t, "backend", project.Name)
	require.Equal(t, "backend", project.Slug)
	require.Equal(t, "central", project.Type)
	require.Equal(t, filepath.Join(memoriesHome, "backend"), project.MemoriesPath)
	require.NotEmpty(t, project.StatePath)
}

func TestProjectListCommandHumanOutputIncludesProjectDetails(t *testing.T) {
	memoriesHome := t.TempDir()
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "backend", "550e8400-e29b-41d4-a716-446655440000", now)

	result := executeCommand("project", "list")
	require.NoError(t, result.Err, "project list returned error\nstderr: %s", result.Stderr)
	require.Contains(t, result.Stdout, "1 projects")
	require.Contains(t, result.Stdout, "NAME")
	require.Contains(t, result.Stdout, "SLUG")
	require.Contains(t, result.Stdout, "TYPE")
	require.Contains(t, result.Stdout, "backend")
	require.Contains(t, result.Stdout, "STATUS")
}
