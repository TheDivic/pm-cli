package cli

import "github.com/spf13/cobra"

// Build metadata, overridden at build time with
// -ldflags "-X github.com/TheDivic/plaintext-tasks/internal/cli.version=... ".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the pm version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("pm %s (commit %s, built %s)\n", version, commit, date)
			return nil
		},
	}
}
