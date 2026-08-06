// Command pm is the PM CLI: it reads, validates, queries, formats,
// and safely mutates *.tasks.yaml project files.
package main

import (
	"os"

	"github.com/TheDivic/pm-cli/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
