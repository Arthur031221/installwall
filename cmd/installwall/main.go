// Command installwall is a firewall for what your AI coding agent
// installs. See README.md for usage.
package main

import (
	"os"

	"github.com/Arthur031221/installwall/internal/cli"
)

// version is set by -ldflags at build time.
var version = "dev"

func main() {
	cli.Version = version
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
