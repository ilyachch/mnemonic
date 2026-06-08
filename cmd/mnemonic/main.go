package main

import (
	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/ilyachch/mnemonic/internal/cli"
)

var version = "dev"

func main() {
	buildinfo.SetVersion(version)
	cli.Execute()
}
