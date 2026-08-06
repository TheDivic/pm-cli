// Package cli wires the pm command tree, parses global flags, and maps
// structured errors to process exit codes and output rendering.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/TheDivic/pm-cli/internal/clock"
	"github.com/TheDivic/pm-cli/internal/pmerr"
)

// Main runs the pm CLI with the given arguments and streams, returning the
// process exit code. It never calls os.Exit, so it is straightforward to test.
func Main(args []string, stdout, stderr io.Writer) int {
	opts := &GlobalOptions{}
	root := newRootCmd(opts, clock.System{})
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	if err := root.Execute(); err != nil {
		// opts.JSON is authoritative for valid commands. When cobra fails before
		// parsing persistent flags (an unknown command or flag), fall back to
		// scanning the raw arguments so error output still honors --json.
		return report(err, opts.JSON || wantsJSON(args), stderr)
	}
	return 0
}

// wantsJSON reports whether --json appears among the arguments before an
// end-of-flags "--" terminator. It is only a fallback for errors raised before
// cobra binds the flag.
func wantsJSON(args []string) bool {
	for _, a := range args {
		switch {
		case a == "--":
			return false
		case a == "--json":
			return true
		case strings.HasPrefix(a, "--json="):
			return a == "--json=true" || a == "--json=1"
		}
	}
	return false
}

// report renders err (human-readable or JSON) to stderr and returns its exit
// code. Errors that are not already *pmerr.Error come from cobra's own flag and
// argument parsing, which are command-syntax problems.
func report(err error, jsonMode bool, stderr io.Writer) int {
	var pe *pmerr.Error
	if !errors.As(err, &pe) {
		pe = pmerr.New(pmerr.CodeUsage, err.Error())
	}

	if jsonMode {
		enc := json.NewEncoder(stderr)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(pe.Detail())
	} else {
		_, _ = fmt.Fprintln(stderr, "pm: "+pe.Error())
	}
	return pe.Code.ExitCode()
}
