package cli

import (
	"github.com/ilyachch/mnemonic/internal/index"
)

func buildProjectIndex(projectID, memoriesRoot string) error {
	_, err := index.RebuildProjectIndex(projectID, memoriesRoot)
	return err
}
