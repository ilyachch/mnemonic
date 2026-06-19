package app

import (
	"database/sql"
	"time"
)

// Services groups the app-owned shared dependencies exposed to adapters.
type Services struct {
	ProjectResolver ProjectResolver
	Registry        *sql.DB
}

// ProjectResolveInput captures the inputs needed to choose a project.
type ProjectResolveInput struct {
	ProjectSelector  string
	EnvironmentValue string
}

// ProjectRecord describes a resolved project without exposing transport concerns.
type ProjectRecord struct {
	ID           string
	Name         string
	Slug         string
	Kind         string
	MemoriesPath string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProjectResolution is the result of a project lookup.
type ProjectResolution struct {
	MnemonicFilePath string
	RepoRootAbs      string
	MemoriesAbs      string
	ManifestAbs      string
	Project          ProjectRecord
}

// ProjectResolver resolves projects without exposing transport-specific concerns.
type ProjectResolver interface {
	Resolve(input ProjectResolveInput) (ProjectResolution, error)
}
