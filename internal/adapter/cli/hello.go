package cli

import (
	"github.com/spf13/cobra"
)

var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Prints hello world",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("hello world")
	},
}
