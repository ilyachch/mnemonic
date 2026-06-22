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
	"github.com/ilyachch/mnemonic/internal/domain/slug"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/platform/idgen"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	registry "github.com/ilyachch/mnemonic/internal/store/registry"
)

// InitMode identifies the supported init flows.
type InitMode string

const (
	// InitModeCentral creates a central project with a mnemonic.toml under memories home.
	InitModeCentral InitMode = "central"
	// InitModeLocal creates a local project backed by .mnemonic-memories.
	InitModeLocal InitMode = "local"
)

// Service owns catalog-level project operations.
type Service struct {
	MemoriesHome string
	StateHome    string
	Registry     registry.Store
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
type ImportInput struct {
	Path   string
	DryRun bool
}

// ImportResult mirrors the project import result.
type ImportResult struct {
	Path        string
	Imported    int
	CopiedFiles int
	Indexed     int
	Candidates  []ImportCandidate
	DryRun      bool
}

// ImportCandidate mirrors the project import candidate.
type ImportCandidate struct {
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Kind            string `json:"kind"`
	MemoriesPath    string `json:"memories_path"`
	MnemonicFileAbs string `json:"mnemonic_file_abs,omitempty"`
	RepoRootAbs     string `json:"repo_root_abs"`
	ManifestAbs     string `json:"manifest_abs"`
}

// InitInput mirrors the project init input.
type InitInput struct {
	WorkingDir  string
	Name        string
	Description string
	Mode        InitMode
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

	entry, err := s.registryStore().Resolve(selector)
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
	entries, issues, err := s.registryStore().Scan()
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
	result, err := importProject(input, s.MemoriesHome)
	if err != nil {
		return ImportResult{}, err
	}
	if result.DryRun {
		return result, nil
	}

	for _, candidate := range result.Candidates {
		resolved, err := s.Resolve(candidate.Slug)
		if err != nil {
			return ImportResult{}, err
		}
		if _, err := indexsvc.New(resolved).Rebuild(context.Background()); err != nil {
			return ImportResult{}, err
		}
		result.Indexed++
	}

	return result, nil
}

// Init creates a project, resolves it, and builds the initial index.
func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error) {
	slugValue, err := slug.Slugify(input.Name)
	if err != nil {
		return InitResult{}, err
	}

	if err := InitProject(InitProjectInput{
		CWD:          input.WorkingDir,
		MemoriesHome: s.MemoriesHome,
		Name:         input.Name,
		Description:  input.Description,
		Mode:         input.Mode,
	}); err != nil {
		return InitResult{}, wrapInitError(err)
	}

	resolved, err := s.Resolve(slugValue)
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
	return s.registryStore().Slugs()
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
			manifestPath, err := registry.ParsePointerFile(data)
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
		manifest, err := registry.ParseMnemonicManifestFromFile(resolved.ManifestPath)
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

func (s Service) registryStore() registry.Store {
	store := s.Registry
	if store.MemoriesHome == "" {
		store.MemoriesHome = s.MemoriesHome
	}
	return store
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

// InitProjectInput configures the direct project init helper.
type InitProjectInput struct {
	CWD          string
	MemoriesHome string
	Name         string
	Description  string
	Mode         InitMode
}

// InitProject creates a project on disk for the requested init mode.
func InitProject(input InitProjectInput) error {
	slugValue, err := slug.Slugify(input.Name)
	if err != nil {
		return err
	}

	now := clock.NowUTC()
	projectID := idgen.NewUUID()

	switch input.Mode {
	case InitModeCentral:
		exists, err := registry.Exists(input.MemoriesHome, slugValue)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("project slug %q already exists", slugValue)
		}

		if err := os.MkdirAll(filepath.Join(input.MemoriesHome, slugValue), 0o755); err != nil {
			return fmt.Errorf("create central memories directory: %w", err)
		}

		manifestPath := filepath.Join(input.MemoriesHome, slugValue, "mnemonic.toml")
		if _, err := os.Stat(manifestPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("stat central manifest: %w", err)
		}

		manifest := registry.NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slugValue
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		return registry.WriteMnemonicManifest(manifestPath, manifest)
	case InitModeLocal:
		exists, err := registry.Exists(input.MemoriesHome, slugValue)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("project slug %q already exists", slugValue)
		}

		memoriesPath := filepath.Join(input.CWD, ".mnemonic-memories", slugValue)
		if err := os.MkdirAll(memoriesPath, 0o755); err != nil {
			return fmt.Errorf("create local memories directory: %w", err)
		}

		localManifestPath := filepath.Join(memoriesPath, "mnemonic.toml")
		manifest := registry.NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slugValue
		manifest.Type = registry.ManifestTypeLocal
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := registry.WriteMnemonicManifest(localManifestPath, manifest); err != nil {
			return err
		}

		pointerPath := filepath.Join(input.MemoriesHome, slugValue+".toml")
		return registry.WritePointerFile(pointerPath, &registry.PointerFile{ManifestPath: localManifestPath})
	default:
		return fmt.Errorf("unknown init mode %q", input.Mode)
	}
}

func importProject(input ImportInput, memoriesHome string) (ImportResult, error) {
	resolvedPath, err := resolveImportPath(input)
	if err != nil {
		return ImportResult{}, err
	}

	manifestPath := filepath.Join(resolvedPath, "mnemonic.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return ImportResult{}, fmt.Errorf("mnemonic.toml not found at %s", resolvedPath)
		}
		return ImportResult{}, fmt.Errorf("stat mnemonic.toml: %w", err)
	}

	manifest, err := registry.ParseMnemonicManifestFile(manifestPath)
	if err != nil {
		return ImportResult{}, err
	}

	candidate := ImportCandidate{
		ProjectID:       manifest.ProjectID,
		Name:            manifest.Name,
		Slug:            manifest.Slug,
		Kind:            kindFromManifest(manifest),
		MemoriesPath:    resolvedPath,
		MnemonicFileAbs: manifestPath,
		RepoRootAbs:     resolvedPath,
		ManifestAbs:     manifestPath,
	}

	result := ImportResult{
		Path:        resolvedPath,
		Imported:    1,
		CopiedFiles: 0,
		Indexed:     0,
		Candidates:  []ImportCandidate{candidate},
		DryRun:      input.DryRun,
	}
	if input.DryRun {
		return result, nil
	}

	pointerPath := filepath.Join(memoriesHome, manifest.Slug+".toml")
	if _, err := os.Stat(pointerPath); err == nil {
		return ImportResult{}, fmt.Errorf("project slug %q already exists", manifest.Slug)
	} else if !os.IsNotExist(err) {
		return ImportResult{}, fmt.Errorf("stat pointer file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(pointerPath), 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("create pointer directory: %w", err)
	}

	if err := registry.WritePointerFile(pointerPath, &registry.PointerFile{ManifestPath: manifestPath}); err != nil {
		return ImportResult{}, err
	}

	return result, nil
}

func resolveImportPath(input ImportInput) (string, error) {
	path := input.Path
	if path == "" {
		path = "."
	}

	absPath, err := paths.NormalizeAbsolutePath(path)
	if err != nil {
		return "", fmt.Errorf("resolve import path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("import path %q not found", path)
		}
		return "", fmt.Errorf("stat import path %s: %w", absPath, err)
	}
	return absPath, nil
}

func kindFromManifest(manifest *registry.MnemonicManifest) string {
	if manifest != nil && manifest.IsLocal() {
		return "local"
	}
	return "central"
}
