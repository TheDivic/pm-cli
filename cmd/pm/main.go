// Command pm is the Plaintext Tasks CLI: it reads, validates, queries, formats,
// and safely mutates *.tasks.yaml project files.
package main

import (
	"os"

	"github.com/TheDivic/plaintext-tasks/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
