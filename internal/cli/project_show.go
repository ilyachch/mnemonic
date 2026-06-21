package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectShowCmd = &cobra.Command{
	Use:               "show SLUG",
	Short:             "Show a registered project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelector(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		project := projectShowOutput{
			ProjectID: runtime.KB.ID,
			Name:      runtime.KB.Name,
			Slug:      runtime.KB.Slug,
			Type:      runtime.KB.Kind,
			StateHome: runtime.KB.StateDir,
			Location: projectLocationOutput{
				MemoriesAbs: runtime.KB.RootDir,
				ManifestAbs: runtime.KB.ManifestPath,
				RepoRootAbs: runtime.KB.RepoRootDir,
			},
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
