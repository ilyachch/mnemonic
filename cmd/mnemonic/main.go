package main

import (
	"fmt"
	"os"

	"github.com/ilyachch/mnemonic/internal/adapter/cli"
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/platform/buildinfo"
)

var version = "dev"

func main() {
	buildinfo.SetVersion(version)
	boot, err := app.New(app.Input{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCodeForError(err))
	}
	defer func() {
		_ = boot.Close()
	}()

	if err := cli.Execute(boot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCodeForError(err))
	}
}
