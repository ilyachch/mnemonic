package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/spf13/cobra"
)

// SharedOptions holds root CLI flags shared across runtime commands.
type SharedOptions struct {
	JSON    bool
	Project string
}

// ProjectSelector returns the selected project in precedence order.
func (o *SharedOptions) ProjectSelector() string {
	if o != nil && o.Project != "" {
		return o.Project
	}
	return os.Getenv("MNEMONIC_PROJECT")
}

type cliState struct {
	boot   *app.Bootstrap
	shared *SharedOptions
}

type cliStateKey struct{}

// NewRootCommand builds the CLI root command with explicit bootstrap wiring.
func NewRootCommand(boot *app.Bootstrap) *cobra.Command {
	state := &cliState{
		boot:   boot,
		shared: &SharedOptions{},
	}

	root := &cobra.Command{
		Use:   "mnemonic",
		Short: "mnemonic is a local-first personal knowledge base and search engine",
		Long:  `mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, calculates page ranks, structures wiki-links, and searches utilizing SQLite FTS5.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetContext(context.WithValue(context.Background(), cliStateKey{}, state))

	root.PersistentFlags().BoolVar(&state.shared.JSON, "json", false, "output in JSON format")
	root.PersistentFlags().StringVarP(&state.shared.Project, "project", "p", "", "select a project by slug or UUID")
	if err := root.RegisterFlagCompletionFunc("project", completeProjectNames); err != nil {
		panic(fmt.Sprintf("register completion for --project: %v", err))
	}

	for _, cmd := range buildCommandTree() {
		root.AddCommand(cmd)
	}
	return root
}

// Execute runs the CLI with the provided bootstrap.
func Execute(boot *app.Bootstrap) error {
	root := NewRootCommand(boot)
	return root.ExecuteContext(root.Context())
}
