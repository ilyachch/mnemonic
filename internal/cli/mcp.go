package cli

import (
	"os"

	"github.com/ilyachch/mnemonic/internal/mcp"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:          "mcp",
	Short:        "Run the MCP stdio adapter",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectSelector, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}
		envSelector := os.Getenv("MNEMONIC_PROJECT")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
			CWD:              cwd,
			ProjectSelector:  projectSelector,
			EnvironmentValue: envSelector,
		})
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		server := mcp.NewServer(resolvedProject, container.Paths)
		return server.Run(cmd.Context())
	},
}

func init() {
	mcpCmd.Flags().String("project", "", "select a project")
	RootCmd.AddCommand(mcpCmd)
}
