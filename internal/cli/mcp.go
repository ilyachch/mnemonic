package cli

import (
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/mcp"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
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

		server := mcp.NewServer(resolvedProject, container.Paths, mcpReadOnlyEnabled())
		return server.Run(cmd.Context(), &sdkmcp.StdioTransport{})
	},
}

var mcpReadOnlyFlag bool

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().BoolVar(&mcpReadOnlyFlag, "read-only", false, "run the MCP server without write tools")
}

func mcpReadOnlyEnabled() bool {
	if mcpReadOnlyFlag {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MNEMONIC_READ_ONLY"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
