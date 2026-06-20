package cli

import "github.com/spf13/cobra"

var webCmd = &cobra.Command{
	Use:          "web",
	Short:        "Manage the web MCP server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(webCmd)
}
