package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/platform/logging"
	"github.com/spf13/cobra"
)

// SharedOptions holds root CLI flags shared across runtime commands.
type SharedOptions struct {
	JSON      bool
	Project   string
	LogLevel  string
	LogFormat string
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
	logger *slog.Logger
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
		Long:  `mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, provides graph-aware reranking, structures wiki-links, and searches utilizing SQLite FTS5.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgLevel, cfgFormat := "info", "text"
			if boot != nil && boot.Config != nil {
				if boot.Config.Logging.Level != "" {
					cfgLevel = boot.Config.Logging.Level
				}
				if boot.Config.Logging.Format != "" {
					cfgFormat = boot.Config.Logging.Format
				}
			}
			level := resolveLogLevel(state.shared.LogLevel, cfgLevel)
			format := resolveLogFormat(state.shared.LogFormat, cfgFormat)

			if err := logging.ValidateLevel(level); err != nil {
				return err
			}
			if err := logging.ValidateFormat(format); err != nil {
				return err
			}

			state.logger = logging.New(level, format)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetContext(context.WithValue(context.Background(), cliStateKey{}, state))

	root.PersistentFlags().BoolVar(&state.shared.JSON, "json", false, "output in JSON format")
	root.PersistentFlags().StringVarP(&state.shared.Project, "project", "p", "", "select a project by slug or UUID")
	root.PersistentFlags().StringVar(&state.shared.LogLevel, "log-level", "", "log level (debug, info, warn, error)")
	root.PersistentFlags().StringVar(&state.shared.LogFormat, "log-format", "", "log format (text, json)")
	if err := root.RegisterFlagCompletionFunc("project", completeProjectNames); err != nil {
		panic(fmt.Sprintf("register completion for --project: %v", err))
	}

	for _, cmd := range buildCommandTree() {
		root.AddCommand(cmd)
	}
	return root
}

func resolveLogLevel(flagValue, configValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("MNEMONIC_LOG_LEVEL"); env != "" {
		return env
	}
	if configValue != "" {
		return configValue
	}
	return "info"
}

func resolveLogFormat(flagValue, configValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("MNEMONIC_LOG_FORMAT"); env != "" {
		return env
	}
	if configValue != "" {
		return configValue
	}
	return "text"
}

// Execute runs the CLI with the provided bootstrap.
func Execute(boot *app.Bootstrap) error {
	root := NewRootCommand(boot)
	return root.ExecuteContext(root.Context())
}
