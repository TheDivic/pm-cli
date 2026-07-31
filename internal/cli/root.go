package cli

import (
	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-tasks/internal/clock"
)

// GlobalOptions holds the flags shared by every command. They are bound on the
// root command's persistent flag set and read back after execution (for
// example to choose JSON error rendering).
type GlobalOptions struct {
	// Root is the discovery root. Empty means the current working directory.
	Root string
	// JSON selects machine-readable output for agents and automation.
	JSON bool
	// NoIgnore disables .gitignore filtering during discovery.
	NoIgnore bool
}

// newRootCmd builds the pm command tree. The clock is threaded in so that
// mutation commands take their lifecycle dates from an injectable source rather
// than the machine date.
func newRootCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	root := &cobra.Command{
		Use:   "pm",
		Short: "Plaintext Tasks: manage projects and tasks stored as *.tasks.yaml files",
		Long: "pm reads, validates, queries, formats, and safely mutates *.tasks.yaml\n" +
			"project files. People get concise output; agents can request --json.",
		// We render errors and usage ourselves so the exit-code and JSON-error
		// contract stays exact.
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.CompletionOptions.DisableDefaultCmd = true

	pf := root.PersistentFlags()
	pf.StringVar(&opts.Root, "root", "", "discovery root (default: $PM_ROOT or current directory)")
	pf.BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON")
	pf.BoolVar(&opts.NoIgnore, "no-ignore", false, "include .gitignored paths in discovery")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newProjectsCmd(opts, clk))
	root.AddCommand(newTasksCmd(opts, clk))
	return root
}
