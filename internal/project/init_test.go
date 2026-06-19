package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestInitRegularProjectWritesManifestAndRegistersRegistry(t *testing.T) {
	_ = testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

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

	// The .mnemonic anchor file should not be created in the project root.
	_, err := os.Stat(filepath.Join(cwd, ".mnemonic"))
	require.True(t, os.IsNotExist(err), ".mnemonic anchor file should not be created")

	manifestPath := filepath.Join(memoriesHome, "my-app", "mnemonic.toml")
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsed, err := ParseMnemonicManifest(manifestData)
	require.NoError(t, err)
	require.Equal(t, ManifestKindRegular, parsed.Kind)
	require.Equal(t, "my-app", parsed.Slug)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", parsed.ProjectID)

	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE removed_at IS NULL AND slug = ?`, "my-app").Scan(&count))
	require.Equal(t, 1, count)
}

func TestInitLocalProjectCreatesMemoriesDirAndRegistersRegistry(t *testing.T) {
	_ = testutil.CleanEnvForTest(t)
	cwd := t.TempDir()

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: t.TempDir(),
		Name:         "backend",
		Mode:         InitModeLocal,
	}))

	_, err := os.Stat(filepath.Join(cwd, ".mnemonic"))
	require.True(t, os.IsNotExist(err), ".mnemonic anchor file should not be created")

	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	_, err = os.Stat(localPath)
	require.NoError(t, err, "local memories directory missing")
}

func TestInitDetachedProjectWritesManifestAndRegistersRegistry(t *testing.T) {
	_ = testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

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

	_, err := os.Stat(filepath.Join(cwd, ".mnemonic"))
	require.True(t, os.IsNotExist(err), ".mnemonic anchor file should not be created")

	manifestPath := filepath.Join(memoriesHome, "personal", "mnemonic.toml")
	_, err = os.Stat(manifestPath)
	require.NoError(t, err)
}

func TestInitProjectRejectsDuplicateSlugInRegistry(t *testing.T) {
	_ = testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	require.NoError(t, InitRegularProject(InitRegularInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
	}))

	restore()
	restore = SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	err := InitRegularProject(InitRegularInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "Backend",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `project slug "backend" already exists`)
}

func TestInitProjectRejectsUnknownMode(t *testing.T) {
	_ = testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "demo",
		Mode:         InitMode("unknown"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown init mode")
}
