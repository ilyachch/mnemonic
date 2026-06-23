package maintsvc

import (
	"context"
	"errors"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
)

// IndexService is the runtime index surface used by maintenance operations.
type IndexService interface {
	Rebuild(context.Context) (indexsvc.RebuildOutput, error)
	Doctor(context.Context) (indexsvc.DoctorOutput, error)
}

// Runtime is the runtime surface required by maintenance operations.
type Runtime interface {
	IndexService() IndexService
}

// RuntimeFactory builds a runtime for one selected knowledge base.
type RuntimeFactory func(context.Context, kb.KnowledgeBase) (Runtime, error)

// Service owns maintenance operations across many knowledge bases.
type Service struct {
	Catalog        *catalogsvc.Service
	RuntimeFactory RuntimeFactory
}

// ProjectResult captures the outcome for one project during maintenance.
type ProjectResult struct {
	ProjectID string `json:"project_id"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ReindexAllResult aggregates the reindex-all result.
type ReindexAllResult struct {
	Total    int             `json:"total"`
	Indexed  int             `json:"indexed"`
	Failed   int             `json:"failed,omitempty"`
	Skipped  int             `json:"skipped,omitempty"`
	Projects []ProjectResult `json:"projects,omitempty"`
}

// DoctorAllResult aggregates the doctor-all result.
type DoctorAllResult struct {
	Total        int             `json:"total"`
	Ok           int             `json:"ok"`
	Warning      int             `json:"warning,omitempty"`
	NeedsReindex int             `json:"needs_reindex,omitempty"`
	Failed       int             `json:"failed,omitempty"`
	Skipped      int             `json:"skipped,omitempty"`
	Projects     []ProjectResult `json:"projects,omitempty"`
}

// New constructs the maintenance service.
func New(catalog *catalogsvc.Service, runtimeFactory RuntimeFactory) *Service {
	return &Service{Catalog: catalog, RuntimeFactory: runtimeFactory}
}

// ReindexAll rebuilds the indexes for every catalog project.
func (s Service) ReindexAll(ctx context.Context) (ReindexAllResult, error) {
	entries, err := s.catalogEntries()
	if err != nil {
		return ReindexAllResult{}, err
	}

	result := ReindexAllResult{Total: len(entries)}
	for _, item := range entries {
		projectResult := ProjectResult{
			ProjectID: item.ProjectID,
			Slug:      item.Slug,
		}

		resolved, err := s.resolveKnowledgeBase(item.Slug)
		if err != nil {
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		runtime, err := s.runtimeFor(ctx, resolved)
		if err != nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		indexService := runtime.IndexService()
		if indexService == nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = "runtime index service is not configured"
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		output, err := indexService.Rebuild(ctx)
		if err != nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		projectResult.ProjectID = output.KBID
		projectResult.Status = output.Status
		if projectResult.Status == "" {
			projectResult.Status = "ok"
		}
		result.Indexed++
		result.Projects = append(result.Projects, projectResult)
	}

	return result, nil
}

// DoctorAll runs the runtime doctor checks for every catalog project.
func (s Service) DoctorAll(ctx context.Context) (DoctorAllResult, error) {
	entries, err := s.catalogEntries()
	if err != nil {
		return DoctorAllResult{}, err
	}

	result := DoctorAllResult{Total: len(entries)}
	for _, item := range entries {
		projectResult := ProjectResult{
			ProjectID: item.ProjectID,
			Slug:      item.Slug,
		}

		resolved, err := s.resolveKnowledgeBase(item.Slug)
		if err != nil {
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		runtime, err := s.runtimeFor(ctx, resolved)
		if err != nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		indexService := runtime.IndexService()
		if indexService == nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = "runtime index service is not configured"
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		output, err := indexService.Doctor(ctx)
		if err != nil {
			projectResult.ProjectID = resolved.ID
			projectResult.Status = "error"
			projectResult.Error = err.Error()
			result.Failed++
			result.Projects = append(result.Projects, projectResult)
			continue
		}

		projectResult.ProjectID = resolved.ID
		projectResult.Status = output.Status
		switch output.Status {
		case "needs_reindex":
			result.NeedsReindex++
		case "warning":
			result.Warning++
		case "ok", "":
			result.Ok++
			if projectResult.Status == "" {
				projectResult.Status = "ok"
			}
		default:
			result.Ok++
		}
		if projectResult.Status == "" {
			projectResult.Status = "ok"
		}
		result.Projects = append(result.Projects, projectResult)
	}

	return result, nil
}

func (s Service) catalogEntries() ([]catalogsvc.ListItem, error) {
	if s.Catalog == nil {
		return nil, errors.New("catalog service is not configured")
	}
	projects, err := s.Catalog.List()
	if err != nil {
		return nil, err
	}
	return projects.Projects, nil
}

func (s Service) resolveKnowledgeBase(selector string) (kb.KnowledgeBase, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return kb.KnowledgeBase{}, errors.New("project selector is required")
	}
	return s.Catalog.Resolve(selector)
}

func (s Service) runtimeFor(ctx context.Context, resolved kb.KnowledgeBase) (Runtime, error) {
	if s.RuntimeFactory == nil {
		return nil, errors.New("runtime factory is not configured")
	}
	return s.RuntimeFactory(ctx, resolved)
}
