package catalogsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddRegistersStructuredProjectAndRebuildsIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440100"
	manifest.Name = "Structured"
	manifest.Slug = "structured"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	result, err := svc.Add(context.Background(), AddInput{Path: repoRoot})
	require.NoError(t, err)
	assert.Equal(t, "structured", result.Slug)
	assert.Equal(t, manifest.ProjectID, result.ProjectID)
	assert.Equal(t, 1, result.Indexed)
	assert.Equal(t, "ok", result.IndexStatus)
	assert.FileExists(t, filepath.Join(memoriesHome, "structured.toml"))
	assert.FileExists(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"))
}

func TestAddRejectsMissingManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()

	_, err := svc.Add(context.Background(), AddInput{Path: repoRoot})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mnemonic.toml not found")
}

func TestImportGeneratesManifestAndHydratesRawDirectory(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "raw-note.md"), []byte("# Raw Note\n\nBody.\n"), 0o644))

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		uuids: []string{"manifest-uuid-0000-0000-000000000001", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
	})
	t.Cleanup(restore)

	result, err := svc.Import(context.Background(), ImportInput{Path: repoRoot})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Indexed)
	assert.Equal(t, "ok", result.IndexStatus)
	assert.True(t, result.ManifestCreated)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "raw-note.md", result.Hydrated[0].Path)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", result.Hydrated[0].NoteID)
	assert.Equal(t, "Raw Note", result.Hydrated[0].Title)
	assert.Equal(t, "raw-note", result.Hydrated[0].Slug)

	assert.FileExists(t, filepath.Join(repoRoot, "mnemonic.toml"))
	assert.FileExists(t, filepath.Join(memoriesHome, filepath.Base(repoRoot)+".toml"))
	assert.FileExists(t, filepath.Join(stateHome, "mnemonic", "projects", "manifest-uuid-0000-0000-000000000001", "index.sqlite"))
}

func TestImportPreservesExistingManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440200"
	manifest.Name = "Existing"
	manifest.Slug = "existing"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		uuids: []string{},
	})
	t.Cleanup(restore)

	result, err := svc.Import(context.Background(), ImportInput{Path: repoRoot})
	require.NoError(t, err)
	assert.False(t, result.ManifestCreated)
	assert.Equal(t, "existing", result.Candidates[0].Slug)
	assert.Equal(t, manifest.ProjectID, result.Candidates[0].ProjectID)
}

func TestImportDryRunDoesNotWrite(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	rawContent := []byte("# Dry Note\n\nBody.\n")
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "dry-note.md"), rawContent, 0o644))

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		uuids: []string{"manifest-uuid-0000-0000-000000000002", "ffffffff-0000-0000-0000-000000000000"},
	})
	t.Cleanup(restore)

	result, err := svc.Import(context.Background(), ImportInput{Path: repoRoot, DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, "skipped", result.IndexStatus)
	assert.True(t, result.ManifestCreated)
	require.Len(t, result.Hydrated, 1)
	assert.True(t, result.DryRun)

	assert.NoFileExists(t, filepath.Join(repoRoot, "mnemonic.toml"))
	assert.NoFileExists(t, filepath.Join(memoriesHome, filepath.Base(repoRoot)+".toml"))

	data, err := os.ReadFile(filepath.Join(repoRoot, "dry-note.md"))
	require.NoError(t, err)
	assert.Equal(t, rawContent, data)
}

func TestImportRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "raw.md"), []byte("# Raw\n"), 0o644))

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC),
		uuids: []string{"manifest-uuid-0000-0000-000000000003", "11111111-2222-3333-4444-555555555555"},
	})
	t.Cleanup(restore)

	_, err := svc.Import(context.Background(), ImportInput{Path: repoRoot})
	require.NoError(t, err)

	_, err = svc.Import(context.Background(), ImportInput{Path: repoRoot})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
