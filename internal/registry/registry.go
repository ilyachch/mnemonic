package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry describes a resolved project from the file-based registry.
type Entry struct {
	ProjectID    string
	Name         string
	Slug         string
	Type         string // "central" or "local"
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

// ManifestData holds the parsed fields needed from a manifest file.
type ManifestData struct {
	ProjectID string
	Name      string
	Slug      string
	Type      string // "local" or empty for central
}

// ManifestParser parses a mnemonic.toml file from disk.
type ManifestParser func(path string) (ManifestData, error)

// PointerParser parses a pointer file from raw data.
type PointerParser func(data []byte) (string, error) // returns manifest_path

// DefaultManifestParser parses a mnemonic.toml file using a basic TOML reader.
// Callers should replace this with their actual manifest parser.
var DefaultManifestParser ManifestParser
var DefaultPointerParser PointerParser

// Scan reads all projects from the memories home directory.
// Callers must set DefaultManifestParser and DefaultPointerParser before calling.
func Scan(memoriesHome string) ([]Entry, []Issue, error) {
	if memoriesHome == "" {
		return nil, nil, fmt.Errorf("memories home is required")
	}

	entries, err := os.ReadDir(memoriesHome)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read memories home %q: %w", memoriesHome, err)
	}

	var results []Entry
	var issues []Issue

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			// Central project: directory with mnemonic.toml
			slug := name
			manifestPath := filepath.Join(memoriesHome, slug, "mnemonic.toml")
			memoriesAbs := filepath.Join(memoriesHome, slug)

			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				continue
			}

			if DefaultManifestParser == nil {
				results = append(results, Entry{
					Slug:         slug,
					Type:         "central",
					ManifestPath: manifestPath,
					MemoriesAbs:  memoriesAbs,
					RepoRootAbs:  memoriesAbs,
				})
				continue
			}

			manifest, err := DefaultManifestParser(manifestPath)
			if err != nil {
				issues = append(issues, Issue{
					Slug:    slug,
					Path:    memoriesAbs,
					Error:   fmt.Sprintf("[CORRUPTED] %v", err),
					Corrupt: true,
				})
				results = append(results, Entry{
					Slug:         slug,
					Type:         "central",
					ManifestPath: manifestPath,
					MemoriesAbs:  memoriesAbs,
					RepoRootAbs:  memoriesAbs,
				})
				continue
			}

			if slug != manifest.Slug {
				issues = append(issues, Issue{
					Slug:    slug,
					Path:    memoriesAbs,
					Error:   fmt.Sprintf("Project name mismatch in registry (%s) and manifest (%s). Please align these values.", slug, manifest.Slug),
					Corrupt: true,
				})
			}

			results = append(results, Entry{
				ProjectID:    manifest.ProjectID,
				Name:         manifest.Name,
				Slug:         slug,
				Type:         "central",
				ManifestPath: manifestPath,
				MemoriesAbs:  memoriesAbs,
				RepoRootAbs:  memoriesAbs,
			})
			continue
		}

		if strings.HasSuffix(name, ".toml") {
			slug := strings.TrimSuffix(name, ".toml")
			pointerPath := filepath.Join(memoriesHome, name)

			if DefaultPointerParser == nil {
				results = append(results, Entry{
					Slug: slug,
					Type: "local",
				})
				continue
			}

			data, err := os.ReadFile(pointerPath)
			if err != nil {
				issues = append(issues, Issue{
					Slug:    slug,
					Path:    pointerPath,
					Error:   fmt.Sprintf("[CORRUPTED] cannot read pointer: %v", err),
					Corrupt: true,
				})
				continue
			}

			manifestPath, err := DefaultPointerParser(data)
			if err != nil {
				issues = append(issues, Issue{
					Slug:    slug,
					Path:    pointerPath,
					Error:   fmt.Sprintf("[CORRUPTED] invalid pointer: %v", err),
					Corrupt: true,
				})
				continue
			}

			if DefaultManifestParser == nil {
				results = append(results, Entry{
					Slug:         slug,
					Type:         "local",
					ManifestPath: manifestPath,
				})
				continue
			}

			manifest, err := DefaultManifestParser(manifestPath)
			if err != nil {
				issues = append(issues, Issue{
					Slug:   slug,
					Path:   pointerPath,
					Error:  fmt.Sprintf("[ORPHANED/MISSING] Local manifest not found at %s. The project might have been moved. Please update the path in %s or re-import the project.", manifestPath, pointerPath),
					Orphan: true,
				})
				results = append(results, Entry{
					Slug:         slug,
					Type:         "local",
					ManifestPath: manifestPath,
				})
				continue
			}

			if slug != manifest.Slug {
				issues = append(issues, Issue{
					Slug:    slug,
					Path:    pointerPath,
					Error:   fmt.Sprintf("Project name mismatch in registry (%s) and manifest (%s). Please align these values.", slug, manifest.Slug),
					Corrupt: true,
				})
			}

			memoriesAbs := filepath.Dir(manifestPath)
			repoRootAbs := filepath.Dir(filepath.Dir(manifestPath))
			results = append(results, Entry{
				ProjectID:    manifest.ProjectID,
				Name:         manifest.Name,
				Slug:         slug,
				Type:         "local",
				ManifestPath: manifestPath,
				MemoriesAbs:  memoriesAbs,
				RepoRootAbs:  repoRootAbs,
			})
		}
	}

	return results, issues, nil
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
func Resolve(memoriesHome, slug string) (Entry, error) {
	entries, issues, err := Scan(memoriesHome)
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
func FindByMemoriesRoot(memoriesHome, absRoot string) (Entry, bool, error) {
	entries, _, err := Scan(memoriesHome)
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
func Exists(memoriesHome, slug string) (bool, error) {
	dirPath := filepath.Join(memoriesHome, slug)
	pointerPath := filepath.Join(memoriesHome, slug+".toml")

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
func Slugs(memoriesHome string) ([]string, error) {
	entries, _, err := Scan(memoriesHome)
	if err != nil {
		return nil, err
	}

	slugs := make([]string, 0, len(entries))
	for _, e := range entries {
		slugs = append(slugs, e.Slug)
	}
	return slugs, nil
}
