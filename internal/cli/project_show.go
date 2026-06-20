package cli

import (
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/spf13/cobra"
)

var projectShowCmd = &cobra.Command{
	Use:               "show SLUG",
	Short:             "Show a registered project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		project, err := loadProjectBySelector(container.Paths, args[0])
		if err != nil {
			return err
		}

		human := fmt.Sprintf("%s %s\n", project.ProjectID, project.Name)
		return PrintOutput(cmd.OutOrStdout(), human, project)
	},
}

func init() {
	projectCmd.AddCommand(projectShowCmd)
}

type projectShowOutput struct {
	ProjectID string                `json:"project_id"`
	Name      string                `json:"name"`
	Slug      string                `json:"slug"`
	Type      string                `json:"type"`
	StateHome string                `json:"state_home"`
	Location  projectLocationOutput `json:"location"`
}

type projectLocationOutput struct {
	MemoriesAbs string `json:"memories_abs"`
	ManifestAbs string `json:"manifest_abs"`
	RepoRootAbs string `json:"repo_root_abs"`
}

func loadProjectBySelector(effectivePaths paths.EffectivePaths, selector string) (projectShowOutput, error) {
	entry, err := registry.Resolve(effectivePaths.MemoriesHome, selector)
	if err != nil {
		return projectShowOutput{}, err
	}

	return projectShowOutput{
		ProjectID: entry.ProjectID,
		Name:      entry.Name,
		Slug:      entry.Slug,
		Type:      entry.Type,
		StateHome: filepath.Join(effectivePaths.StateHome, "mnemonic", "projects", entry.ProjectID),
		Location: projectLocationOutput{
			MemoriesAbs: entry.MemoriesAbs,
			ManifestAbs: entry.ManifestPath,
			RepoRootAbs: entry.RepoRootAbs,
		},
	}, nil
}
