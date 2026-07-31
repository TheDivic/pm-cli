package cli

import "github.com/spf13/cobra"

// version is the CLI version. It defaults to "dev" and can be overridden at
// build time with -ldflags "-X ...cli.version=<value>".
var version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the pm version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(version)
			return nil
		},
	}
}
