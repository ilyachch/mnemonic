package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/paths"
)

// Path returns the absolute index.sqlite path for a project UUID.
func Path(projectID string) (string, error) {
	if strings.TrimSpace(projectID) == "" {
		return "", fmt.Errorf("project id is required")
	}
	if strings.ContainsAny(projectID, string(os.PathSeparator)) {
		return "", fmt.Errorf("project id must not contain path separators")
	}

	effective, err := paths.GetMnemonicPaths()
	if err != nil {
		return "", err
	}

	return filepath.Join(effective.StateHome, "mnemonic", "projects", projectID, "index.sqlite"), nil
}
