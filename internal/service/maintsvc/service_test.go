package maintsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/stretchr/testify/require"
)

func TestReindexAllVisitsEveryProject(t *testing.T) {
	testCatalogParsers(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	projects := []projectSpec{
		createMaintProject(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440001"),
		createMaintProject(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440002"),
		createMaintProject(t, memoriesHome, "charlie", "550e8400-e29b-41d4-a716-446655440003"),
	}

	svc := Service{
		Catalog: &catalogsvc.Service{
			MemoriesHome: memoriesHome,
			StateHome:    stateHome,
		},
		RuntimeFactory: func(ctx context.Context, resolved kb.KnowledgeBase) (Runtime, error) {
			switch resolved.Slug {
			case "bravo":
				return fakeMaintRuntime{index: fakeMaintIndexService{
					rebuildFn: func(context.Context) (indexsvc.RebuildOutput, error) {
						return indexsvc.RebuildOutput{}, errors.New("rebuild failed")
					},
					doctorFn: func(context.Context) (indexsvc.DoctorOutput, error) {
						return indexsvc.DoctorOutput{}, nil
					},
				}}, nil
			default:
				return fakeMaintRuntime{index: fakeMaintIndexService{
					rebuildFn: func(context.Context) (indexsvc.RebuildOutput, error) {
						return indexsvc.RebuildOutput{KBID: resolved.ID, Status: "ok"}, nil
					},
					doctorFn: func(context.Context) (indexsvc.DoctorOutput, error) {
						return indexsvc.DoctorOutput{Status: "ok"}, nil
					},
				}}, nil
			}
		},
	}

	result, err := svc.ReindexAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, result.Indexed)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Projects, 3)

	got := map[string]ProjectResult{}
	for _, project := range result.Projects {
		got[project.Slug] = project
	}

	require.Equal(t, "ok", got["alpha"].Status)
	require.Equal(t, "error", got["bravo"].Status)
	require.Equal(t, "rebuild failed", got["bravo"].Error)
	require.Equal(t, "ok", got["charlie"].Status)
	require.Equal(t, projects[0].id, got["alpha"].ProjectID)
	require.Equal(t, projects[1].id, got["bravo"].ProjectID)
	require.Equal(t, projects[2].id, got["charlie"].ProjectID)
}

func TestDoctorAllVisitsEveryProject(t *testing.T) {
	testCatalogParsers(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	projects := []projectSpec{
		createMaintProject(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440011"),
		createMaintProject(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440012"),
		createMaintProject(t, memoriesHome, "charlie", "550e8400-e29b-41d4-a716-446655440013"),
	}

	svc := Service{
		Catalog: &catalogsvc.Service{
			MemoriesHome: memoriesHome,
			StateHome:    stateHome,
		},
		RuntimeFactory: func(ctx context.Context, resolved kb.KnowledgeBase) (Runtime, error) {
			switch resolved.Slug {
			case "alpha":
				return fakeMaintRuntime{index: fakeMaintIndexService{
					rebuildFn: func(context.Context) (indexsvc.RebuildOutput, error) {
						return indexsvc.RebuildOutput{}, nil
					},
					doctorFn: func(context.Context) (indexsvc.DoctorOutput, error) {
						return indexsvc.DoctorOutput{Status: "ok"}, nil
					},
				}}, nil
			case "bravo":
				return fakeMaintRuntime{index: fakeMaintIndexService{
					rebuildFn: func(context.Context) (indexsvc.RebuildOutput, error) {
						return indexsvc.RebuildOutput{}, nil
					},
					doctorFn: func(context.Context) (indexsvc.DoctorOutput, error) {
						return indexsvc.DoctorOutput{Status: "needs_reindex"}, nil
					},
				}}, nil
			default:
				return fakeMaintRuntime{index: fakeMaintIndexService{
					rebuildFn: func(context.Context) (indexsvc.RebuildOutput, error) {
						return indexsvc.RebuildOutput{}, nil
					},
					doctorFn: func(context.Context) (indexsvc.DoctorOutput, error) {
						return indexsvc.DoctorOutput{}, errors.New("doctor failed")
					},
				}}, nil
			}
		},
	}

	result, err := svc.DoctorAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Ok)
	require.Equal(t, 1, result.NeedsReindex)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Projects, 3)

	got := map[string]ProjectResult{}
	for _, project := range result.Projects {
		got[project.Slug] = project
	}

	require.Equal(t, "ok", got["alpha"].Status)
	require.Equal(t, "needs_reindex", got["bravo"].Status)
	require.Equal(t, "error", got["charlie"].Status)
	require.Equal(t, "doctor failed", got["charlie"].Error)
	require.Equal(t, projects[0].id, got["alpha"].ProjectID)
	require.Equal(t, projects[1].id, got["bravo"].ProjectID)
	require.Equal(t, projects[2].id, got["charlie"].ProjectID)
}

func TestCatalogEnumerationFailureReturnsImmediately(t *testing.T) {
	memoriesHome := filepath.Join(t.TempDir(), "registry-file")
	require.NoError(t, os.WriteFile(memoriesHome, []byte("not a directory"), 0o644))

	svc := Service{
		Catalog: &catalogsvc.Service{
			MemoriesHome: memoriesHome,
			StateHome:    t.TempDir(),
		},
		RuntimeFactory: func(context.Context, kb.KnowledgeBase) (Runtime, error) {
			t.Fatal("runtime factory should not be called")
			return nil, nil
		},
	}

	_, err := svc.ReindexAll(context.Background())
	require.Error(t, err)
}

type projectSpec struct {
	id   string
	slug string
}

type fakeMaintRuntime struct {
	index fakeMaintIndexService
}

func (r fakeMaintRuntime) IndexService() IndexService {
	return r.index
}

type fakeMaintIndexService struct {
	rebuildFn func(context.Context) (indexsvc.RebuildOutput, error)
	doctorFn  func(context.Context) (indexsvc.DoctorOutput, error)
}

func (s fakeMaintIndexService) Rebuild(ctx context.Context) (indexsvc.RebuildOutput, error) {
	return s.rebuildFn(ctx)
}

func (s fakeMaintIndexService) Doctor(ctx context.Context) (indexsvc.DoctorOutput, error) {
	return s.doctorFn(ctx)
}

func testCatalogParsers(t *testing.T) {
	t.Helper()

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
}

func createMaintProject(t *testing.T, memoriesHome, slug, projectID string) projectSpec {
	t.Helper()

	projectDir := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = projectID
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))

	return projectSpec{id: projectID, slug: slug}
}
