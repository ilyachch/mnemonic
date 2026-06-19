package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "mnemonic",
	Short: "mnemonic is a local-first personal knowledge base and search engine",
	Long:  `mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, calculates page ranks, structures wiki-links, and searches utilizing SQLite FTS5.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	defer closeAppContainer()

	err := RootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitCodeForError(err))
	}
}

var (
	jsonFlag    bool
	projectFlag string
)

func init() {
	RootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output in JSON format")
	RootCmd.PersistentFlags().StringVarP(&projectFlag, "project", "p", "", "select a project by slug or UUID")
	if err := RootCmd.RegisterFlagCompletionFunc("project", completeProjectNames); err != nil {
		panic(fmt.Sprintf("register completion for --project: %v", err))
	}
}

// projectSelectorValue returns the project selector in precedence order:
// explicit CLI flag, then MNEMONIC_PROJECT environment variable, then empty.
func projectSelectorValue() string {
	if projectFlag != "" {
		return projectFlag
	}
	return os.Getenv("MNEMONIC_PROJECT")
}
