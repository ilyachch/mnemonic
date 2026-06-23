package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	manifest "github.com/ilyachch/mnemonic/internal/format/manifest"
)

// Entry describes a resolved project from the file-based registry.
type Entry struct {
	ProjectID    string
	Name         string
	Slug         string
	Type         string // "central" or "local"
	Description  string
	ManifestPath string
	MemoriesAbs  string
	RepoRootAbs  string
}

// ErrNotFound is returned when a project slug is not found.
type ErrNotFound struct {
	Slug string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("project %q not found", e.Slug)
}

// ManifestParser parses a mnemonic.toml file from disk.
type ManifestParser func(path string) (*manifest.Manifest, error)

// PointerParser parses a pointer file from raw data.
type PointerParser func(data []byte) (*manifest.PointerFile, error)

// Store owns file-based registry access for one memories home.
type Store struct {
	MemoriesHome   string
	ManifestParser ManifestParser
	PointerParser  PointerParser
}

// New returns a registry store configured for the given memories home.
func New(memoriesHome string, manifestParser ManifestParser, pointerParser PointerParser) Store {
	return Store{
		MemoriesHome:   memoriesHome,
		ManifestParser: manifestParser,
		PointerParser:  pointerParser,
	}
}

// Scan reads all projects from the memories home directory.
func (s Store) Scan() ([]Entry, []Issue, error) {
	if s.MemoriesHome == "" {
		return nil, nil, errors.New("memories home is required")
	}
	entries, err := os.ReadDir(s.MemoriesHome)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read memories home %q: %w", s.MemoriesHome, err)
	}

	var results []Entry
	var issues []Issue
	for _, entry := range entries {
		if entry.IsDir() {
			s.scanDirectory(entry.Name(), &results, &issues)
		} else if strings.HasSuffix(entry.Name(), ".toml") {
			s.scanPointer(entry.Name(), &results, &issues)
		}
	}
	return results, issues, nil
}

func (s Store) scanDirectory(name string, results *[]Entry, issues *[]Issue) {
	slug := name
	manifestPath := filepath.Join(s.MemoriesHome, slug, "mnemonic.toml")
	memoriesAbs := filepath.Join(s.MemoriesHome, slug)

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return
	}
	if s.ManifestParser == nil {
		*results = append(*results, Entry{
			Slug: slug, Type: "central",
			ManifestPath: manifestPath, MemoriesAbs: memoriesAbs, RepoRootAbs: memoriesAbs,
		})
		return
	}
	manifestData, err := s.ManifestParser(manifestPath)
	if err != nil {
		*issues = append(*issues, Issue{Slug: slug, Path: memoriesAbs, Error: fmt.Sprintf("[CORRUPTED] %v", err), Corrupt: true})
		*results = append(*results, Entry{Slug: slug, Type: "central", ManifestPath: manifestPath, MemoriesAbs: memoriesAbs, RepoRootAbs: memoriesAbs})
		return
	}
	if slug != manifestData.Slug {
		*issues = append(*issues, Issue{Slug: slug, Path: memoriesAbs, Error: fmt.Sprintf("Project name mismatch in registry (%s) and manifest (%s). Please align these values.", slug, manifestData.Slug), Corrupt: true})
	}
	*results = append(*results, Entry{
		ProjectID: manifestData.ProjectID, Name: manifestData.Name, Slug: slug, Type: "central",
		Description: manifestData.Description, ManifestPath: manifestPath, MemoriesAbs: memoriesAbs, RepoRootAbs: memoriesAbs,
	})
}

func (s Store) scanPointer(name string, results *[]Entry, issues *[]Issue) {
	slug := strings.TrimSuffix(name, ".toml")
	pointerPath := filepath.Join(s.MemoriesHome, name)

	if s.PointerParser == nil {
		*results = append(*results, Entry{Slug: slug, Type: "local"})
		return
	}
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		*issues = append(*issues, Issue{Slug: slug, Path: pointerPath, Error: fmt.Sprintf("[CORRUPTED] cannot read pointer: %v", err), Corrupt: true})
		return
	}
	pointerFile, err := s.PointerParser(data)
	if err != nil {
		*issues = append(*issues, Issue{Slug: slug, Path: pointerPath, Error: fmt.Sprintf("[CORRUPTED] invalid pointer: %v", err), Corrupt: true})
		return
	}
	if s.ManifestParser == nil {
		*results = append(*results, Entry{Slug: slug, Type: "local", ManifestPath: pointerFile.ManifestPath})
		return
	}
	manifestData, err := s.ManifestParser(pointerFile.ManifestPath)
	if err != nil {
		*issues = append(*issues, Issue{Slug: slug, Path: pointerPath, Error: fmt.Sprintf("[ORPHANED/MISSING] Local manifest not found at %s. The project might have been moved. Please update the path in %s or re-import the project.", pointerFile.ManifestPath, pointerPath), Orphan: true})
		*results = append(*results, Entry{Slug: slug, Type: "local", ManifestPath: pointerFile.ManifestPath})
		return
	}
	if slug != manifestData.Slug {
		*issues = append(*issues, Issue{Slug: slug, Path: pointerPath, Error: fmt.Sprintf("Project name mismatch in registry (%s) and manifest (%s). Please align these values.", slug, manifestData.Slug), Corrupt: true})
	}
	memoriesAbs := filepath.Dir(pointerFile.ManifestPath)
	repoRootAbs := filepath.Dir(filepath.Dir(pointerFile.ManifestPath))
	*results = append(*results, Entry{
		ProjectID: manifestData.ProjectID, Name: manifestData.Name, Slug: slug, Type: "local",
		Description: manifestData.Description, ManifestPath: pointerFile.ManifestPath,
		MemoriesAbs: memoriesAbs, RepoRootAbs: repoRootAbs,
	})
}

// Issue reports a problem encountered during registry scanning.
type Issue struct {
	Slug    string `json:"slug"`
	Path    string `json:"path"`
	Error   string `json:"error"`
	Orphan  bool   `json:"orphan"`
	Corrupt bool   `json:"corrupt"`
}

// Resolve finds a single project by slug in the memories home.
func (s Store) Resolve(slug string) (Entry, error) {
	entries, issues, err := s.Scan()
	if err != nil {
		return Entry{}, err
	}

	for _, issue := range issues {
		if issue.Slug == slug && (issue.Corrupt || issue.Orphan) {
			return Entry{}, errors.New(issue.Error)
		}
	}

	for _, e := range entries {
		if e.Slug == slug {
			return e, nil
		}
	}

	return Entry{}, ErrNotFound{Slug: slug}
}

// FindByMemoriesRoot finds a project whose memories directory matches the given absolute path.
func (s Store) FindByMemoriesRoot(absRoot string) (Entry, bool, error) {
	entries, _, err := s.Scan()
	if err != nil {
		return Entry{}, false, err
	}

	for _, e := range entries {
		if filepath.Clean(e.MemoriesAbs) == filepath.Clean(absRoot) {
			return e, true, nil
		}
	}

	return Entry{}, false, nil
}

// Exists checks whether a slug already has a directory or pointer file.
func (s Store) Exists(slug string) (bool, error) {
	dirPath := filepath.Join(s.MemoriesHome, slug)
	pointerPath := filepath.Join(s.MemoriesHome, slug+".toml")

	for _, p := range []string{dirPath, pointerPath} {
		if _, err := os.Stat(p); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("stat %q: %w", p, err)
		}
	}

	return false, nil
}

// Slugs returns the list of active project slugs in the registry.
func (s Store) Slugs() ([]string, error) {
	entries, _, err := s.Scan()
	if err != nil {
		return nil, err
	}

	slugs := make([]string, 0, len(entries))
	for _, e := range entries {
		slugs = append(slugs, e.Slug)
	}
	return slugs, nil
}
