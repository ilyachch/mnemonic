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

func TestProjectSyncCommandHydratesRawNotesAndSkipsIndexed(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	projectRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := manifestfmt.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440100"
	manifest.Name = "Synced"
	manifest.Slug = "synced"
	manifest.Type = manifestfmt.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))
	require.NoError(t, manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "synced.toml"), &manifestfmt.PointerFile{ManifestPath: filepath.Join(projectRoot, "mnemonic.toml")}))

	indexedNote := markdown.Note{
		MnemonicNoteID: "existing-id-0001",
		Title:          "Indexed",
		Slug:           "indexed",
		CreatedAt:      time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Body:           []byte("# Indexed\n"),
	}
	indexedRendered, err := markdown.RenderNote(indexedNote)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "indexed.md"), indexedRendered, 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "raw.md"), []byte("# Raw Note\n\nBody.\n"), 0o644))

	restore := clock.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	))
	t.Cleanup(restore)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "sync", "synced", "--json")
	require.NoError(t, result.Err, "project sync returned error\nstderr: %s", result.Stderr)

	var got projectSyncOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Equal(t, "synced", got.Slug)
	require.Len(t, got.Hydrated, 1)
	require.Equal(t, "raw.md", got.Hydrated[0].Path)
	require.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", got.Hydrated[0].NoteID)
	require.Equal(t, "Raw Note", got.Hydrated[0].Title)
	require.Equal(t, []string{"indexed.md"}, got.Skipped)
	require.Equal(t, "ok", got.IndexStatus)

	data, err := os.ReadFile(filepath.Join(projectRoot, "raw.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	require.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", note.MnemonicNoteID)

	indexedData, err := os.ReadFile(filepath.Join(projectRoot, "indexed.md"))
	require.NoError(t, err)
	require.Equal(t, indexedRendered, indexedData)
}

func TestProjectSyncCommandProcessesOnlyExplicitFile(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	projectRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := manifestfmt.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440200"
	manifest.Name = "Targeted"
	manifest.Slug = "targeted"
	manifest.Type = manifestfmt.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))
	require.NoError(t, manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "targeted.toml"), &manifestfmt.PointerFile{ManifestPath: filepath.Join(projectRoot, "mnemonic.toml")}))

	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "a.md"), []byte("# A Note\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "b.md"), []byte("# B Note\n"), 0o644))

	restore := clock.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		"bbbbbbbb-cccc-dddd-eeee-ffffffffffff",
	))
	t.Cleanup(restore)

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "sync", "targeted", "b.md", "--json")
	require.NoError(t, result.Err, "project sync returned error\nstderr: %s", result.Stderr)

	var got projectSyncOutput
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "failed to decode JSON\nstdout: %s", result.Stdout)
	require.Len(t, got.Hydrated, 1)
	require.Equal(t, "b.md", got.Hydrated[0].Path)

	aData, err := os.ReadFile(filepath.Join(projectRoot, "a.md"))
	require.NoError(t, err)
	require.Equal(t, "# A Note\n", string(aData))
}

func TestProjectSyncCommandRejectsFileOutsideRoot(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(cwd, "memories")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	projectRoot := filepath.Join(cwd, "repo")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := manifestfmt.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440300"
	manifest.Name = "Outside"
	manifest.Slug = "outside"
	manifest.Type = manifestfmt.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))
	require.NoError(t, manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "outside.toml"), &manifestfmt.PointerFile{ManifestPath: filepath.Join(projectRoot, "mnemonic.toml")}))

	externalDir := t.TempDir()
	externalFile := filepath.Join(externalDir, "external.md")
	require.NoError(t, os.WriteFile(externalFile, []byte("# External\n"), 0o644))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	result := executeCommand("project", "sync", "outside", externalFile)
	require.Error(t, result.Err, "project sync error = nil, want validation error")
	require.Equal(t, 2, ExitCodeForError(result.Err))
}
