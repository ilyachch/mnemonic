package project

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestResolveProjectReturnsNotSelectedWhenSelectorAndEnvEmpty(t *testing.T) {
	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = ResolveProject(ResolveProjectInput{Registry: db})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project selected")
}

func TestResolveProjectReturnsNotFoundForUnknownSelector(t *testing.T) {
	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = ResolveProject(ResolveProjectInput{
		Registry:        db,
		ProjectSelector: "missing",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `project "missing" not found`)
}

func TestResolveProjectFromEnvOpensRegistry(t *testing.T) {
	testutil.CleanEnvForTest(t)

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      "demo",
		Slug:      "demo",
		Kind:      registry.ProjectKindLocal,
		CreatedAt: now,
		UpdatedAt: now,
		SeenAt:    now,
		Location: registry.ProjectLocationInput{
			RepoRootAbs: t.TempDir(),
			MemoriesAbs: t.TempDir(),
			SourceKind:  registry.ProjectSourceKindInit,
		},
	}))

	resolved, err := ResolveProjectFromEnv("demo")
	require.NoError(t, err)
	require.Equal(t, "demo", resolved.Project.Slug)
}

func TestResolveProjectRejectsNilRegistry(t *testing.T) {
	_, err := ResolveProject(ResolveProjectInput{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "registry database is required")
}

func TestResolveProjectPicksSelectorOverEnv(t *testing.T) {
	db, err := registry.OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	for _, slug := range []string{"alpha", "beta"} {
		memoriesHome := t.TempDir()
		require.NoError(t, registry.RegisterProject(db, registry.RegisterProjectInput{
			ProjectID: "550e8400-e29b-41d4-a716-4466554400" + slug[:1],
			Name:      slug,
			Slug:      slug,
			Kind:      registry.ProjectKindRegular,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				MemoriesAbs: memoriesHome,
				ManifestAbs: filepath.Join(memoriesHome, "mnemonic.toml"),
				SourceKind:  registry.ProjectSourceKindInit,
			},
		}))
	}

	resolved, err := ResolveProject(ResolveProjectInput{
		Registry:         db,
		ProjectSelector:  "beta",
		EnvironmentValue: "alpha",
	})
	require.NoError(t, err)
	require.Equal(t, "beta", resolved.Project.Slug)
}
