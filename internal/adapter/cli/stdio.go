package cli

import (
	"context"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/adapter/stdio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var stdioCmd = &cobra.Command{
	Use:          "stdio",
	Short:        "Run the MCP stdio adapter",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		server, err := stdio.NewServer(runtime.KB, stdio.Dependencies{
			Notes:  runtime.Services.Notes,
			Search: runtime.Services.Search,
			Index:  runtime.Services.Index,
		}, stdioReadOnlyEnabled(cmd))
		if err != nil {
			return err
		}

		return stdioRun(server, commandContext(cmd), &mcp.StdioTransport{})
	},
}

var mcpCmd = &cobra.Command{
	Use:          "mcp",
	Short:        "Run the MCP stdio adapter",
	Hidden:       true,
	Deprecated:   "use `stdio` instead",
	SilenceUsage: true,
	RunE:         stdioCmd.RunE,
}

var stdioRun = stdioRunReal

func stdioRunReal(server *stdio.Server, ctx context.Context, transport mcp.Transport) error {
	return server.Run(ctx, transport)
}

func stdioReadOnlyEnabled(cmd *cobra.Command) bool {
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
