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

	runtime, err := boot.Runtime(ctx, selector)
	if err != nil {
		return nil, err
	}

	logger := loggerFromContext(ctx)
	if logger != nil {
		if runtime.Services.Notes != nil {
			runtime.Services.Notes.Logger = logger
		}
		if runtime.Services.Search != nil {
			runtime.Services.Search.Logger = logger
		}
		if runtime.Services.Index != nil {
			runtime.Services.Index.Logger = logger
		}
	}

	return runtime, nil
}

func runtimeAppForSelectedProject(cmd *cobra.Command) (*app.RuntimeApp, error) {
	return runtimeAppForSelector(commandContext(cmd), projectSelectorValue(cmd))
}

func injectLoggerToBootstrap(boot *app.Bootstrap, logger *slog.Logger) {
	if boot == nil || logger == nil || boot.Services.Catalog == nil {
		return
	}
	boot.Services.Catalog.Logger = logger
}
