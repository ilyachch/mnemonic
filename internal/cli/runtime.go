package cli

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/app"
)

func runtimeAppForSelector(ctx context.Context, selector string) (*app.RuntimeApp, error) {
	container, err := mustAppContainer()
	if err != nil {
		return nil, err
	}

	return container.Runtime(ctx, selector)
}

func runtimeAppForSelectedProject(ctx context.Context) (*app.RuntimeApp, error) {
	return runtimeAppForSelector(ctx, projectSelectorValue())
}
