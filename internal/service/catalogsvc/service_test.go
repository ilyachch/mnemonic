package catalogsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	registry "github.com/ilyachch/mnemonic/internal/store/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTrimsSelectorAndReturnsUsageError(t *testing.T) {
	svc := Service{MemoriesHome: t.TempDir(), StateHome: t.TempDir()}

	_, err := svc.Resolve("   ", nil)
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

	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.Description = "Catalog description"
	manifest.CustomInstructions = "Catalog instructions"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}
	resolved, err := svc.Resolve("  "+slug+"  ", nil)
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, resolved.ID)
	require.Equal(t, manifest.Name, resolved.Name)
	require.Equal(t, slug, resolved.Slug)
	require.Equal(t, "central", resolved.Kind)
	require.Equal(t, "Catalog description", resolved.Description)
	require.Equal(t, "Catalog instructions", resolved.CustomInstructions)
	require.Equal(t, projectDir, resolved.RootDir)
	require.Equal(t, projectDir, resolved.RepoRootDir)
	require.Equal(t, filepath.Join(projectDir, "mnemonic.toml"), resolved.ManifestPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID), resolved.StateDir)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"), resolved.IndexPath)
}

func TestKnowledgeBaseFromEntryPrefersEntryMetadataAndFallsBackToManifest(t *testing.T) {
	svc := Service{MemoriesHome: t.TempDir(), StateHome: t.TempDir()}
	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "mnemonic.toml")
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440099"
	manifest.Name = "Manifest Name"
	manifest.Slug = "demo"
	manifest.Description = "Manifest description"
	manifest.CustomInstructions = "Manifest instructions"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(manifestPath, manifest))

	tests := []struct {
		name              string
		entryDescription  string
		entryInstructions string
		wantDescription   string
		wantInstructions  string
	}{
		{
			name:              "uses entry metadata when present",
			entryDescription:  "Entry description",
			entryInstructions: "Entry instructions",
			wantDescription:   "Entry description",
			wantInstructions:  "Entry instructions",
		},
		{
			name:              "falls back when description missing",
			entryDescription:  "",
			entryInstructions: "Entry instructions",
			wantDescription:   "Manifest description",
			wantInstructions:  "Entry instructions",
		},
		{
			name:              "falls back when custom instructions missing",
			entryDescription:  "Entry description",
			entryInstructions: "",
			wantDescription:   "Entry description",
			wantInstructions:  "Manifest instructions",
		},
		{
			name:              "falls back when both missing",
			entryDescription:  "",
			entryInstructions: "",
			wantDescription:   "Manifest description",
			wantInstructions:  "Manifest instructions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := svc.knowledgeBaseFromEntry(registry.Entry{
				ProjectID:          "550e8400-e29b-41d4-a716-446655440099",
				Name:               "Demo",
				Slug:               "demo",
				Type:               "central",
				Description:        tc.entryDescription,
				CustomInstructions: tc.entryInstructions,
				ManifestPath:       manifestPath,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantDescription, resolved.Description)
			require.Equal(t, tc.wantInstructions, resolved.CustomInstructions)
			require.Equal(t, "Demo", resolved.Name)
			require.Equal(t, "demo", resolved.Slug)
			require.Equal(t, "central", resolved.Kind)
			require.Equal(t, manifestPath, resolved.ManifestPath)
		})
	}
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
	}, nil)
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
	manifest, err := manifestfmt.ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, desc, manifest.Description)
}

func TestInitReturnsStaleIndexWhenRebuildFails(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	cwd := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	blockingDir := filepath.Join(stateHome, "mnemonic")
	require.NoError(t, os.MkdirAll(blockingDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(blockingDir, "projects"), []byte("block rebuild"), 0o644))

	restore := clock.SetClock(&testClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440012"},
	})
	t.Cleanup(restore)

	result, err := svc.Init(context.Background(), InitInput{
		WorkingDir:  cwd,
		Name:        "Backend",
		Description: "Central knowledge base",
		Mode:        InitModeCentral,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440012", result.ID)
	require.Equal(t, "Backend", result.Name)
	require.Equal(t, "backend", result.Slug)
	require.Equal(t, "central", result.Kind)
	require.Equal(t, filepath.Join(memoriesHome, "backend"), result.RootDir)
	require.Equal(t, filepath.Join(memoriesHome, "backend"), result.RepoRootDir)
	require.Equal(t, filepath.Join(memoriesHome, "backend", "mnemonic.toml"), result.ManifestPath)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID), result.StateDir)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", result.ID, "index.sqlite"), result.IndexPath)
	require.Equal(t, "stale", result.IndexStatus)
	require.NotEmpty(t, result.IndexError)
	require.FileExists(t, result.ManifestPath)
	_, statErr := os.Stat(result.IndexPath)
	require.Error(t, statErr)
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
	}, nil)
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
	}, nil)
	require.NoError(t, err)

	_, err = svc.Init(context.Background(), InitInput{
		WorkingDir: cwd,
		Name:       "backend",
		Mode:       InitModeLocal,
	}, nil)
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
	centralManifest := manifestfmt.New()
	centralManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	centralManifest.Name = "Backend"
	centralManifest.Slug = centralSlug
	centralManifest.Description = "Central description"
	centralManifest.CustomInstructions = "Central instructions"
	centralManifest.MarkdownFormatVersion = 1
	centralManifest.CreatedAt = time.Now().UTC().Unix()
	centralManifest.UpdatedAt = centralManifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(centralDir, "mnemonic.toml"), centralManifest))

	repoRoot := t.TempDir()
	localSlug := "personal"
	localManifestDir := filepath.Join(repoRoot, ".mnemonic-memories", localSlug)
	require.NoError(t, os.MkdirAll(localManifestDir, 0o755))
	localManifest := manifestfmt.New()
	localManifest.ProjectID = "550e8400-e29b-41d4-a716-446655440002"
	localManifest.Name = "Personal"
	localManifest.Slug = localSlug
	localManifest.Type = manifestfmt.ManifestTypeLocal
	localManifest.Description = "Local description"
	localManifest.CustomInstructions = "Local instructions"
	localManifest.MarkdownFormatVersion = 1
	localManifest.CreatedAt = time.Now().UTC().Unix()
	localManifest.UpdatedAt = localManifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(localManifestDir, "mnemonic.toml"), localManifest))
	require.NoError(t, manifestfmt.WritePointerFile(filepath.Join(memoriesHome, localSlug+".toml"), &manifestfmt.PointerFile{ManifestPath: filepath.Join(localManifestDir, "mnemonic.toml")}))

	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}
	list, err := svc.List(nil)
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

	shown, err := svc.Show("  "+centralSlug+" ", nil)
	require.NoError(t, err)
	require.Equal(t, centralManifest.ProjectID, shown.ProjectID)
	require.Equal(t, "Backend", shown.Name)
	require.Equal(t, centralSlug, shown.Slug)
	require.Equal(t, "central", shown.Type)
	require.Equal(t, "Central description", shown.Description)
	require.Equal(t, "Central instructions", shown.CustomInstructions)
	require.Equal(t, filepath.Join(stateHome, "mnemonic", "projects", centralManifest.ProjectID), shown.StateHome)
	require.Equal(t, centralDir, shown.Location.MemoriesAbs)
	require.Equal(t, filepath.Join(centralDir, "mnemonic.toml"), shown.Location.ManifestAbs)
	require.Equal(t, centralDir, shown.Location.RepoRootAbs)
}

func testRegistryStore(memoriesHome string) registry.Store {
	return registry.New(memoriesHome, func(path string) (*manifestfmt.Manifest, error) {
		return manifestfmt.ParseMnemonicManifestFromFile(path)
	}, func(data []byte) (*manifestfmt.PointerFile, error) {
		return manifestfmt.ParsePointerFile(data)
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
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440003"
	manifest.Name = "Import Demo"
	manifest.Slug = "import-demo"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	imported, err := svc.Import(context.Background(), ImportInput{Path: repoRoot}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, imported.Imported)
	require.Equal(t, 1, imported.Indexed)
	require.Equal(t, "ok", imported.IndexStatus)
	require.Empty(t, imported.IndexErrors)
	require.Len(t, imported.Candidates, 1)
	require.FileExists(t, filepath.Join(memoriesHome, manifest.Slug+".toml"))
	require.FileExists(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"))

	slugged, err := svc.Slugs()
	require.NoError(t, err)
	require.Equal(t, []string{manifest.Slug}, slugged)

	projectDir := filepath.Join(memoriesHome, manifest.Slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	removedManifest := manifestfmt.New()
	removedManifest.ProjectID = manifest.ProjectID
	removedManifest.Name = manifest.Name
	removedManifest.Slug = manifest.Slug
	removedManifest.MarkdownFormatVersion = 1
	removedManifest.CreatedAt = time.Now().UTC().Unix()
	removedManifest.UpdatedAt = removedManifest.CreatedAt
	removedManifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), removedManifest))

	stateDir := filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID)
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite"), []byte("db"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite-wal"), []byte("wal"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "index.sqlite-shm"), []byte("shm"), 0o644))

	removed, err := svc.Remove(manifest.Slug, true, nil)
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

func TestImportReturnsSkippedIndexStatusForDryRun(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440010"
	manifest.Name = "Dry Run"
	manifest.Slug = "dry-run"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	imported, err := svc.Import(context.Background(), ImportInput{Path: repoRoot, DryRun: true}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, imported.Imported)
	require.Equal(t, 0, imported.Indexed)
	require.Equal(t, "skipped", imported.IndexStatus)
	require.Empty(t, imported.IndexErrors)
	require.Len(t, imported.Candidates, 1)
	require.NoFileExists(t, filepath.Join(stateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"))
}

func TestFinalizeImportIndexStatusReportsPartialFailures(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	okResult := seedImportedProject(t, memoriesHome, "550e8400-e29b-41d4-a716-446655440020", "ok-project", "ok")
	failResult := seedImportedProject(t, memoriesHome, "550e8400-e29b-41d4-a716-446655440021", "fail-project", "fail")
	require.NoError(t, os.MkdirAll(filepath.Join(stateHome, "mnemonic", "projects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateHome, "mnemonic", "projects", failResult.Candidates[0].ProjectID), []byte("block rebuild"), 0o644))

	result := svc.finalizeImportIndexStatus(context.Background(), ImportResult{
		Candidates: append(okResult.Candidates, failResult.Candidates...),
	}, nil)

	require.Equal(t, 1, result.Indexed)
	require.Equal(t, "stale", result.IndexStatus)
	require.Len(t, result.IndexErrors, 1)
	require.Equal(t, failResult.Candidates[0].ProjectID, result.IndexErrors[0].ProjectID)
	require.Equal(t, failResult.Candidates[0].Slug, result.IndexErrors[0].Slug)
	require.NotEmpty(t, result.IndexErrors[0].Error)
	require.FileExists(t, filepath.Join(stateHome, "mnemonic", "projects", okResult.Candidates[0].ProjectID, "index.sqlite"))
	_, err := os.Stat(filepath.Join(stateHome, "mnemonic", "projects", failResult.Candidates[0].ProjectID, "index.sqlite"))
	require.Error(t, err)
}

func TestFinalizeImportIndexStatusReportsAllFailures(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	firstResult := seedImportedProject(t, memoriesHome, "550e8400-e29b-41d4-a716-446655440030", "first-project", "first")
	secondResult := seedImportedProject(t, memoriesHome, "550e8400-e29b-41d4-a716-446655440031", "second-project", "second")
	require.NoError(t, os.MkdirAll(filepath.Join(stateHome, "mnemonic", "projects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateHome, "mnemonic", "projects", firstResult.Candidates[0].ProjectID), []byte("block rebuild"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateHome, "mnemonic", "projects", secondResult.Candidates[0].ProjectID), []byte("block rebuild"), 0o644))

	result := svc.finalizeImportIndexStatus(context.Background(), ImportResult{
		Candidates: append(firstResult.Candidates, secondResult.Candidates...),
	}, nil)

	require.Equal(t, 0, result.Indexed)
	require.Equal(t, "stale", result.IndexStatus)
	require.Len(t, result.IndexErrors, 2)
	require.ElementsMatch(t, []string{
		firstResult.Candidates[0].Slug,
		secondResult.Candidates[0].Slug,
	}, []string{
		result.IndexErrors[0].Slug,
		result.IndexErrors[1].Slug,
	})
}

func TestImportReturnsErrorWhenRegistrationFails(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	svc := Service{MemoriesHome: memoriesHome, StateHome: stateHome, Registry: testRegistryStore(memoriesHome)}

	repoRoot := t.TempDir()
	manifest := manifestfmt.New()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440050"
	manifest.Name = "Duplicate"
	manifest.Slug = "duplicate"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(repoRoot, "mnemonic.toml"), manifest))

	_, err := svc.Import(context.Background(), ImportInput{Path: repoRoot}, nil)
	require.NoError(t, err)
	_, err = svc.Import(context.Background(), ImportInput{Path: repoRoot}, nil)
	require.Error(t, err)
}

func seedImportedProject(t *testing.T, memoriesHome, projectID, name, slug string) ImportResult {
	t.Helper()

	repoRoot := t.TempDir()
	manifest := manifestfmt.New()
	manifest.ProjectID = projectID
	manifest.Name = name
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC().Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifestPath := filepath.Join(repoRoot, "mnemonic.toml")
	require.NoError(t, manifestfmt.WriteMnemonicManifest(manifestPath, manifest))

	pointerPath := filepath.Join(memoriesHome, slug+".toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(pointerPath), 0o755))
	require.NoError(t, manifestfmt.WritePointerFile(pointerPath, &manifestfmt.PointerFile{ManifestPath: manifestPath}))

	return ImportResult{
		Path:        repoRoot,
		Imported:    1,
		Candidates:  []ImportCandidate{{ProjectID: projectID, Name: name, Slug: slug, MemoriesPath: repoRoot, ManifestAbs: manifestPath}},
		IndexErrors: []ImportIndexError{},
	}
}

func TestWrapRegistryError(t *testing.T) {
	// nil -> nil
	require.Nil(t, wrapRegistryError(nil))

	// ErrNotFound -> apperr.NotFound
	notFoundErr := wrapRegistryError(registry.ErrNotFound{Slug: "test"})
	require.Error(t, notFoundErr)
	var appErr *apperr.Error
	require.ErrorAs(t, notFoundErr, &appErr)
	require.Equal(t, apperr.CodeNotFound, appErr.Code)

	// Unknown error passes through
	plain := errors.New("plain error")
	require.Equal(t, plain, wrapRegistryError(plain))
}

func TestRegistryPathForKind(t *testing.T) {
	assert.Equal(t, filepath.Join("/mem", "slug", "mnemonic.toml"), registryPathForKind("/mem", "central", "slug"))
	assert.Equal(t, filepath.Join("/mem", "slug.toml"), registryPathForKind("/mem", "local", "slug"))
	assert.Equal(t, "", registryPathForKind("/mem", "unknown", "slug"))
}

func TestResolveImportPath(t *testing.T) {
	tmp := t.TempDir()

	// Empty path defaults to current directory
	_, err := resolveImportPath(ImportInput{Path: ""})
	require.NoError(t, err)

	// Existing directory
	_, err = resolveImportPath(ImportInput{Path: tmp})
	require.NoError(t, err)

	// Non-existent path
	_, err = resolveImportPath(ImportInput{Path: "/nonexistent/path/12345"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
