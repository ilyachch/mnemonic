package cli

import (
	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of mnemonic",
	RunE: func(cmd *cobra.Command, args []string) error {
		version := buildinfo.Version()
		data := struct {
			Version string `json:"version"`
		}{
			Version: version,
		}
		return PrintOutput(cmd.OutOrStdout(), version+"\n", data)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
