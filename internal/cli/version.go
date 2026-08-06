package cli

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Build metadata, overridden at build time with
// -ldflags "-X github.com/TheDivic/pm-cli/internal/cli.version=... ".
// Binaries produced by `go install` carry no ldflags, so the values fall back to
// the module version and VCS stamps the Go toolchain embeds automatically.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// buildMetadata reports the version, commit, and build date, preferring
// ldflags-injected values and falling back to the embedded build info.
func buildMetadata() (ver, rev, built string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, date
	}
	var vcsRev, vcsTime string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			vcsRev = s.Value
		case "vcs.time":
			vcsTime = s.Value
		}
	}
	return mergeBuildInfo(version, commit, date, info.Main.Version, vcsRev, vcsTime)
}

// mergeBuildInfo fills each unset placeholder from the toolchain-embedded build
// info. An ldflags-injected value always wins, so goreleaser and `make build`
// output is unaffected.
func mergeBuildInfo(ver, rev, built, mainVer, vcsRev, vcsTime string) (string, string, string) {
	if ver == "dev" && mainVer != "" && mainVer != "(devel)" {
		ver = mainVer
	}
	if rev == "none" && vcsRev != "" {
		rev = vcsRev
		if len(rev) > 7 {
			rev = rev[:7]
		}
	}
	if built == "unknown" && vcsTime != "" {
		built = vcsTime
	}
	return ver, rev, built
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the pm version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ver, rev, built := buildMetadata()
			cmd.Printf("pm %s (commit %s, built %s)\n", ver, rev, built)
			return nil
		},
	}
}
