package catalogsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/domain/slug"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/platform/idgen"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
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
	ProjectID          string             `json:"project_id"`
	Name               string             `json:"name"`
	Slug               string             `json:"slug"`
	Type               string             `json:"type"`
	Description        string             `json:"description"`
	CustomInstructions string             `json:"custom_instructions"`
	LinksStyle         string             `json:"links_style"`
	StateHome          string             `json:"state_home"`
	Location           ShowLocationResult `json:"location"`
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
	Path            string
	Imported        int
	CopiedFiles     int
	Indexed         int
	IndexStatus     string
	IndexErrors     []ImportIndexError
	Candidates      []ImportCandidate
	DryRun          bool
	ManifestCreated bool
	Hydrated        []markdownstore.HydratedNote
	Skipped         []string
}

// AddInput mirrors the project add input.
type AddInput struct {
	Path string
}

// AddResult mirrors the project add result.
type AddResult struct {
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	ProjectID   string `json:"project_id"`
	Indexed     int    `json:"indexed"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

// ImportIndexError describes one imported project whose index rebuild failed.
type ImportIndexError struct {
	ProjectID string `json:"project_id"`
	Slug      string `json:"slug"`
	Error     string `json:"error"`
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
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Kind         string `json:"kind"`
	Description  string `json:"description,omitempty"`
	RootDir      string `json:"root_dir"`
	RepoRootDir  string `json:"repo_root_dir,omitempty"`
	ManifestPath string `json:"manifest_path,omitempty"`
	StateDir     string `json:"state_dir"`
	IndexPath    string `json:"index_path"`
	IndexStatus  string `json:"index_status"`
	IndexError   string `json:"index_error,omitempty"`
}

// Resolve returns the selected knowledge base for a project selector.
func (s Service) Resolve(selector string, logger *slog.Logger) (kb.KnowledgeBase, error) {
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

	if logger != nil {
		logger.Info("project selected",
			"slug", resolved.Slug,
			"kind", resolved.Kind,
			"root_dir", resolved.RootDir,
		)
	}

	return resolved, nil
}

// List returns the registered projects with state paths and issue status.
func (s Service) List(logger *slog.Logger) (ListResult, error) {
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

			if logger != nil {
				logger.Warn("project issue detected",
					"slug", entry.Slug,
					"status", item.Status,
					"error", issue.Error,
				)
			}
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
func (s Service) Show(selector string, logger *slog.Logger) (ShowResult, error) {
	resolved, err := s.Resolve(selector, logger)
	if err != nil {
		return ShowResult{}, err
	}

	return ShowResult{
		ProjectID:          resolved.ID,
		Name:               resolved.Name,
		Slug:               resolved.Slug,
		Type:               resolved.Kind,
		Description:        resolved.Description,
		CustomInstructions: resolved.CustomInstructions,
		LinksStyle:         resolved.LinksStyle,
		StateHome:          resolved.StateDir,
		Location: ShowLocationResult{
			MemoriesAbs: resolved.RootDir,
			ManifestAbs: resolved.ManifestPath,
			RepoRootAbs: resolved.RepoRootDir,
		},
	}, nil
}

// Import imports a raw directory of markdown notes into the mnemonic
// ecosystem. When mnemonic.toml is missing it is generated in-place; all
// notes lacking canonical frontmatter are hydrated; the project is
// registered via a pointer file and the search index is rebuilt.
func (s Service) Import(ctx context.Context, input ImportInput, logger *slog.Logger) (ImportResult, error) {
	resolvedPath, err := resolveImportPath(input)
	if err != nil {
		return ImportResult{}, err
	}

	manifestPath := filepath.Join(resolvedPath, "mnemonic.toml")
	manifest, manifestCreated, err := ensureManifest(manifestPath, resolvedPath, input.DryRun)
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
		Path:            resolvedPath,
		Imported:        1,
		CopiedFiles:     0,
		Indexed:         0,
		IndexErrors:     []ImportIndexError{},
		Candidates:      []ImportCandidate{candidate},
		DryRun:          input.DryRun,
		ManifestCreated: manifestCreated,
	}

	hydrateResult, err := markdownstore.Store{RootDir: resolvedPath, StateDir: s.statePath(manifest.ProjectID)}.Hydrate(markdownstore.HydrateInput{
		DryRun: input.DryRun,
	})
	if err != nil {
		return ImportResult{}, err
	}
	result.Hydrated = hydrateResult.Hydrated
	result.Skipped = hydrateResult.Skipped

	if logger != nil {
		logger.Info("import hydration complete",
			"slug", candidate.Slug,
			"hydrated", len(hydrateResult.Hydrated),
			"skipped", len(hydrateResult.Skipped),
		)
	}

	if input.DryRun {
		result.IndexStatus = "skipped"
		return result, nil
	}

	if err := s.registerPointer(memoriesHomePointerPath(s.MemoriesHome, manifest.Slug), manifestPath); err != nil {
		return ImportResult{}, err
	}

	return s.finalizeImportIndexStatus(ctx, result, logger), nil
}

// Add registers an existing structured project (one that already contains a
// valid mnemonic.toml) and rebuilds its search index.
func (s Service) Add(ctx context.Context, input AddInput, logger *slog.Logger) (AddResult, error) {
	resolvedPath, err := resolveAddPath(input)
	if err != nil {
		return AddResult{}, err
	}

	manifestPath := filepath.Join(resolvedPath, "mnemonic.toml")
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return AddResult{}, fmt.Errorf("mnemonic.toml not found at %s", resolvedPath)
		}
		return AddResult{}, fmt.Errorf("stat mnemonic.toml: %w", statErr)
	}

	manifest, err := manifestfmt.ParseMnemonicManifestFromFile(manifestPath)
	if err != nil {
		return AddResult{}, err
	}

	if registerErr := s.registerPointer(memoriesHomePointerPath(s.MemoriesHome, manifest.Slug), manifestPath); registerErr != nil {
		return AddResult{}, registerErr
	}

	result := AddResult{
		Path:        resolvedPath,
		Slug:        manifest.Slug,
		ProjectID:   manifest.ProjectID,
		IndexStatus: "skipped",
	}

	resolved, err := s.Resolve(manifest.Slug, logger)
	if err != nil {
		result.IndexError = err.Error()
		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
	}
	if _, err := indexsvc.New(resolved, logger).Rebuild(ctx); err != nil {
		result.IndexStatus = "stale"
		result.IndexError = err.Error()
		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
	}
	result.Indexed = 1
	result.IndexStatus = "ok"
	return result, nil
}

// ensureManifest parses the manifest at manifestPath, generating one in-place
// when it is missing. The boolean reports whether a new manifest was created.
// When dryRun is true a missing manifest is generated in memory only — it is
// not written to disk — so callers can preview the would-be project metadata.
func ensureManifest(manifestPath, resolvedPath string, dryRun bool) (*manifestfmt.Manifest, bool, error) {
	if _, statErr := os.Stat(manifestPath); statErr == nil {
		manifest, parseErr := manifestfmt.ParseMnemonicManifestFromFile(manifestPath)
		if parseErr != nil {
			return nil, false, parseErr
		}
		return manifest, false, nil
	} else if !os.IsNotExist(statErr) {
		return nil, false, fmt.Errorf("stat mnemonic.toml: %w", statErr)
	}

	projectID := idgen.NewUUID()
	name := filepath.Base(resolvedPath)
	slugValue, err := slug.Slugify(name)
	if err != nil {
		return nil, false, err
	}
	manifest := buildInitManifest(projectID, name, slugValue, string(manifestfmt.ManifestTypeLocal), "", clock.NowUTC())
	if !dryRun {
		if err := manifestfmt.WriteMnemonicManifest(manifestPath, manifest); err != nil {
			return nil, false, err
		}
	}
	return manifest, true, nil
}

// registerPointer writes the pointer file at pointerPath unless it already
// exists, returning an Ambiguous-style error on conflict.
func (s Service) registerPointer(pointerPath, manifestPath string) error {
	if _, err := os.Stat(pointerPath); err == nil {
		return fmt.Errorf("project slug %q already exists", strings.TrimSuffix(filepath.Base(pointerPath), ".toml"))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat pointer file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(pointerPath), 0o755); err != nil {
		return fmt.Errorf("create pointer directory: %w", err)
	}
	return manifestfmt.WritePointerFile(pointerPath, &manifestfmt.PointerFile{ManifestPath: manifestPath})
}

// memoriesHomePointerPath returns the pointer file path for a slug.
func memoriesHomePointerPath(memoriesHome, slugValue string) string {
	return filepath.Join(memoriesHome, slugValue+".toml")
}

func resolveAddPath(input AddInput) (string, error) {
	path := input.Path
	if path == "" {
		path = "."
	}
	absPath, err := paths.NormalizeAbsolutePath(path)
	if err != nil {
		return "", fmt.Errorf("resolve add path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("add path %q not found", path)
		}
		return "", fmt.Errorf("stat add path %s: %w", absPath, err)
	}
	return absPath, nil
}

func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportResult, logger *slog.Logger) ImportResult {
	if result.IndexErrors == nil {
		result.IndexErrors = []ImportIndexError{}
	}
	if result.DryRun {
		result.IndexStatus = "skipped"
		return result
	}

	for _, candidate := range result.Candidates {
		resolved, err := s.Resolve(candidate.Slug, logger)
		if err != nil {
			result.IndexErrors = append(result.IndexErrors, ImportIndexError{
				ProjectID: candidate.ProjectID,
				Slug:      candidate.Slug,
				Error:     err.Error(),
			})
			continue
		}
		if _, err := indexsvc.New(resolved, logger).Rebuild(ctx); err != nil {
			result.IndexErrors = append(result.IndexErrors, ImportIndexError{
				ProjectID: resolved.ID,
				Slug:      resolved.Slug,
				Error:     err.Error(),
			})
			continue
		}
		result.Indexed++
	}
	if len(result.IndexErrors) > 0 {
		result.IndexStatus = "stale"
		return result
	}
	result.IndexStatus = "ok"
	return result
}

// Init creates a project, resolves it, and builds the initial index.
func (s Service) Init(ctx context.Context, input InitInput, logger *slog.Logger) (InitResult, error) {
	slugValue, err := slug.Slugify(input.Name)
	if err != nil {
		return InitResult{}, err
	}

	if err = InitProject(InitProjectInput{
		CWD:          input.WorkingDir,
		MemoriesHome: s.MemoriesHome,
		Name:         input.Name,
		Description:  input.Description,
		Mode:         input.Mode,
	}); err != nil {
		return InitResult{}, wrapInitError(err)
	}

	resolved, err := s.Resolve(slugValue, logger)
	if err != nil {
		return InitResult{}, err
	}

	result := InitResult{
		ID:           resolved.ID,
		Name:         resolved.Name,
		Slug:         resolved.Slug,
		Kind:         resolved.Kind,
		Description:  resolved.Description,
		RootDir:      resolved.RootDir,
		RepoRootDir:  resolved.RepoRootDir,
		ManifestPath: resolved.ManifestPath,
		StateDir:     resolved.StateDir,
		IndexPath:    resolved.IndexPath,
		IndexStatus:  "stale",
	}
	_, err = indexsvc.New(resolved, logger).Rebuild(ctx)
	if err != nil {
		result.IndexError = err.Error()
		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
	}
	result.IndexStatus = "ok"
	return result, nil
}

// Remove removes a project from the registry and state directories.
func (s Service) Remove(selector string, wipe bool, logger *slog.Logger) (RemoveResult, error) {
	resolved, err := s.Resolve(selector, logger)
	if err != nil {
		return RemoveResult{}, err
	}
	registryRemoved, err := s.removeRegistryEntry(registryPathForKind(s.MemoriesHome, resolved.Kind, resolved.Slug))
	if err != nil {
		return RemoveResult{}, err
	}
	indexDeleted, err := s.removeIndexArtifacts(resolved.IndexPath, resolved.StateDir)
	if err != nil {
		return RemoveResult{}, err
	}
	markdownDeleted, err := s.wipeMarkdown(wipe, resolved.RootDir)
	if err != nil {
		return RemoveResult{}, err
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

func (s Service) removeRegistryEntry(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("remove registry entry %q: %w", path, err)
	}
	return true, nil
}

func (s Service) removeIndexArtifacts(indexPath, stateDir string) (bool, error) {
	deleted := false
	if indexPath != "" {
		for _, path := range []string{indexPath, indexPath + "-wal", indexPath + "-shm"} {
			if _, err := os.Stat(path); err == nil {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return false, fmt.Errorf("remove index artifact %q: %w", path, err)
				}
				deleted = true
			}
		}
	}
	if stateDir != "" {
		if err := os.RemoveAll(stateDir); err != nil {
			return false, fmt.Errorf("remove state directory %q: %w", stateDir, err)
		}
	}
	return deleted, nil
}

func (s Service) wipeMarkdown(wipe bool, rootDir string) (bool, error) {
	if !wipe || rootDir == "" {
		return false, nil
	}
	if err := os.RemoveAll(rootDir); err != nil {
		return false, fmt.Errorf("wipe markdown: %w", err)
	}
	return true, nil
}

// Slugs returns the active project slugs.
func (s Service) Slugs() ([]string, error) {
	return s.registryStore().Slugs()
}

func (s Service) knowledgeBaseFromEntry(entry registry.Entry) (kb.KnowledgeBase, error) {
	resolved := entry
	if strings.TrimSpace(resolved.Slug) == "" {
		return kb.KnowledgeBase{}, errors.New("project slug is required")
	}
	if err := s.resolvePathsForKind(&resolved); err != nil {
		return kb.KnowledgeBase{}, err
	}
	s.enrichFromManifest(&resolved)
	if strings.TrimSpace(resolved.ProjectID) == "" {
		return kb.KnowledgeBase{}, errors.New("project id is required")
	}
	stateDir := s.statePath(resolved.ProjectID)
	return kb.KnowledgeBase{
		ID:                 resolved.ProjectID,
		Name:               resolved.Name,
		Slug:               resolved.Slug,
		Kind:               resolved.Type,
		Description:        resolved.Description,
		CustomInstructions: resolved.CustomInstructions,
		LinksStyle:         resolved.LinksStyle,
		RootDir:            resolved.MemoriesAbs,
		RepoRootDir:        resolved.RepoRootAbs,
		ManifestPath:       resolved.ManifestPath,
		StateDir:           stateDir,
		IndexPath:          filepath.Join(stateDir, "index.sqlite"),
	}, nil
}

func (s Service) resolvePathsForKind(resolved *registry.Entry) error {
	switch resolved.Type {
	case "local":
		return s.resolveLocalPaths(resolved)
	default:
		s.resolveCentralDefaults(resolved)
		return nil
	}
}

func (s Service) resolveCentralDefaults(resolved *registry.Entry) {
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

func (s Service) resolveLocalPaths(resolved *registry.Entry) error {
	if resolved.ManifestPath == "" {
		mp, err := s.resolveLocalManifest(resolved.Slug)
		if err != nil {
			return err
		}
		resolved.ManifestPath = mp
	}
	if resolved.ManifestPath != "" && resolved.MemoriesAbs == "" {
		resolved.MemoriesAbs = filepath.Dir(resolved.ManifestPath)
	}
	if resolved.ManifestPath != "" && resolved.RepoRootAbs == "" {
		resolved.RepoRootAbs = filepath.Dir(filepath.Dir(resolved.ManifestPath))
	}
	return nil
}

func (s Service) resolveLocalManifest(slug string) (string, error) {
	pointerPath := filepath.Join(s.MemoriesHome, slug+".toml")
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		return "", fmt.Errorf("read pointer file: %w", err)
	}
	manifestPath, err := manifestfmt.ParsePointerFile(data)
	if err != nil {
		return "", err
	}
	return manifestPath.ManifestPath, nil
}

func (s Service) enrichFromManifest(resolved *registry.Entry) {
	if resolved.ManifestPath == "" {
		return
	}
	needsManifest := resolved.ProjectID == "" || resolved.Name == "" || resolved.Slug == "" || resolved.Type == ""
	if needsManifest {
		s.fillEntryFromManifest(resolved)
	} else {
		s.fillMetadataOnly(resolved)
	}
}

func (s Service) fillEntryFromManifest(resolved *registry.Entry) {
	manifest, err := manifestfmt.ParseMnemonicManifestFromFile(resolved.ManifestPath)
	if err != nil {
		return
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
	if strings.TrimSpace(resolved.Description) == "" {
		resolved.Description = manifest.Description
	}
	if strings.TrimSpace(resolved.CustomInstructions) == "" {
		resolved.CustomInstructions = manifest.CustomInstructions
	}
	resolved.LinksStyle = manifest.Format.LinksStyle
}

func (s Service) fillMetadataOnly(resolved *registry.Entry) {
	if strings.TrimSpace(resolved.Description) != "" && strings.TrimSpace(resolved.CustomInstructions) != "" && resolved.LinksStyle != "" {
		return
	}
	manifest, err := manifestfmt.ParseMnemonicManifestFromFile(resolved.ManifestPath)
	if err != nil {
		return
	}
	if strings.TrimSpace(resolved.Description) == "" {
		resolved.Description = manifest.Description
	}
	if strings.TrimSpace(resolved.CustomInstructions) == "" {
		resolved.CustomInstructions = manifest.CustomInstructions
	}
	if resolved.LinksStyle == "" {
		resolved.LinksStyle = manifest.Format.LinksStyle
		if resolved.LinksStyle == "" {
			resolved.LinksStyle = "wiki"
		}
	}
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
		return initCentralProject(input.MemoriesHome, input.Name, slugValue, input.Description, projectID, now)
	case InitModeLocal:
		return initLocalProject(input.MemoriesHome, input.CWD, input.Name, slugValue, input.Description, projectID, now)
	default:
		return fmt.Errorf("unknown init mode %q", input.Mode)
	}
}

func initCentralProject(memoriesHome, name, slugValue, description, projectID string, now time.Time) error {
	reg := registry.New(memoriesHome, nil, nil)
	exists, err := reg.Exists(slugValue)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("project slug %q already exists", slugValue)
	}
	if err := os.MkdirAll(filepath.Join(memoriesHome, slugValue), 0o755); err != nil {
		return fmt.Errorf("create central memories directory: %w", err)
	}
	manifestPath := filepath.Join(memoriesHome, slugValue, "mnemonic.toml")
	if _, err := os.Stat(manifestPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat central manifest: %w", err)
	}
	manifest := buildInitManifest(projectID, name, slugValue, "", description, now)
	return manifestfmt.WriteMnemonicManifest(manifestPath, manifest)
}

func initLocalProject(memoriesHome, cwd, name, slugValue, description, projectID string, now time.Time) error {
	reg := registry.New(memoriesHome, nil, nil)
	exists, err := reg.Exists(slugValue)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("project slug %q already exists", slugValue)
	}
	memoriesPath := filepath.Join(cwd, ".mnemonic-memories", slugValue)
	if err := os.MkdirAll(memoriesPath, 0o755); err != nil {
		return fmt.Errorf("create local memories directory: %w", err)
	}
	localManifestPath := filepath.Join(memoriesPath, "mnemonic.toml")
	manifest := buildInitManifest(projectID, name, slugValue, string(manifestfmt.ManifestTypeLocal), description, now)
	if err := manifestfmt.WriteMnemonicManifest(localManifestPath, manifest); err != nil {
		return err
	}
	pointerPath := filepath.Join(memoriesHome, slugValue+".toml")
	return manifestfmt.WritePointerFile(pointerPath, &manifestfmt.PointerFile{ManifestPath: localManifestPath})
}

func buildInitManifest(projectID, name, slugValue, kind, description string, now time.Time) *manifestfmt.Manifest {
	m := manifestfmt.New()
	m.ProjectID = projectID
	m.Name = name
	m.Slug = slugValue
	m.MarkdownFormatVersion = 1
	m.Description = description
	m.CustomInstructions = ""
	m.CreatedAt = now.Unix()
	m.UpdatedAt = now.Unix()
	m.Generator.App = "mnemonic"
	if kind != "" {
		m.Type = manifestfmt.ManifestType(kind)
	}
	return m
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

func kindFromManifest(manifest *manifestfmt.Manifest) string {
	if manifest != nil && manifest.IsLocal() {
		return "local"
	}
	return "central"
}
