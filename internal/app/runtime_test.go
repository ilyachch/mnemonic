package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewRuntimeAppStoresKnowledgeBase(t *testing.T) {
	resolved := kb.KnowledgeBase{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Name:         "Demo",
		Slug:         "demo",
		Kind:         "central",
		RootDir:      "/tmp/demo",
		RepoRootDir:  "/tmp",
		ManifestPath: "/tmp/demo/mnemonic.toml",
		StateDir:     "/tmp/state",
		IndexPath:    "/tmp/state/index.sqlite",
	}

	runtime, err := NewRuntimeApp(RuntimeInput{KB: resolved})
	require.NoError(t, err)
	require.Equal(t, resolved, runtime.KB)
	require.NotNil(t, runtime.Services.Notes)
	require.NotNil(t, runtime.Services.Search)
	require.NotNil(t, runtime.Services.Index)
	require.Equal(t, resolved.RootDir, runtime.Services.Notes.Notes.RootDir)
	require.Equal(t, resolved.IndexPath, runtime.Services.Notes.Index.IndexPath)
	require.Equal(t, resolved.RootDir, runtime.Services.Notes.Index.RootDir)
	require.Equal(t, resolved.ID, runtime.Services.Notes.Index.KBID)
	require.Equal(t, resolved, runtime.Services.Search.KB)
	require.Equal(t, resolved, runtime.Services.Index.KB)

	typ := reflect.TypeOf(*runtime)
	require.Equal(t, 2, typ.NumField())
	require.Equal(t, "KB", typ.Field(0).Name)
	require.Equal(t, "Services", typ.Field(1).Name)
}

func TestBootstrapRuntimeResolvesSelector(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	slug := "demo"
	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440001"
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Now().UTC()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	bootstrap, err := New(Input{})
	require.NoError(t, err)

	runtime, err := bootstrap.Runtime(context.Background(), "  "+slug+"  ")
	require.NoError(t, err)
	require.Equal(t, manifest.ProjectID, runtime.KB.ID)
	require.Equal(t, manifest.Name, runtime.KB.Name)
	require.Equal(t, slug, runtime.KB.Slug)
	require.Equal(t, "central", runtime.KB.Kind)
	require.Equal(t, projectDir, runtime.KB.RootDir)
	require.Equal(t, filepath.Join(projectDir, "mnemonic.toml"), runtime.KB.ManifestPath)
	require.Equal(t, filepath.Join(bootstrap.Paths.StateHome, "mnemonic", "projects", manifest.ProjectID), runtime.KB.StateDir)
	require.Equal(t, filepath.Join(bootstrap.Paths.StateHome, "mnemonic", "projects", manifest.ProjectID, "index.sqlite"), runtime.KB.IndexPath)
}

func TestBootstrapRuntimeReturnsUsageErrorForEmptySelector(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	bootstrap, err := New(Input{})
	require.NoError(t, err)

	_, err = bootstrap.Runtime(context.Background(), "   ")
	require.Error(t, err)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeCLIUsage, appErr.Code)
}
