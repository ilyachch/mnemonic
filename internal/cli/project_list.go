package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		result, err := container.Services.Catalog.List()
		if err != nil {
			return err
		}

		output := projectListOutput{Projects: make([]projectListItem, 0, len(result.Projects))}
		for _, project := range result.Projects {
			output.Projects = append(output.Projects, projectListItem{
				ProjectID:    project.ProjectID,
				Name:         project.Name,
				Slug:         project.Slug,
				Type:         project.Type,
				MemoriesPath: project.MemoriesPath,
				StatePath:    project.StatePath,
				Status:       project.Status,
				Issue:        project.Issue,
			})
		}
		if output.Projects == nil {
			output.Projects = []projectListItem{}
		}

		human := formatProjectListHuman(output.Projects)
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	RootCmd.AddCommand(projectCmd)
}

type projectListOutput struct {
	Projects []projectListItem `json:"projects"`
}

type projectListItem struct {
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
	MemoriesPath string `json:"memories_path"`
	StatePath    string `json:"state_path"`
	Status       string `json:"status"`
	Issue        string `json:"issue,omitempty"`
}

func formatProjectListHuman(projects []projectListItem) string {
	if len(projects) == 0 {
		return "0 projects\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d projects\n", len(projects))
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tSLUG\tTYPE\tPATH\tSTATUS")
	for _, p := range projects {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Slug, p.Type, p.MemoriesPath, p.Status)
	}
	_ = w.Flush()
	return b.String()
}
