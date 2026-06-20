package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitMode identifies the supported init flows.
type InitMode string

const (
	// InitModeCentral creates a central project with a mnemonic.toml under memories home.
	InitModeCentral InitMode = "central"
	// InitModeLocal creates a local project backed by .mnemonic-memories.
	InitModeLocal InitMode = "local"
)

// InitInput captures the inputs required to initialize a project.
type InitInput struct {
	CWD          string
	MemoriesHome string
	Name         string
	Description  string
	Mode         InitMode
}

// InitCentralInput is the compatibility wrapper for central init tests.
type InitCentralInput = InitInput

// InitCentralProject creates a central project and its mnemonic.toml manifest.
func InitCentralProject(input InitCentralInput) error {
	input.Mode = InitModeCentral
	return InitProject(input)
}

// InitProject creates the manifest (central) or local memories directory
// (local) using the file-based registry.
func InitProject(input InitInput) error {
	slug, err := Slugify(input.Name)
	if err != nil {
		return err
	}

	now := NowUTC()
	projectID := NewUUID()

	switch input.Mode {
	case InitModeCentral:
		manifestPath := filepath.Join(input.MemoriesHome, slug, "mnemonic.toml")
		if _, err := os.Stat(manifestPath); err == nil {
			return slugAlreadyExistsError(slug)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat central manifest: %w", err)
		}

		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := WriteMnemonicManifest(manifestPath, manifest); err != nil {
			return err
		}

		return nil

	case InitModeLocal:
		memoriesPath := filepath.Join(input.CWD, ".mnemonic-memories", slug)
		if err := os.MkdirAll(memoriesPath, 0o755); err != nil {
			return fmt.Errorf("create local memories directory: %w", err)
		}

		localManifestPath := filepath.Join(memoriesPath, "mnemonic.toml")
		manifest := NewMnemonicManifest()
		manifest.ProjectID = projectID
		manifest.Name = input.Name
		manifest.Slug = slug
		manifest.Type = ManifestTypeLocal
		manifest.MarkdownFormatVersion = 1
		manifest.Description = input.Description
		manifest.CreatedAt = now
		manifest.UpdatedAt = now
		manifest.Generator.App = "mnemonic"

		if err := WriteMnemonicManifest(localManifestPath, manifest); err != nil {
			return err
		}

		pointerPath := filepath.Join(input.MemoriesHome, slug+".toml")
		pointer := PointerFile{ManifestPath: localManifestPath}
		if err := WritePointerFile(pointerPath, &pointer); err != nil {
			return err
		}

		return nil

	default:
		return fmt.Errorf("unknown init mode %q", input.Mode)
	}
}

func slugAlreadyExistsError(slug string) error {
	return fmt.Errorf("project slug %q already exists", slug)
}
