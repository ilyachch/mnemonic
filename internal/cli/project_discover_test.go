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

func TestProjectDiscoverCommandRegistersDirectChildrenOnly(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	err := os.MkdirAll(memoriesHome, 0o755)
	require.NoError(t, err)
	err = writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "backend"), "backend", project.ManifestKindRegular)
	require.NoError(t, err)
	err = writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "personal"), "personal", project.ManifestKindDetached)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(memoriesHome, "ignored", "nested"), 0o755)
	require.NoError(t, err)
	err = writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "ignored", "nested"), "nested", project.ManifestKindRegular)
	require.NoError(t, err)

	result := executeCommand("project", "discover", "--json")
	require.NoError(t, result.Err, "project discover returned error\nstderr: %s", result.Stderr)

	var got projectDiscoverOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, memoriesHome, got.MemoriesHome)
	require.Equal(t, 2, got.Discovered)

	listResult := executeCommand("project", "list", "--json")
	require.NoError(t, listResult.Err, "project list returned error\nstderr: %s", listResult.Stderr)
	var listOutput projectListOutput
	err = json.Unmarshal([]byte(listResult.Stdout), &listOutput)
	require.NoError(t, err, "failed to decode list JSON\nstdout: %s", listResult.Stdout)
	require.Len(t, listOutput.Projects, 2)

	_, err = os.Stat(filepath.Join(memoriesHome, "backend", "index.sqlite"))
	require.True(t, os.IsNotExist(err), "backend index.sqlite exists or stat failed unexpectedly: %v", err)
	_, err = os.Stat(filepath.Join(memoriesHome, "personal", "index.sqlite"))
	require.True(t, os.IsNotExist(err), "personal index.sqlite exists or stat failed unexpectedly: %v", err)
}

func TestProjectDiscoverCommandDryRunReportsInvalidManifests(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	err := os.MkdirAll(memoriesHome, 0o755)
	require.NoError(t, err)
	err = writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "backend"), "backend", project.ManifestKindRegular)
	require.NoError(t, err)
	err = writeInvalidDiscoverCommandManifest(filepath.Join(memoriesHome, "broken"), "broken")
	require.NoError(t, err)

	result := executeCommand("project", "discover", "--dry-run", "--json")
	require.NoError(t, result.Err, "project discover returned error\nstderr: %s", result.Stderr)

	var got projectDiscoverOutput
	err = json.Unmarshal([]byte(result.Stdout), &got)
	require.NoError(t, err, "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, memoriesHome, got.MemoriesHome)
	require.Equal(t, 1, got.Discovered)
	require.Len(t, got.Errors, 1)
	require.Equal(t, filepath.Join(memoriesHome, "broken", "mnemonic.toml"), got.Errors[0].Path)

	listResult := executeCommand("project", "list", "--json")
	require.NoError(t, listResult.Err, "project list returned error\nstderr: %s", listResult.Stderr)
	var listOutput projectListOutput
	err = json.Unmarshal([]byte(listResult.Stdout), &listOutput)
	require.NoError(t, err, "failed to decode list JSON\nstdout: %s", listResult.Stdout)
	require.Len(t, listOutput.Projects, 0)
}

func writeDiscoverCommandManifest(t *testing.T, projectRoot, name string, kind project.ManifestKind) error {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = projectIDForCommandSlug(name)
	manifest.Name = name
	manifest.Slug = name
	manifest.Kind = kind
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now
	manifest.UpdatedAt = now
	manifest.Generator.App = "mnemonic"

	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return err
	}

	return project.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest)
}

func writeInvalidDiscoverCommandManifest(projectRoot, name string) error {
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return err
	}

	manifest := []byte(`version = 1
project_id = "550e8400-e29b-41d4-a716-446655440010"
name = "` + name + `"
slug = "` + name + `"
kind = "local"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[layout]
notes_glob = ["**/*.md"]
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
`)

	return os.WriteFile(filepath.Join(projectRoot, "mnemonic.toml"), manifest, 0o644)
}

func projectIDForCommandSlug(slug string) string {
	switch slug {
	case "backend":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "personal":
		return "550e8400-e29b-41d4-a716-446655440001"
	case "nested":
		return "550e8400-e29b-41d4-a716-446655440002"
	default:
		return "550e8400-e29b-41d4-a716-446655440099"
	}
}