package registry

import (
	"database/sql"

	"github.com/ilyachch/mnemonic/internal/registryschema"
)

// ApplySchema creates or upgrades the registry schema to the current version.
func ApplySchema(db *sql.DB) error {
	return registryschema.Apply(db)
}
