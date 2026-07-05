package cli

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/maintsvc"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestProjectReindexAllReportsPartialFailure(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	createdAt := time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440021", createdAt)
	writeCentralProjectFixture(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440022", createdAt)

	boot := buildMaintAllBootstrap(t, memoriesHome, stateHome, "reindex")
	result := executeCommandWithBootstrap(boot, "project", "reindex", "--all", "--json")

	require.Error(t, result.Err)
	require.Equal(t, 1, ExitCodeForError(result.Err))
	require.Contains(t, result.Stdout, `"total": 2`)
	require.Contains(t, result.Stdout, `"slug": "bravo"`)
	require.Contains(t, result.Stdout, `"error": "rebuild failed"`)
}

func TestProjectReindexAllPrintsSummaryBeforeFailure(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	createdAt := time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440041", createdAt)
	writeCentralProjectFixture(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440042", createdAt)

	boot := buildMaintAllBootstrap(t, memoriesHome, stateHome, "reindex")
	result := executeCommandWithBootstrap(boot, "project", "reindex", "--all")

	require.Error(t, result.Err)
	require.Equal(t, 1, ExitCodeForError(result.Err))
	require.Contains(t, result.Stdout, "1 projects reindexed")
}

func TestProjectDoctorAllReportsPartialFailure(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	createdAt := time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440031", createdAt)
	writeCentralProjectFixture(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440032", createdAt)

	boot := buildMaintAllBootstrap(t, memoriesHome, stateHome, "doctor")
	result := executeCommandWithBootstrap(boot, "project", "doctor", "--all", "--json")

	require.Error(t, result.Err)
	require.Equal(t, 1, ExitCodeForError(result.Err))
	require.Contains(t, result.Stdout, `"total": 2`)
	require.Contains(t, result.Stdout, `"slug": "bravo"`)
	require.Contains(t, result.Stdout, `"error": "doctor failed"`)
}

func TestProjectDoctorAllPrintsSummaryBeforeFailure(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	createdAt := time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	writeCentralProjectFixture(t, memoriesHome, "alpha", "550e8400-e29b-41d4-a716-446655440051", createdAt)
	writeCentralProjectFixture(t, memoriesHome, "bravo", "550e8400-e29b-41d4-a716-446655440052", createdAt)

	boot := buildMaintAllBootstrap(t, memoriesHome, stateHome, "doctor")
	result := executeCommandWithBootstrap(boot, "project", "doctor", "--all")

	require.Error(t, result.Err)
	require.Equal(t, 1, ExitCodeForError(result.Err))
	require.Contains(t, result.Stdout, "2 projects checked")
}

func buildMaintAllBootstrap(t *testing.T, memoriesHome, stateHome, mode string) *app.Bootstrap {
	t.Helper()

	catalog := &catalogsvc.Service{
		MemoriesHome: memoriesHome,
		StateHome:    stateHome,
		Registry:     testCatalogStore(memoriesHome),
	}

	return &app.Bootstrap{
		Services: app.Services{
			Maint: &maintsvc.Service{
				Catalog: catalog,
				RuntimeFactory: func(ctx context.Context, resolved kb.KnowledgeBase, logger *slog.Logger) (maintsvc.Runtime, error) {
					_ = ctx
					if resolved.Slug == "bravo" {
						switch mode {
						case "reindex":
							return maintAllRuntime{index: maintAllIndexService{
								projectID:   resolved.ID,
								failRebuild: errors.New("rebuild failed"),
							}}, nil
						case "doctor":
							return maintAllRuntime{index: maintAllIndexService{
								projectID:  resolved.ID,
								failDoctor: errors.New("doctor failed"),
							}}, nil
						}
					}

					return maintAllRuntime{index: maintAllIndexService{projectID: resolved.ID}}, nil
				},
			},
		},
	}
}

type maintAllRuntime struct {
	index maintAllIndexService
}

func (r maintAllRuntime) IndexService() maintsvc.IndexService {
	return r.index
}

type maintAllIndexService struct {
	projectID   string
	failRebuild error
	failDoctor  error
}

func (s maintAllIndexService) Rebuild(context.Context) (indexsvc.RebuildOutput, error) {
	if s.failRebuild != nil {
		return indexsvc.RebuildOutput{}, s.failRebuild
	}
	return indexsvc.RebuildOutput{KBID: s.projectID, Status: "ok"}, nil
}

func (s maintAllIndexService) Doctor(context.Context) (indexsvc.DoctorOutput, error) {
	if s.failDoctor != nil {
		return indexsvc.DoctorOutput{}, s.failDoctor
	}
	return indexsvc.DoctorOutput{Status: "ok"}, nil
}
