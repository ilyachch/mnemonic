package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
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
	Mode         InitMode
}

// InitRegularInput is the compatibility wrapper for regular init tests.
type InitRegularInput = InitInput

// InitRegularProject creates a regular .mnemonic project and its mnemonic.toml manifest.
func InitRegularProject(input InitRegularInput) error {
	input.Mode = InitModeRegular
	return InitProject(input)
}

// InitProject creates the project file and any mode-specific filesystem layout.
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

		file, err := loadMnemonicFile(filepath.Join(input.CWD, ".mnemonic"))
		if err != nil {
			return err
		}
		if projectSlugExists(file, slug) {
			return app.NewAmbiguousError(fmt.Sprintf("project slug %q already exists", slug), nil)
		}
		file.Projects = append(file.Projects, projectEntry)
		file.UpdatedAt = now

		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.Kind = ManifestKindRegular
		manifest.MarkdownFormatVersion = 1
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := WriteMnemonicManifest(filepath.Join(input.MemoriesHome, slug, "mnemonic.toml"), manifest); err != nil {
			return err
		}

		if err := WriteMnemonicFile(filepath.Join(input.CWD, ".mnemonic"), file); err != nil {
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
				MnemonicFileAbs: filepath.Join(input.CWD, ".mnemonic"),
				RepoRootAbs:     input.CWD,
				MemoriesAbs:     filepath.Join(input.MemoriesHome, slug),
				ManifestAbs:     filepath.Join(input.MemoriesHome, slug, "mnemonic.toml"),
				SourceKind:      registry.ProjectSourceKindInit,
			},
		})
	case InitModeLocal:
		projectEntry.Kind = ProjectKindLocal
		projectEntry.MemoriesPath = filepath.Join(".mnemonic-memories", slug)

		file, err := loadMnemonicFile(filepath.Join(input.CWD, ".mnemonic"))
		if err != nil {
			return err
		}
		if projectSlugExists(file, slug) {
			return app.NewAmbiguousError(fmt.Sprintf("project slug %q already exists", slug), nil)
		}

		if err := os.MkdirAll(filepath.Join(input.CWD, projectEntry.MemoriesPath), 0o755); err != nil {
			return fmt.Errorf("create local memories directory: %w", err)
		}

		file.Projects = append(file.Projects, projectEntry)
		file.UpdatedAt = now

		if err := WriteMnemonicFile(filepath.Join(input.CWD, ".mnemonic"), file); err != nil {
			return err
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
				MnemonicFileAbs: filepath.Join(input.CWD, ".mnemonic"),
				RepoRootAbs:     input.CWD,
				MemoriesAbs:     filepath.Join(input.CWD, projectEntry.MemoriesPath),
				SourceKind:      registry.ProjectSourceKindInit,
			},
		})
	case InitModeDetached:
		manifestPath := filepath.Join(input.MemoriesHome, slug, "mnemonic.toml")
		if _, err := os.Stat(manifestPath); err == nil {
			return app.NewAmbiguousError(fmt.Sprintf("project slug %q already exists", slug), nil)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat detached manifest: %w", err)
		}

		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.Kind = ManifestKindDetached
		manifest.MarkdownFormatVersion = 1
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

func loadMnemonicFile(path string) (*MnemonicFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewMnemonicFile(), nil
		}
		return nil, fmt.Errorf("read .mnemonic: %w", err)
	}

	file, err := ParseMnemonicFile(data)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func projectSlugExists(file *MnemonicFile, slug string) bool {
	for i := range file.Projects {
		if file.Projects[i].Slug == slug {
			return true
		}
	}
	return false
}
