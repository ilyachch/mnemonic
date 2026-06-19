package app

import (
	"database/sql"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/project"
)

// RegistryResolver implements ProjectResolver by delegating to the project
// package's registry-based lookup.
type RegistryResolver struct {
	DB *sql.DB
}

// Resolve looks up a project in the registry using the explicit selector or
// environment value.
func (r RegistryResolver) Resolve(input ProjectResolveInput) (ProjectResolution, error) {
	resolved, err := project.ResolveProject(project.ResolveProjectInput{
		ProjectSelector:  input.ProjectSelector,
		EnvironmentValue: input.EnvironmentValue,
		Registry:         r.DB,
	})
	if err != nil {
		switch err.(type) {
		case project.ErrNoProjectSelected, project.ErrProjectNotFound:
			return ProjectResolution{}, err
		default:
			return ProjectResolution{}, err
		}
	}

	return ProjectResolution{
		MnemonicFilePath: resolved.MnemonicFilePath,
		RepoRootAbs:      resolved.RepoRootAbs,
		MemoriesAbs:      resolved.MemoriesAbs,
		ManifestAbs:      resolved.ManifestAbs,
		Project: ProjectRecord{
			ID:           resolved.Project.ID,
			Name:         resolved.Project.Name,
			Slug:         resolved.Project.Slug,
			Kind:         string(resolved.Project.Kind),
			MemoriesPath: memoriesPathFor(resolved),
		},
	}, nil
}

func memoriesPathFor(resolved project.ResolvedProject) string {
	if resolved.RepoRootAbs != "" && resolved.MemoriesAbs != "" {
		rel, err := filepath.Rel(resolved.RepoRootAbs, resolved.MemoriesAbs)
		if err == nil && rel != "" && rel != "." {
			return filepath.ToSlash(rel)
		}
	}
	if resolved.MemoriesAbs != "" {
		return filepath.Base(resolved.MemoriesAbs)
	}
	return ""
}
