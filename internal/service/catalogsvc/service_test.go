package catalogsvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
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

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome}
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

func TestListAndShowShapeRegistryData(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()

	centralSlug := "backend"
	centralDir := filepath.Join(memoriesHome, centralSlug)
	require.NoError(t, os.MkdirAll(centralDir, 0o755))
	centralManifest := project.NewMnemonicManifest()
	centralManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	centralManifest.Name = "Backend"
	centralManifest.Slug = centralSlug
	centralManifest.MarkdownFormatVersion = 1
	centralManifest.CreatedAt = time.Now().UTC()
	centralManifest.UpdatedAt = centralManifest.CreatedAt
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(centralDir, "mnemonic.toml"), centralManifest))

	repoRoot := t.TempDir()
	localSlug := "personal"
	localManifestDir := filepath.Join(repoRoot, ".mnemonic-memories", localSlug)
	require.NoError(t, os.MkdirAll(localManifestDir, 0o755))
	localManifest := project.NewMnemonicManifest()
	localManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440002"
	localManifest.Name = "Personal"
	localManifest.Slug = localSlug
	localManifest.Type = project.ManifestTypeLocal
	localManifest.MarkdownFormatVersion = 1
	localManifest.CreatedAt = time.Now().UTC()
	localManifest.UpdatedAt = localManifest.CreatedAt
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(localManifestDir, "mnemonic.toml"), localManifest))
	require.NoError(t, project.WritePointerFile(filepath.Join(memoriesHome, localSlug+".toml"), &project.PointerFile{ManifestPath: filepath.Join(localManifestDir, "mnemonic.toml")}))

	origManifestParser := registry.DefaultManifestParser
	origPointerParser := registry.DefaultPointerParser
	registry.DefaultManifestParser = func(path string) (registry.ManifestData, error) {
		manifest, err := project.ParseMnemonicManifestFromFile(path)
		if err != nil {
			return registry.ManifestData{}, err
		}
		kind := "central"
		if manifest.IsLocal() {
			kind = "local"
		}
		return registry.ManifestData{
			ProjectID: manifest.ProjectID,
			Name:      manifest.Name,
			Slug:      manifest.Slug,
			Type:      kind,
		}, nil
	}
	registry.DefaultPointerParser = func(data []byte) (string, error) {
		pointer, err := project.ParsePointerFile(data)
		if err != nil {
			return "", err
		}
		return pointer.ManifestPath, nil
	}
	t.Cleanup(func() {
		registry.DefaultManifestParser = origManifestParser
		registry.DefaultPointerParser = origPointerParser
	})

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome}
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

func TestImportRemoveAndSlugs(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome}

	repoRoot := t.TempDir()
	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440003"
	manifest.Name = "Import Demo"
	manifest.Slug = "import-demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	imported, err := svc.Import(ImportInput{Path: repoRoot})
	require.NoError(t, err)
	require.Equal(t, 1, imported.Imported)
	require.Len(t, imported.Candidates, 1)
	require.FileExists(t, filepath.Join(memoriesHome, manifest.Slug+".toml"))

	slugged, err := svc.Slugs()
	require.NoError(t, err)
	require.Equal(t, []string{manifest.Slug}, slugged)

	projectDir := filepath.Join(memoriesHome, manifest.Slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	removedManifest := project.NewMnemonicManifest()
	removedManifest.ProjectID = manifest.ProjectID
	removedManifest.Name = manifest.Name
	removedManifest.Slug = manifest.Slug
	removedManifest.MarkdownFormatVersion = 1
	removedManifest.CreatedAt = time.Now().UTC()
	removedManifest.UpdatedAt = removedManifest.CreatedAt
	removedManifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), removedManifest))

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
