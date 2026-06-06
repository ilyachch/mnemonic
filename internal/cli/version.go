package cli

import (
	"github.com/spf13/cobra"
)

// Version is the current version of mnemonic, defaulted to dev build.
var Version = "0.1.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of mnemonic",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := struct {
			Version string `json:"version"`
		}{
			Version: Version,
		}
		return PrintOutput(cmd.OutOrStdout(), Version+"\n", data)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
