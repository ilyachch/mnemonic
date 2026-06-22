package cli

import (
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/apperr"
)

func requireRuntimeSearchIndex(runtime *app.RuntimeApp) error {
	exists, err := runtime.Services.Search.Index.Exists()
	if err != nil {
		return err
	}
	if !exists {
		return apperr.NotFound("index missing; run `mnemonic project reindex`", nil)
	}
	return nil
}
