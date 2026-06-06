package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/registry"
)

// DiscoverInput configures discovery under a memories home directory.
type DiscoverInput struct {
	MemoriesHome string
	DryRun       bool
}

// DiscoverResult reports the outcome of a discovery run.
type DiscoverResult struct {
	MemoriesHome string
	Discovered   int
	Errors       []DiscoverIssue
}

// DiscoverIssue reports a manifest that could not be discovered.
type DiscoverIssue struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// DiscoverProjects scans the direct children of MemoriesHome for mnemonic.toml files
// and registers valid regular/detached projects in the registry.
func DiscoverProjects(input DiscoverInput) (DiscoverResult, error) {
	if input.MemoriesHome == "" {
		return DiscoverResult{}, fmt.Errorf("memories home is required")
	}

	memoriesHome, err := filepath.Abs(input.MemoriesHome)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("resolve memories home: %w", err)
	}

	entries, err := os.ReadDir(memoriesHome)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("read memories home: %w", err)
	}

	if input.DryRun {
		return discoverProjectsDryRun(memoriesHome, entries)
	}

	db, err := registry.OpenDB()
	if err != nil {
		return DiscoverResult{}, err
	}
	defer func() {
		_ = db.Close()
	}()

	if err := registry.ApplySchema(db); err != nil {
		return DiscoverResult{}, err
	}

	result := DiscoverResult{
		MemoriesHome: memoriesHome,
	}
	seenAt := NowUTC()
	for i := range entries {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}

		projectRoot := filepath.Join(memoriesHome, entry.Name())
		manifestPath := filepath.Join(projectRoot, "mnemonic.toml")

		if info, err := os.Stat(manifestPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return DiscoverResult{}, fmt.Errorf("stat mnemonic.toml: %w", err)
		} else if info.IsDir() {
			continue
		}

		manifest, err := loadMnemonicManifest(manifestPath)
		if err != nil {
			return DiscoverResult{}, err
		}
		if manifest.Kind != ManifestKindRegular && manifest.Kind != ManifestKindDetached {
			continue
		}

		if err := registry.RegisterProject(db, registry.RegisterProjectInput{
			ProjectID: manifest.ProjectID,
			Name:      manifest.Name,
			Slug:      manifest.Slug,
			Kind:      registry.ProjectKind(manifest.Kind),
			CreatedAt: manifest.CreatedAt,
			UpdatedAt: manifest.UpdatedAt,
			SeenAt:    seenAt,
			Location: registry.ProjectLocationInput{
				RepoRootAbs: projectRoot,
				MemoriesAbs: projectRoot,
				ManifestAbs: manifestPath,
				SourceKind:  registry.ProjectSourceKindDiscover,
			},
		}); err != nil {
			return DiscoverResult{}, err
		}

		result.Discovered++
	}

	return result, nil
}

func discoverProjectsDryRun(memoriesHome string, entries []os.DirEntry) (DiscoverResult, error) {
	result := DiscoverResult{MemoriesHome: memoriesHome}
	for i := range entries {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}

		projectRoot := filepath.Join(memoriesHome, entry.Name())
		manifestPath := filepath.Join(projectRoot, "mnemonic.toml")

		info, err := os.Stat(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			result.Errors = append(result.Errors, DiscoverIssue{
				Path:  manifestPath,
				Error: fmt.Sprintf("stat mnemonic.toml: %v", err),
			})
			continue
		}
		if info.IsDir() {
			continue
		}

		manifest, err := loadMnemonicManifest(manifestPath)
		if err != nil {
			result.Errors = append(result.Errors, DiscoverIssue{
				Path:  manifestPath,
				Error: err.Error(),
			})
			continue
		}
		if manifest.Kind != ManifestKindRegular && manifest.Kind != ManifestKindDetached {
			result.Errors = append(result.Errors, DiscoverIssue{
				Path:  manifestPath,
				Error: fmt.Sprintf("unsupported kind %q", manifest.Kind),
			})
			continue
		}

		result.Discovered++
	}

	return result, nil
}

func loadMnemonicManifest(path string) (*MnemonicManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mnemonic.toml: %w", err)
	}

	manifest, err := ParseMnemonicManifest(data)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}
