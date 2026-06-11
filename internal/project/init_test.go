package project

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestInitRegularProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitRegularProject(InitRegularInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "my-app",
	}))

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "my-app", "mnemonic.toml")

	_, err := os.Stat(projectPath)
	require.NoError(t, err)
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(memoriesHome, "my-app", "index.sqlite"))
	require.True(t, os.IsNotExist(err))

	projectData, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	parsedProject, err := ParseMnemonicFile(projectData)
	require.NoError(t, err)
	require.Len(t, parsedProject.Projects, 1)

	projectEntry := parsedProject.Projects[0]
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", projectEntry.ID)
	require.Equal(t, "my-app", projectEntry.Name)
	require.Equal(t, "my-app", projectEntry.Slug)
	require.Equal(t, ProjectKindRegular, projectEntry.Kind)
	require.Equal(t, "my-app", projectEntry.MemoriesPath)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, ManifestKindRegular, parsedManifest.Kind)
	require.Equal(t, "my-app", parsedManifest.Name)
	require.Equal(t, "my-app", parsedManifest.Slug)
	require.Equal(t, projectEntry.ID, parsedManifest.ProjectID)

	assertProjectRegistered(t, dataHome, projectEntry.ID, filepath.Join(memoriesHome, projectEntry.MemoriesPath), filepath.Join(memoriesHome, "my-app", "mnemonic.toml"))
}

func TestInitLocalProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}))

	projectPath := filepath.Join(cwd, ".mnemonic")
	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	manifestPath := filepath.Join(localPath, "mnemonic.toml")

	_, err := os.Stat(projectPath)
	require.NoError(t, err)
	_, err = os.Stat(localPath)
	require.NoError(t, err)
	_, err = os.Stat(manifestPath)
	require.True(t, os.IsNotExist(err))

	projectData, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	parsedProject, err := ParseMnemonicFile(projectData)
	require.NoError(t, err)
	require.Len(t, parsedProject.Projects, 1)
	projectEntry := parsedProject.Projects[0]
	require.Equal(t, ProjectKindLocal, projectEntry.Kind)
	require.Equal(t, filepath.Join(".mnemonic-memories", "backend"), projectEntry.MemoriesPath)

	assertProjectRegistered(t, dataHome, projectEntry.ID, filepath.Join(cwd, projectEntry.MemoriesPath), "")
}

func TestInitDetachedProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "personal",
		Mode:         InitModeDetached,
	}))

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "personal", "mnemonic.toml")

	_, err := os.Stat(projectPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsedManifest, err := ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, ManifestKindDetached, parsedManifest.Kind)
	require.Equal(t, "personal", parsedManifest.Slug)

	assertProjectRegistered(t, dataHome, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(memoriesHome, "personal"), manifestPath)
}

func TestInitLocalProjectAppendsExistingMnemonicFile(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}))

	restore()
	restore = SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "frontend",
		Mode:         InitModeLocal,
	}))

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	require.NoError(t, err)
	parsedProject, err := ParseMnemonicFile(projectData)
	require.NoError(t, err)
	require.Len(t, parsedProject.Projects, 2)

	require.True(t, parsedProject.CreatedAt.Equal(time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)))
	require.True(t, parsedProject.UpdatedAt.Equal(time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC)))

	first := parsedProject.Projects[0]
	require.Equal(t, "backend", first.Name)
	require.True(t, first.CreatedAt.Equal(time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)))

	second := parsedProject.Projects[1]
	require.Equal(t, "frontend", second.Name)
	require.True(t, second.CreatedAt.Equal(time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC)))

	assertProjectRegistered(t, dataHome, first.ID, filepath.Join(cwd, first.MemoriesPath), "")
	assertProjectRegistered(t, dataHome, second.ID, filepath.Join(cwd, second.MemoriesPath), "")
}

func TestInitLocalProjectRejectsDuplicateSlug(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}))

	restore()
	restore = SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "Backend",
		Mode:         InitModeLocal,
	})
	require.Error(t, err)
	require.Equal(t, `project slug "backend" already exists`, err.Error())

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	require.NoError(t, err)
	parsedProject, err := ParseMnemonicFile(projectData)
	require.NoError(t, err)
	require.Len(t, parsedProject.Projects, 1)

	assertProjectRegistered(t, dataHome, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(cwd, parsedProject.Projects[0].MemoriesPath), "")
}

func assertProjectRegistered(t *testing.T, dataHome, projectID, memoriesAbs, manifestAbs string) {
	t.Helper()

	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, registry.ApplySchema(db))

	var gotMemoriesAbs, gotManifestAbs sql.NullString
	require.NoError(t, db.QueryRow(`SELECT memories_abs, manifest_abs FROM project_locations WHERE project_id = ?`, projectID).Scan(&gotMemoriesAbs, &gotManifestAbs))
	require.Equal(t, memoriesAbs, gotMemoriesAbs.String)
	require.Equal(t, manifestAbs, gotManifestAbs.String)

	var needsReindex, indexPresent int
	require.NoError(t, db.QueryRow(`SELECT needs_reindex, index_present FROM project_status WHERE project_id = ?`, projectID).Scan(&needsReindex, &indexPresent))
	require.Equal(t, 1, needsReindex)
	require.Equal(t, 0, indexPresent)
}