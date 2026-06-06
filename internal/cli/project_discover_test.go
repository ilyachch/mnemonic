package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestProjectDiscoverCommandRegistersDirectChildrenOnly(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	if err := os.MkdirAll(memoriesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "backend"), "backend", project.ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverCommandManifest() error = %v", err)
	}
	if err := writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "personal"), "personal", project.ManifestKindDetached); err != nil {
		t.Fatalf("writeDiscoverCommandManifest() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(memoriesHome, "ignored", "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "ignored", "nested"), "nested", project.ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverCommandManifest() error = %v", err)
	}

	result := executeCommand("project", "discover", "--json")
	if result.Err != nil {
		t.Fatalf("project discover returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectDiscoverOutput
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.MemoriesHome != memoriesHome {
		t.Fatalf("memories_home = %q, want %q", got.MemoriesHome, memoriesHome)
	}
	if got.Discovered != 2 {
		t.Fatalf("discovered = %d, want 2", got.Discovered)
	}

	listResult := executeCommand("project", "list", "--json")
	if listResult.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", listResult.Err, listResult.Stderr)
	}
	var listOutput projectListOutput
	if err := json.Unmarshal([]byte(listResult.Stdout), &listOutput); err != nil {
		t.Fatalf("failed to decode list JSON: %v\nstdout: %s", err, listResult.Stdout)
	}
	if len(listOutput.Projects) != 2 {
		t.Fatalf("project list length = %d, want 2", len(listOutput.Projects))
	}

	if _, err := os.Stat(filepath.Join(memoriesHome, "backend", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("backend index.sqlite exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(memoriesHome, "personal", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("personal index.sqlite exists or stat failed unexpectedly: %v", err)
	}
}

func TestProjectDiscoverCommandDryRunReportsInvalidManifests(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	if err := os.MkdirAll(memoriesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := writeDiscoverCommandManifest(t, filepath.Join(memoriesHome, "backend"), "backend", project.ManifestKindRegular); err != nil {
		t.Fatalf("writeDiscoverCommandManifest() error = %v", err)
	}
	if err := writeInvalidDiscoverCommandManifest(filepath.Join(memoriesHome, "broken"), "broken"); err != nil {
		t.Fatalf("writeInvalidDiscoverCommandManifest() error = %v", err)
	}

	result := executeCommand("project", "discover", "--dry-run", "--json")
	if result.Err != nil {
		t.Fatalf("project discover returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got projectDiscoverOutput
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.MemoriesHome != memoriesHome {
		t.Fatalf("memories_home = %q, want %q", got.MemoriesHome, memoriesHome)
	}
	if got.Discovered != 1 {
		t.Fatalf("discovered = %d, want 1", got.Discovered)
	}
	if len(got.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(got.Errors))
	}
	if got.Errors[0].Path != filepath.Join(memoriesHome, "broken", "mnemonic.toml") {
		t.Fatalf("error path = %q, want %q", got.Errors[0].Path, filepath.Join(memoriesHome, "broken", "mnemonic.toml"))
	}

	listResult := executeCommand("project", "list", "--json")
	if listResult.Err != nil {
		t.Fatalf("project list returned error: %v\nstderr: %s", listResult.Err, listResult.Stderr)
	}
	var listOutput projectListOutput
	if err := json.Unmarshal([]byte(listResult.Stdout), &listOutput); err != nil {
		t.Fatalf("failed to decode list JSON: %v\nstdout: %s", err, listResult.Stdout)
	}
	if len(listOutput.Projects) != 0 {
		t.Fatalf("project list length = %d, want 0", len(listOutput.Projects))
	}
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
