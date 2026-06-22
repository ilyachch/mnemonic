package catalogsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	registry "github.com/ilyachch/mnemonic/internal/store/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestResolveTrimsSelectorAndReturnsUsageError(t *testing.T) {
	svc := Service{MemoriesHome: t.TempDir(), StateHome: t.TempDir()}

	_, err := svc.Resolve("   ")
	require.Error(t, err)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeCLIUsage, appErr.Code)
}

func TestResolveBuildsKnowledgeBase(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	slug := "demo"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := registry.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, registry.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}
	resolved, err := svc.Resolve("  " + slug + "  ")
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, resolved.ID)
	require.Equal(t, manifest.Name, resolved.Name)
	require.Equal(t, slug, resolved.Slug)
	require.Equal(t, "central", resolved.Kind)
	require.Equal(t, projectDir, resolved.RootDir)
	require.Equal(t, projectDir, resolved.RepoRootDir)
	require.Equal(t, filepath.Join(projectDir, "mnemonic.toml"), resolved.ManifestPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID), resolved.StateDir)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"), resolved.IndexPath)
}

func TestInitCreatesCentralProjectAndIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	cwd := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440010"},
	})
	t.Cleanup(restore)

	desc := "Central knowledge base"
	result, err := svc.Init(context.Background(), InitInput{
		WorkingDir:  cwd,
		Name:        "Backend",
		Description: desc,
		Mode:        InitModeCentral,
	})
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440010", result.ID)
	require.Equal(t, "Backend", result.Name)
	require.Equal(t, "backend", result.Slug)
	require.Equal(t, "central", result.Kind)
	require.Equal(t, filepath.Join(memoriesHome, "backend"), result.RootDir)
	require.Equal(t, filepath.Join(memoriesHome, "backend"), result.RepoRootDir)
	require.Equal(t, filepath.Join(memoriesHome, "backend", "mnemonic.toml"), result.ManifestPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID), result.StateDir)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID, "index.sqlite"), result.IndexPath)
	require.Equal(t, "ok", result.IndexStatus)
	require.Empty(t, result.IndexError)
	require.FileExists(t, result.IndexPath)

	manifestData, err := os.ReadFile(result.ManifestPath)
	require.NoError(t, err)
	manifest, err := registry.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, desc, manifest.Description)
}

func TestInitCreatesLocalProjectAndIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	cwd := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440011"},
	})
	t.Cleanup(restore)

	result, err := svc.Init(context.Background(), InitInput{
		WorkingDir:  cwd,
		Name:        "Personal",
		Description: "Local knowledge base",
		Mode:        InitModeLocal,
	})
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440011", result.ID)
	require.Equal(t, "Personal", result.Name)
	require.Equal(t, "personal", result.Slug)
	require.Equal(t, "local", result.Kind)
	require.Equal(t, filepath.Join(cwd, ".mnemonic-memories", "personal"), result.RootDir)
	require.Equal(t, filepath.Join(cwd, ".mnemonic-memories"), result.RepoRootDir)
	require.Equal(t, filepath.Join(cwd, ".mnemonic-memories", "personal", "mnemonic.toml"), result.ManifestPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID), result.StateDir)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID, "index.sqlite"), result.IndexPath)
	require.Equal(t, "ok", result.IndexStatus)
	require.Empty(t, result.IndexError)
	require.FileExists(t, filepath.Join(memoriesHome, "personal.toml"))
	require.FileExists(t, result.IndexPath)
}

func TestInitRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	cwd := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440012", "550e8400-e29b-41d4-a716-446655440013"},
	})
	t.Cleanup(restore)

	_, err := svc.Init(context.Background(), InitInput{
		WorkingDir: cwd,
		Name:       "Backend",
		Mode:       InitModeLocal,
	})
	require.NoError(t, err)

	_, err = svc.Init(context.Background(), InitInput{
		WorkingDir: cwd,
		Name:       "backend",
		Mode:       InitModeLocal,
	})
	require.Error(t, err)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeAmbiguous, appErr.Code)
	require.Contains(t, err.Error(), `project slug "backend" already exists`)
}

func TestListAndShowShapeRegistryData(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()

	centralSlug := "backend"
	centralDir := filepath.Join(memoriesHome, centralSlug)
	require.NoError(t, os.MkdirAll(centralDir, 0o755))
	centralManifest := registry.NewMnemonicManifest()
	centralManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	centralManifest.Name = "Backend"
	centralManifest.Slug = centralSlug
	centralManifest.MarkdownFormatVersion = 1
	centralManifest.CreatedAt = time.Now().UTC()
	centralManifest.UpdatedAt = centralManifest.CreatedAt
	require.NoError(t, registry.WriteMnemonicManifest(filepath.Join(centralDir, "mnemonic.toml"), centralManifest))

	repoRoot := t.TempDir()
	localSlug := "personal"
	localManifestDir := filepath.Join(repoRoot, ".mnemonic-memories", localSlug)
	require.NoError(t, os.MkdirAll(localManifestDir, 0o755))
	localManifest := registry.NewMnemonicManifest()
	localManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440002"
	localManifest.Name = "Personal"
	localManifest.Slug = localSlug
	localManifest.Type = registry.ManifestTypeLocal
	localManifest.MarkdownFormatVersion = 1
	localManifest.CreatedAt = time.Now().UTC()
	localManifest.UpdatedAt = localManifest.CreatedAt
	require.NoError(t, registry.WriteMnemonicManifest(filepath.Join(localManifestDir, "mnemonic.toml"), localManifest))
	require.NoError(t, registry.WritePointerFile(filepath.Join(memoriesHome, localSlug+".toml"), &registry.PointerFile{ManifestPath: filepath.Join(localManifestDir, "mnemonic.toml")}))

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}
	list, err := svc.List()
	require.NoError(t, err)
	require.Len(t, list.Projects, 2)

	gotBySlug := map[string]ListItem{}
	for _, item := range list.Projects {
		gotBySlug[item.Slug] = item
	}

	backend := gotBySlug[centralSlug]
	require.Equal(t, centralManifest.ProjectID, backend.ProjectID)
	require.Equal(t, "Backend", backend.Name)
	require.Equal(t, "central", backend.Type)
	require.Equal(t, "ok", backend.Status)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", centralManifest.ProjectID), backend.StatePath)

	personal := gotBySlug[localSlug]
	require.Equal(t, localManifest.ProjectID, personal.ProjectID)
	require.Equal(t, "Personal", personal.Name)
	require.Equal(t, "local", personal.Type)
	require.Equal(t, "ok", personal.Status)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", localManifest.ProjectID), personal.StatePath)

	shown, err := svc.Show("  " + centralSlug + " ")
	require.NoError(t, err)
	require.Equal(t, centralManifest.ProjectID, shown.ProjectID)
	require.Equal(t, "Backend", shown.Name)
	require.Equal(t, centralSlug, shown.Slug)
	require.Equal(t, "central", shown.Type)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", centralManifest.ProjectID), shown.StateHome)
	require.Equal(t, centralDir, shown.Location.MemoriesAbs)
	require.Equal(t, filepath.Join(centralDir, "mnemonic.toml"), shown.Location.ManifestAbs)
	require.Equal(t, centralDir, shown.Location.RepoRootAbs)
}

func testRegistryStore(memoriesHome string) registry.Store {
	return registry.New(memoriesHome, func(path string) (registry.ManifestData, error) {
		manifest, err := registry.ParseMnemonicManifestFromFile(path)
		if err != nil {
			return registry.ManifestData{}, err
		}
		kind := "central"
		if manifest.Type == string(registry.ManifestTypeLocal) {
			kind = "local"
		}
		return registry.ManifestData{
			ProjectID: manifest.ProjectID,
			Name:      manifest.Name,
			Slug:      manifest.Slug,
			Type:      kind,
		}, nil
	}, func(data []byte) (string, error) {
		pointer, err := registry.ParsePointerFile(data)
		if err != nil {
			return "", err
		}
		return pointer.ManifestPath, nil
	})
}

type testClock struct {
	now   time.Time
	uuids []string
}

func (c testClock) Now() time.Time {
	return c.now
}

func (c *testClock) UUID() string {
	if len(c.uuids) == 0 {
		return ""
	}
	uuid := c.uuids[0]
	c.uuids = c.uuids[1:]
	return uuid
}

func TestImportRemoveAndSlugs(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	manifest := registry.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440003"
	manifest.Name = "Import Demo"
	manifest.Slug = "import-demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, registry.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	imported, err := svc.Import(ImportInput{Path: repoRoot})
	require.NoError(t, err)
	require.Equal(t, 1, imported.Imported)
	require.Equal(t, 1, imported.Indexed)
	require.Len(t, imported.Candidates, 1)
	require.FileExists(t, filepath.Join(memoriesHome, manifest.Slug+".toml"))
	require.FileExists(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"))

	slugged, err := svc.Slugs()
	require.NoError(t, err)
	require.Equal(t, []string{manifest.Slug}, slugged)

	projectDir := filepath.Join(memoriesHome, manifest.Slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	removedManifest := registry.NewMnemonicManifest()
	removedManifest.ProjectID = manifest.ProjectID
	removedManifest.Name = manifest.Name
	removedManifest.Slug = manifest.Slug
	removedManifest.MarkdownFormatVersion = 1
	removedManifest.CreatedAt = time.Now().UTC()
	removedManifest.UpdatedAt = removedManifest.CreatedAt
	removedManifest.Generator.App = "mnemonic"
	require.NoError(t, registry.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), removedManifest))

	stateDir := filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID)
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite"), []byte("db"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite-wal"), []byte("wal"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite-shm"), []byte("shm"), 0o644))

	removed, err := svc.Remove(manifest.Slug, true)
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, removed.ProjectID)
	require.True(t, removed.RegistryRemoved)
	require.True(t, removed.IndexDeleted)
	require.True(t, removed.MarkdownDeleted)
	require.True(t, removed.FullWipe)
	_, err = os.Stat(filepath.Join(projectDir, "mnemonic.toml"))
	require.Error(t, err)
	_, err = os.Stat(stateDir)
	require.Error(t, err)
}
