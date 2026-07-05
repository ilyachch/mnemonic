package cli

import (
	"context"
	"log/slog"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/spf13/cobra"
)

func runtimeAppForSelector(ctx context.Context, selector string) (*app.RuntimeApp, error) {
	boot, err := bootstrapFromContext(ctx)
	if err != nil {
		return nil, err
	}

	boot.Logger = loggerFromContext(ctx)

	return boot.Runtime(ctx, selector)
}

func runtimeAppForSelectedProject(cmd *cobra.Command) (*app.RuntimeApp, error) {
	return runtimeAppForSelector(commandContext(cmd), projectSelectorValue(cmd))
}

func injectLoggerToBootstrap(boot *app.Bootstrap, logger *slog.Logger) {
	if boot == nil || logger == nil || boot.Services.Catalog == nil {
		return
	}
	boot.Logger = logger
	boot.Services.Catalog.Logger = logger
}
