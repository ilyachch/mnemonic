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
		boot, err := bootstrapFromContext(commandContext(cmd))
		if err != nil {
			return err
		}

		resolvedProject, err := boot.Services.ProjectResolver.Resolve(app.ProjectResolveInput{
			ProjectSelector:  projectSelectorValue(cmd),
			EnvironmentValue: os.Getenv(project.EnvironmentProjectSelector),
		})
		if err != nil {
			return err
		}

		server := mcp.NewServer(resolvedProject, boot.Paths, mcpReadOnlyEnabled(cmd))
		return server.Run(commandContext(cmd), &sdkmcp.StdioTransport{})
	},
}

func mcpReadOnlyEnabled(cmd *cobra.Command) bool {
	enabled, err := cmd.Flags().GetBool("read-only")
	if err != nil {
		return false
	}
	if enabled {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MNEMONIC_READ_ONLY"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
