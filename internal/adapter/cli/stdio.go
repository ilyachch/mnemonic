package cli

import (
	"context"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/adapter/stdio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func newStdioCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "stdio",
		Short:        "Run the MCP stdio adapter",
		SilenceUsage: true,
		RunE:         runStdioAdapter,
	}
	cmd.Flags().Bool("read-only", false, "run the MCP server without write tools")
	return cmd
}

func newMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mcp",
		Short:        "Run the MCP stdio adapter",
		Hidden:       true,
		Deprecated:   "use `stdio` instead",
		SilenceUsage: true,
		RunE:         runStdioAdapter,
	}
	cmd.Flags().Bool("read-only", false, "run the MCP server without write tools")
	return cmd
}

func runStdioAdapter(cmd *cobra.Command, args []string) error {
	runtime, err := runtimeAppForSelectedProject(cmd)
	if err != nil {
		return err
	}

	srv, err := stdio.NewServer(runtime.KB, stdio.Dependencies{
		Notes:  runtime.Services.Notes,
		Search: runtime.Services.Search,
		Index:  runtime.Services.Index,
	}, stdioReadOnlyEnabled(cmd))
	if err != nil {
		return err
	}

	srv.Logger = loggerFromContext(commandContext(cmd))

	return stdioRun(srv, commandContext(cmd), &mcp.StdioTransport{})
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
