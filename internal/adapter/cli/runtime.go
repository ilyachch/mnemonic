package cli

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/spf13/cobra"
)

func runtimeAppForSelector(ctx context.Context, selector string) (*app.RuntimeApp, error) {
	boot, err := bootstrapFromContext(ctx)
	if err != nil {
		return nil, err
	}

	logger := loggerFromContext(ctx)

	return boot.Runtime(ctx, selector, logger)
}

func runtimeAppForSelectedProject(cmd *cobra.Command) (*app.RuntimeApp, error) {
	return runtimeAppForSelector(commandContext(cmd), projectSelectorValue(cmd))
}
