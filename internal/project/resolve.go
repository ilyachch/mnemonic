package project

import (
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
)

// ResolveProjectInput configures project resolution precedence.
type ResolveProjectInput struct {
	CWD              string
	ProjectSelector  string
	EnvironmentValue string
}

// ResolvedProject is the selected project together with the source file path.
type ResolvedProject struct {
	MnemonicFilePath string
	Project          MnemonicProject
}

// ResolveProject chooses a project using explicit selector, env selector, or nearest .mnemonic.
func ResolveProject(input ResolveProjectInput) (ResolvedProject, error) {
	mnemonicPath, err := FindNearestMnemonicFile(input.CWD)
	if err != nil {
		return ResolvedProject{}, err
	}

	file, err := loadMnemonicFile(mnemonicPath)
	if err != nil {
		return ResolvedProject{}, err
	}

	selector := input.ProjectSelector
	if selector == "" {
		selector = input.EnvironmentValue
	}

	project, err := selectMnemonicProject(file, selector)
	if err != nil {
		return ResolvedProject{}, err
	}

	return ResolvedProject{
		MnemonicFilePath: filepath.Clean(mnemonicPath),
		Project:          project,
	}, nil
}

func selectMnemonicProject(file *MnemonicFile, selector string) (MnemonicProject, error) {
	if file == nil || len(file.Projects) == 0 {
		return MnemonicProject{}, app.NewNotFoundError("no projects found in .mnemonic", nil)
	}

	if selector == "" {
		if len(file.Projects) == 1 {
			return file.Projects[0], nil
		}
		return MnemonicProject{}, app.NewAmbiguousError("multiple projects found in .mnemonic; use --project", nil)
	}

	var matched *MnemonicProject
	for i := range file.Projects {
		project := &file.Projects[i]
		if project.ID == selector || project.Slug == selector {
			if matched != nil {
				return MnemonicProject{}, app.NewAmbiguousError(fmt.Sprintf("project selector %q matches multiple entries", selector), nil)
			}
			matched = project
		}
	}
	if matched == nil {
		return MnemonicProject{}, app.NewNotFoundError(fmt.Sprintf("project %q not found", selector), nil)
	}

	return *matched, nil
}
