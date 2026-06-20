package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

type projectShowJSON struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type"`
	StateHome string `json:"state_home"`
	Location  struct {
		MemoriesAbs string `json:"memories_abs"`
		ManifestAbs string `json:"manifest_abs"`
		RepoRootAbs string `json:"repo_root_abs"`
	} `json:"location"`
}

func TestProjectShowCommandFindsProject(t *testing.T) {
	testutil.CleanEnvForTest(t)
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	slug := "backend"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now
	manifest.UpdatedAt = now
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	result := executeCommand("project", "show", "backend", "--json")
	require.NoError(t, result.Err, "project show returned error\nstderr: %s", result.Stderr)

	var got projectShowJSON
	err := json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)

	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.ProjectID)
	require.Equal(t, "backend", got.Name)
	require.Equal(t, "backend", got.Slug)
	require.Equal(t, "central", got.Type)
}

func TestProjectShowCommandMissingProject(t *testing.T) {
	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", t.TempDir())

}
