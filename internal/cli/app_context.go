package cli

import "github.com/ilyachch/mnemonic/internal/app"

var currentApp *app.App

func initAppContainer() error {
	container, err := app.New(app.Input{})
	if err != nil {
		return err
	}
	currentApp = container
	return nil
}

func closeAppContainer() {
	if currentApp == nil {
		return
	}
	_ = currentApp.Close()
	currentApp = nil
}

func mustAppContainer() (*app.App, error) {
	if currentApp == nil {
		if err := initAppContainer(); err != nil {
			return nil, err
		}
	}
	return currentApp, nil
}
