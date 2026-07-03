package cli

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/spf13/cobra"
)

func commandContext(cmd *cobra.Command) context.Context {
	if cmd == nil {
		return context.Background()
	}
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	if root := cmd.Root(); root != nil && root.Context() != nil {
		return root.Context()
	}
	return context.Background()
}

func cliStateFromContext(ctx context.Context) *cliState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(cliStateKey{}).(*cliState)
	return state
}

func bootstrapFromContext(ctx context.Context) (*app.Bootstrap, error) {
	state := cliStateFromContext(ctx)
	if state == nil || state.boot == nil {
		return nil, errors.New("application bootstrap is not configured")
	}
	return state.boot, nil
}

func sharedOptionsFromContext(ctx context.Context) *SharedOptions {
	state := cliStateFromContext(ctx)
	if state == nil {
		return nil
	}
	return state.shared
}

func jsonOutputEnabled(cmd *cobra.Command) bool {
	shared := sharedOptionsFromContext(commandContext(cmd))
	return shared != nil && shared.JSON
}

func projectSelectorValue(cmd *cobra.Command) string {
	shared := sharedOptionsFromContext(commandContext(cmd))
	if shared != nil {
		return shared.ProjectSelector()
	}
	return os.Getenv("MNEMONIC_PROJECT")
}

func loggerFromContext(ctx context.Context) *slog.Logger {
	state := cliStateFromContext(ctx)
	if state == nil {
		return nil
	}
	return state.logger
}
