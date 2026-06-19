package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/registry"
)

// InitMode identifies the supported init flows.
type InitMode string

const (
	// InitModeRegular creates a regular project with a mnemonic.toml manifest.
	InitModeRegular InitMode = "regular"
	// InitModeLocal creates a local project backed by .mnemonic-memories.
	InitModeLocal InitMode = "local"
	// InitModeDetached is reserved for the detached flow introduced later.
	InitModeDetached InitMode = "detached"
)

// InitInput captures the inputs required to initialize a project.
type InitInput struct {
	CWD          string
	MemoriesHome string
	Name         string
	Description  string
	Mode         InitMode
}

// InitRegularInput is the compatibility wrapper for regular init tests.
type InitRegularInput = InitInput

// InitRegularProject creates a regular project and its mnemonic.toml manifest.
func InitRegularProject(input InitRegularInput) error {
	input.Mode = InitModeRegular
	return InitProject(input)
}

// InitProject creates the manifest (regular/detached) or local memories directory
// (local) and registers the project directly in the global registry database.
func InitProject(input InitInput) error {
	slug, err := Slugify(input.Name)
	if err != nil {
		return err
	}

	now := NowUTC()
	projectID := NewUUID()

	projectEntry := MnemonicProject{
		ID:                    projectID,
		Name:                  input.Name,
		Slug:                  slug,
		MarkdownFormatVersion: 1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	switch input.Mode {
	case InitModeRegular:
		projectEntry.Kind = ProjectKindRegular
		projectEntry.MemoriesPath = slug

		manifestPath := filepath.Join(input.MemoriesHome, slug, "mnemonic.toml")
		if _, err := os.Stat(manifestPath); err == nil {
			return slugAlreadyExistsError(slug)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat regular manifest: %w", err)
		}

		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.Kind = ManifestKindRegular
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := WriteMnemonicManifest(manifestPath, manifest); err != nil {
			return err
		}

		return registerInitProject(registry.RegisterProjectInput{
			ProjectID: projectID,
			Name:      input.Name,
			Slug:      slug,
			Kind:      registry.ProjectKindRegular,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				RepoRootAbs: filepath.Join(input.MemoriesHome, slug),
				MemoriesAbs: filepath.Join(input.MemoriesHome, slug),
				ManifestAbs: manifestPath,
				SourceKind:  registry.ProjectSourceKindInit,
			},
		})
	case InitModeLocal:
		projectEntry.Kind = ProjectKindLocal
		projectEntry.MemoriesPath = filepath.Join(".mnemonic-memories", slug)

		if err := os.MkdirAll(filepath.Join(input.CWD, projectEntry.MemoriesPath), 0o755); err != nil {
			return fmt.Errorf("create local memories directory: %w", err)
		}

		return registerInitProject(registry.RegisterProjectInput{
			ProjectID: projectID,
			Name:      input.Name,
			Slug:      slug,
			Kind:      registry.ProjectKindLocal,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				RepoRootAbs: input.CWD,
				MemoriesAbs: filepath.Join(input.CWD, projectEntry.MemoriesPath),
				SourceKind:  registry.ProjectSourceKindInit,
			},
		})
	case InitModeDetached:
		manifestPath := filepath.Join(input.MemoriesHome, slug, "mnemonic.toml")
		if _, err := os.Stat(manifestPath); err == nil {
			return slugAlreadyExistsError(slug)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat detached manifest: %w", err)
		}

		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.Kind = ManifestKindDetached
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := WriteMnemonicManifest(manifestPath, manifest); err != nil {
			return err
		}

		return registerInitProject(registry.RegisterProjectInput{
			ProjectID: projectID,
			Name:      input.Name,
			Slug:      slug,
			Kind:      registry.ProjectKindDetached,
			CreatedAt: now,
			UpdatedAt: now,
			SeenAt:    now,
			Location: registry.ProjectLocationInput{
				MemoriesAbs: filepath.Join(input.MemoriesHome, slug),
				ManifestAbs: manifestPath,
				SourceKind:  registry.ProjectSourceKindInit,
			},
		})
	default:
		return fmt.Errorf("unknown init mode %q", input.Mode)
	}
}

func registerInitProject(input registry.RegisterProjectInput) error {
	db, err := registry.OpenDB()
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	return registry.RegisterProject(db, input)
}

func slugAlreadyExistsError(slug string) error {
	return fmt.Errorf("project slug %q already exists", slug)
}
