package project

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registry"
)

// EnvironmentProjectSelector is the environment variable used to provide an
// implicit project selector when the CLI flag is not set.
const EnvironmentProjectSelector = "MNEMONIC_PROJECT"

// ErrProjectNotFound is returned when the requested selector does not match
// any active project in the registry.
type ErrProjectNotFound struct {
	Selector string
}

// Error implements the error interface.
func (e ErrProjectNotFound) Error() string {
	return fmt.Sprintf("project %q not found", e.Selector)
}

// ResolveProject resolves a project by selector using the file-based registry.
// Returns the project ID that can be used for index and lock paths.
func ResolveProject(memoriesHome, selector string) (string, error) {
	entry, err := registry.Resolve(memoriesHome, selector)
	if err != nil {
		return "", err
	}
	return entry.ProjectID, nil
}

// MemoriesHome returns the effective memories home directory.
func MemoriesHome() (string, error) {
	mnemonicPaths, err := paths.GetMnemonicPaths()
	if err != nil {
		return "", err
	}
	return mnemonicPaths.MemoriesHome, nil
}
