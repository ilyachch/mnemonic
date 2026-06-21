package catalogsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
)

// Service owns catalog-level project operations.
type Service struct {
	MemoriesHome string
	StateHome    string
}

// ListResult mirrors the project list payload.
type ListResult struct {
	Projects []ListItem `json:"projects"`
}

// ListItem mirrors the project list item payload.
type ListItem struct {
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
	MemoriesPath string `json:"memories_path"`
	StatePath    string `json:"state_path"`
	Status       string `json:"status"`
	Issue        string `json:"issue,omitempty"`
}

// ShowResult mirrors the project show payload.
type ShowResult struct {
	ProjectID string             `json:"project_id"`
	Name      string             `json:"name"`
	Slug      string             `json:"slug"`
	Type      string             `json:"type"`
	StateHome string             `json:"state_home"`
	Location  ShowLocationResult `json:"location"`
}

// ShowLocationResult describes the resolved project location.
type ShowLocationResult struct {
	MemoriesAbs string `json:"memories_abs"`
	ManifestAbs string `json:"manifest_abs"`
	RepoRootAbs string `json:"repo_root_abs"`
}

// RemoveResult mirrors the project remove payload.
type RemoveResult struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	RegistryRemoved bool   `json:"registry_removed"`
	IndexDeleted    bool   `json:"index_deleted"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	FullWipe        bool   `json:"full_wipe"`
}

// ImportInput mirrors the project import input.
type ImportInput = project.ImportInput

// ImportResult mirrors the project import result.
type ImportResult = project.ImportResult

// ImportCandidate mirrors the project import candidate.
type ImportCandidate = project.ImportCandidate

// InitInput mirrors the project init input.
type InitInput struct {
	WorkingDir  string
	Name        string
	Description string
	Mode        project.InitMode
}

// InitResult mirrors the project init payload.
type InitResult struct {
	kb.KnowledgeBase
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

// Resolve returns the selected knowledge base for a project selector.
func (s Service) Resolve(selector string) (kb.KnowledgeBase, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return kb.KnowledgeBase{}, apperr.CLIUsage("no project selected; specify --project or set MNEMONIC_PROJECT", nil)
	}

	entry, err := registry.Resolve(s.MemoriesHome, selector)
	if err != nil {
		return kb.KnowledgeBase{}, wrapRegistryError(err)
	}

	resolved, err := s.knowledgeBaseFromEntry(entry)
	if err != nil {
		return kb.KnowledgeBase{}, err
	}

	return resolved, nil
}

// List returns the registered projects with state paths and issue status.
func (s Service) List() (ListResult, error) {
	entries, issues, err := registry.Scan(s.MemoriesHome)
	if err != nil {
		return ListResult{}, fmt.Errorf("scan registry: %w", err)
	}

	issueMap := make(map[string]registry.Issue, len(issues))
	for _, issue := range issues {
		issueMap[issue.Slug] = issue
	}

	projects := make([]ListItem, 0, len(entries))
	for _, entry := range entries {
		item := ListItem{
			ProjectID:    entry.ProjectID,
			Name:         entry.Name,
			Slug:         entry.Slug,
			Type:         entry.Type,
			MemoriesPath: entry.MemoriesAbs,
			StatePath:    s.statePath(entry.ProjectID),
			Status:       "ok",
		}

		if issue, found := issueMap[entry.Slug]; found {
			if issue.Corrupt {
				item.Status = "[CORRUPTED]"
			} else if issue.Orphan {
				item.Status = "[ORPHANED/MISSING]"
			}
			item.Issue = issue.Error
			projects = append(projects, item)
			continue
		}

		resolved, err := s.knowledgeBaseFromEntry(entry)
		if err != nil {
			return ListResult{}, err
		}
		item.ProjectID = resolved.ID
		item.Name = resolved.Name
		item.Slug = resolved.Slug
		item.Type = resolved.Kind
		item.MemoriesPath = resolved.RootDir
		item.StatePath = resolved.StateDir
		projects = append(projects, item)
	}

	return ListResult{Projects: projects}, nil
}

// Show returns the registered project details for a selector.
func (s Service) Show(selector string) (ShowResult, error) {
	resolved, err := s.Resolve(selector)
	if err != nil {
		return ShowResult{}, err
	}

	return ShowResult{
		ProjectID: resolved.ID,
		Name:      resolved.Name,
		Slug:      resolved.Slug,
		Type:      resolved.Kind,
		StateHome: resolved.StateDir,
		Location: ShowLocationResult{
			MemoriesAbs: resolved.RootDir,
			ManifestAbs: resolved.ManifestPath,
			RepoRootAbs: resolved.RepoRootDir,
		},
	}, nil
}

// Import imports a project into the registry.
func (s Service) Import(input ImportInput) (ImportResult, error) {
	return project.ImportProject(input, s.MemoriesHome)
}

// Init creates a project, resolves it, and builds the initial index.
func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error) {
	slug, err := project.Slugify(input.Name)
	if err != nil {
		return InitResult{}, err
	}

	if err := project.InitProject(project.InitInput{
		CWD:          input.WorkingDir,
		MemoriesHome: s.MemoriesHome,
		Name:         input.Name,
		Description:  input.Description,
		Mode:         input.Mode,
	}); err != nil {
		return InitResult{}, wrapInitError(err)
	}

	resolved, err := s.Resolve(slug)
	if err != nil {
		return InitResult{}, err
	}

	result := InitResult{
		KnowledgeBase: resolved,
		IndexStatus:   "stale",
	}
	rebuild, err := indexsvc.New(resolved).Rebuild(ctx)
	if err != nil {
		result.IndexError = err.Error()
		return result, err
	}
	result.IndexStatus = rebuild.Status
	return result, nil
}

// Remove removes a project from the registry and state directories.
func (s Service) Remove(selector string, wipe bool) (RemoveResult, error) {
	resolved, err := s.Resolve(selector)
	if err != nil {
		return RemoveResult{}, err
	}

	registryRemoved := false
	registryPath := registryPathForKind(s.MemoriesHome, resolved.Kind, resolved.Slug)
	if registryPath != "" {
		if err := os.Remove(registryPath); err != nil && !os.IsNotExist(err) {
			return RemoveResult{}, fmt.Errorf("remove registry entry %q: %w", registryPath, err)
		}
		registryRemoved = true
	}

	indexDeleted := false
	if resolved.IndexPath != "" {
		for _, path := range []string{resolved.IndexPath, resolved.IndexPath + "-wal", resolved.IndexPath + "-shm"} {
			if _, err := os.Stat(path); err == nil {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return RemoveResult{}, fmt.Errorf("remove index artifact %q: %w", path, err)
				}
				indexDeleted = true
			}
		}
	}
	if resolved.StateDir != "" {
		if err := os.RemoveAll(resolved.StateDir); err != nil {
			return RemoveResult{}, fmt.Errorf("remove state directory %q: %w", resolved.StateDir, err)
		}
	}

	markdownDeleted := false
	if wipe && resolved.RootDir != "" {
		if err := os.RemoveAll(resolved.RootDir); err != nil {
			return RemoveResult{}, fmt.Errorf("wipe markdown: %w", err)
		}
		markdownDeleted = true
	}

	return RemoveResult{
		ProjectID:       resolved.ID,
		Slug:            resolved.Slug,
		RegistryRemoved: registryRemoved,
		IndexDeleted:    indexDeleted,
		MarkdownDeleted: markdownDeleted,
		FullWipe:        wipe,
	}, nil
}

// Slugs returns the active project slugs.
func (s Service) Slugs() ([]string, error) {
	return registry.Slugs(s.MemoriesHome)
}

func (s Service) knowledgeBaseFromEntry(entry registry.Entry) (kb.KnowledgeBase, error) {
	resolved := entry

	if strings.TrimSpace(resolved.Slug) == "" {
		return kb.KnowledgeBase{}, fmt.Errorf("project slug is required")
	}

	switch resolved.Type {
	case "central":
		if resolved.ManifestPath == "" {
			resolved.ManifestPath = filepath.Join(s.MemoriesHome, resolved.Slug, "mnemonic.toml")
		}
		if resolved.MemoriesAbs == "" {
			resolved.MemoriesAbs = filepath.Dir(resolved.ManifestPath)
		}
		if resolved.RepoRootAbs == "" {
			resolved.RepoRootAbs = resolved.MemoriesAbs
		}
	case "local":
		if resolved.ManifestPath == "" {
			pointerPath := filepath.Join(s.MemoriesHome, resolved.Slug+".toml")
			data, err := os.ReadFile(pointerPath)
			if err != nil {
				return kb.KnowledgeBase{}, fmt.Errorf("read pointer file: %w", err)
			}
			manifestPath, err := project.ParsePointerFile(data)
			if err != nil {
				return kb.KnowledgeBase{}, err
			}
			resolved.ManifestPath = manifestPath.ManifestPath
		}
		if resolved.ManifestPath != "" && resolved.MemoriesAbs == "" {
			resolved.MemoriesAbs = filepath.Dir(resolved.ManifestPath)
		}
		if resolved.ManifestPath != "" && resolved.RepoRootAbs == "" {
			resolved.RepoRootAbs = filepath.Dir(filepath.Dir(resolved.ManifestPath))
		}
	default:
		if resolved.ManifestPath == "" {
			resolved.ManifestPath = filepath.Join(s.MemoriesHome, resolved.Slug, "mnemonic.toml")
		}
		if resolved.MemoriesAbs == "" {
			resolved.MemoriesAbs = filepath.Dir(resolved.ManifestPath)
		}
		if resolved.RepoRootAbs == "" {
			resolved.RepoRootAbs = resolved.MemoriesAbs
		}
	}

	if resolved.ManifestPath != "" && (resolved.ProjectID == "" || resolved.Name == "" || resolved.Slug == "") {
		manifest, err := project.ParseMnemonicManifestFromFile(resolved.ManifestPath)
		if err != nil {
			return kb.KnowledgeBase{}, err
		}
		if resolved.ProjectID == "" {
			resolved.ProjectID = manifest.ProjectID
		}
		if resolved.Name == "" {
			resolved.Name = manifest.Name
		}
		if resolved.Slug == "" {
			resolved.Slug = manifest.Slug
		}
		if resolved.Type == "" {
			resolved.Type = string(manifest.Type)
		}
	}

	if strings.TrimSpace(resolved.ProjectID) == "" {
		return kb.KnowledgeBase{}, fmt.Errorf("project id is required")
	}

	stateDir := s.statePath(resolved.ProjectID)

	return kb.KnowledgeBase{
		ID:           resolved.ProjectID,
		Name:         resolved.Name,
		Slug:         resolved.Slug,
		Kind:         resolved.Type,
		RootDir:      resolved.MemoriesAbs,
		RepoRootDir:  resolved.RepoRootAbs,
		ManifestPath: resolved.ManifestPath,
		StateDir:     stateDir,
		IndexPath:    filepath.Join(stateDir, "index.sqlite"),
	}, nil
}

func (s Service) statePath(projectID string) string {
	return filepath.Join(s.StateHome, "mnemonic", "projects", projectID)
}

func registryPathForKind(memoriesHome, kind, slug string) string {
	switch kind {
	case "central":
		return filepath.Join(memoriesHome, slug, "mnemonic.toml")
	case "local":
		return filepath.Join(memoriesHome, slug+".toml")
	default:
		return ""
	}
}

func wrapRegistryError(err error) error {
	if err == nil {
		return nil
	}

	var notFoundErr registry.ErrNotFound
	if errors.As(err, &notFoundErr) {
		return apperr.NotFound(notFoundErr.Error(), nil)
	}

	return err
}

func wrapInitError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "project slug") && strings.Contains(err.Error(), "already exists") {
		return apperr.Ambiguous(err.Error(), nil)
	}

	return err
}
