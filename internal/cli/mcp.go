package cli

import (
	"os"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/mcp"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:          "mcp",
	Short:        "Run the MCP stdio adapter",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		resolvedProject, err := container.Services.ProjectResolver.Resolve(app.ProjectResolveInput{
			ProjectSelector:  projectSelectorValue(),
			EnvironmentValue: os.Getenv(project.EnvironmentProjectSelector),
		})
		if err != nil {
			return err
		}

		server := mcp.NewServer(resolvedProject, container.Paths)
		return server.Run(cmd.Context())
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
}
