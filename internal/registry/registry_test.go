package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestScanSkipsCentralDirectoriesWithoutManifest(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	projectDir := filepath.Join(memoriesHome, "backend")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifestPath := filepath.Join(projectDir, "mnemonic.toml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("project_id = \"550e8400-e29b-41d4-a716-446655440000\"\nname = \"backend\"\nslug = \"backend\"\n"), 0o644))

	origManifestParser := DefaultManifestParser
	DefaultManifestParser = func(path string) (ManifestData, error) {
		return ManifestData{
			ProjectID: "550e8400-e29b-41d4-a716-446655440000",
			Name:      "backend",
			Slug:      "backend",
			Type:      "central",
		}, nil
	}
	t.Cleanup(func() {
		DefaultManifestParser = origManifestParser
	})

	entries, issues, err := Scan(memoriesHome)
	require.NoError(t, err)
	require.Empty(t, issues)
	require.Len(t, entries, 1)

	require.NoError(t, os.Remove(manifestPath))

	entries, issues, err = Scan(memoriesHome)
	require.NoError(t, err)
	require.Empty(t, issues)
	require.Empty(t, entries)
}

func TestResolveFailsOnOrphanedLocalProject(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	pointerPath := filepath.Join(memoriesHome, "personal.toml")
	require.NoError(t, os.WriteFile(pointerPath, []byte(filepath.Join(memoriesHome, "missing", "mnemonic.toml")), 0o644))

	origManifestParser := DefaultManifestParser
	origPointerParser := DefaultPointerParser
	DefaultManifestParser = func(path string) (ManifestData, error) {
		return ManifestData{}, os.ErrNotExist
	}
	DefaultPointerParser = func(data []byte) (string, error) {
		return string(data), nil
	}
	t.Cleanup(func() {
		DefaultManifestParser = origManifestParser
		DefaultPointerParser = origPointerParser
	})

	entry, err := Resolve(memoriesHome, "personal")
	require.Error(t, err)
	require.Empty(t, entry)
	require.Contains(t, err.Error(), "Local manifest not found")
}

func TestFindByMemoriesRootNormalizesPaths(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	projectDir := filepath.Join(memoriesHome, "backend")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "mnemonic.toml"), []byte("project_id = \"550e8400-e29b-41d4-a716-446655440000\"\nname = \"backend\"\nslug = \"backend\"\n"), 0o644))

	origManifestParser := DefaultManifestParser
	DefaultManifestParser = func(path string) (ManifestData, error) {
		return ManifestData{
			ProjectID: "550e8400-e29b-41d4-a716-446655440000",
			Name:      "backend",
			Slug:      "backend",
			Type:      "central",
		}, nil
	}
	t.Cleanup(func() {
		DefaultManifestParser = origManifestParser
	})

	entry, found, err := FindByMemoriesRoot(memoriesHome, filepath.Join(projectDir, "."))
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "backend", entry.Slug)
}
