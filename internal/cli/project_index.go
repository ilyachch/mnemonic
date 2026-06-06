package cli

import (
	"database/sql"

	"github.com/ilyachch/mnemonic/internal/index"
)

func buildProjectIndex(db *sql.DB, projectID, memoriesRoot string) error {
	if _, err := index.RebuildProjectIndex(projectID, memoriesRoot); err != nil {
		return err
	}
	return updateIndexStatus(db, projectID, true, false)
}
