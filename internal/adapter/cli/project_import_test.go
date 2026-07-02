package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectImportCommandOnboardsRawDirectory(t *testing.T) {
	cwd, importRoot := seedRawImportDir(t)
	seedRawNote(t, importRoot, "raw-note.md", "# Raw Note\n\nBody.\n")

	restore := clock.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		"manifest-uuid-0000-0000-000000000001",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	))
	t.Cleanup(restore)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(importRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "import", ".", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, 1, got.Imported)
	require.Equal(t, 1, got.Indexed)
	require.Equal(t, "ok", got.IndexStatus)
	require.True(t, got.ManifestCreated)
	require.Len(t, got.Hydrated, 1)
	require.Equal(t, "raw-note.md", got.Hydrated[0].Path)
	require.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", got.Hydrated[0].NoteID)
	require.Equal(t, "Raw Note", got.Hydrated[0].Title)
	require.Equal(t, "raw-note", got.Hydrated[0].Slug)

	require.FileExists(t, filepath.Join(importRoot, "mnemonic.toml"))
	require.FileExists(t, filepath.Join(cwd, "state", "mnemonic", "projects", "manifest-uuid-0000-0000-000000000001", "index.sqlite"))

	data, err := os.ReadFile(filepath.Join(importRoot, "raw-note.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", note.MnemonicNoteID)
	require.Equal(t, "Raw Note", note.Title)
	require.Equal(t, "raw-note", note.Slug)
	require.Contains(t, string(note.Body), "Body.")
}

func TestProjectImportCommandPreservesCustomFrontmatter(t *testing.T) {
	_, importRoot := seedRawImportDir(t)
	raw := []byte("---\nauthor: jane\nstatus: draft\n---\n# Custom Title\n\nBody.\n")
	require.NoError(t, os.WriteFile(filepath.Join(importRoot, "custom.md"), raw, 0o644))

	restore := clock.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		"manifest-uuid-0000-0000-000000000002",
		"22222222-3333-4444-5555-666666666666",
	))
	t.Cleanup(restore)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(importRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "import", ".", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	data, err := os.ReadFile(filepath.Join(importRoot, "custom.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, "22222222-3333-4444-5555-666666666666", note.MnemonicNoteID)
	require.Equal(t, "Custom Title", note.Title)
	require.Equal(t, "custom-title", note.Slug)
	require.Equal(t, "jane", note.Frontmatter["author"])
	require.Equal(t, "draft", note.Frontmatter["status"])
	require.Contains(t, string(note.Body), "Body.")
}

func TestProjectImportCommandDryRunDoesNotWrite(t *testing.T) {
	_, importRoot := seedRawImportDir(t)
	rawContent := []byte("# Dry Note\n\nBody.\n")
	require.NoError(t, os.WriteFile(filepath.Join(importRoot, "dry-note.md"), rawContent, 0o644))

	restore := clock.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		"manifest-uuid-0000-0000-000000000003",
		"ffffffff-0000-0000-0000-000000000000",
	))
	t.Cleanup(restore)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(importRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "import", ".", "--dry-run", "--json")
	require.NoError(t, result.Err, "project import returned error\nstderr: %s", result.Stderr)

	var got projectImportOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "skipped", got.IndexStatus)
	require.True(t, got.ManifestCreated)
	require.Len(t, got.Hydrated, 1)
	require.Equal(t, "skipped", got.IndexStatus)

	require.NoFileExists(t, filepath.Join(importRoot, "mnemonic.toml"))

	data, err := os.ReadFile(filepath.Join(importRoot, "dry-note.md"))
	require.NoError(t, err)
	require.Equal(t, rawContent, data)
}

func TestProjectImportCommandReturnsNotFoundForMissingPath(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "import", "missing")
	require.Error(t, result.Err, "project import error = nil, want not found")
	require.Equal(t, 3, ExitCodeForError(result.Err))
}

func seedRawImportDir(t *testing.T) (string, string) {
	t.Helper()

	cwd := testutil.CleanEnvForTest(t)
	importRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(importRoot, 0o755))
	return cwd, importRoot
}

func seedRawNote(t *testing.T, root, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o644))
}

// Ensure manifestfmt import is used (referenced in helpers above).
var _ = manifestfmt.NewMnemonicManifest
