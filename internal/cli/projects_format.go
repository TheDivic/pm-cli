package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-projects/internal/discover"
	"github.com/TheDivic/plaintext-projects/internal/emit"
	"github.com/TheDivic/plaintext-projects/internal/fsatomic"
	"github.com/TheDivic/plaintext-projects/internal/pmerr"
)

func newProjectsFormatCmd(opts *GlobalOptions) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "format [project-id]",
		Short: "Rewrite task files in canonical form",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case all && len(args) > 0:
				return pmerr.Usage("cannot combine a project id with --all")
			case !all && len(args) == 0:
				return pmerr.Usage("provide a project id or --all")
			}

			ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
			if err != nil {
				return pmerr.IO("cannot discover projects: %v", err)
			}
			targets, err := selectTargets(ws, args, all)
			if err != nil {
				return err
			}

			// format rejects invalid semantics rather than guessing repairs, and
			// writes nothing when any target is invalid.
			if invalid := invalidTargets(targets); len(invalid) > 0 {
				reportInvalid(cmd.ErrOrStderr(), invalid)
				return pmerr.Validation("cannot format: %d file(s) are invalid", len(invalid))
			}

			formatted, unchanged, err := applyFormat(targets)
			if err != nil {
				return err
			}
			if opts.JSON {
				return writeFormatJSON(cmd.OutOrStdout(), formatted, unchanged)
			}
			writeFormatText(cmd.OutOrStdout(), formatted, unchanged)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "format every discovered project")
	return cmd
}

func invalidTargets(targets []*discover.Project) []*discover.Project {
	var invalid []*discover.Project
	for _, p := range targets {
		if p.LoadErr != nil || len(p.Findings) > 0 {
			invalid = append(invalid, p)
		}
	}
	return invalid
}

func reportInvalid(w io.Writer, invalid []*discover.Project) {
	for _, p := range invalid {
		fmt.Fprintf(w, "%s:\n", p.Path)
		if p.LoadErr != nil {
			fmt.Fprintf(w, "  - %s\n", p.LoadErr)
		}
		for _, f := range p.Findings {
			fmt.Fprintf(w, "  - %s\n", f.String())
		}
	}
}

// applyFormat writes canonical bytes for any target whose current bytes differ,
// and returns the changed and unchanged paths.
func applyFormat(targets []*discover.Project) (formatted, unchanged []string, err error) {
	for _, p := range targets {
		out := emit.Document(p.Doc)
		cur, readErr := os.ReadFile(p.AbsPath)
		if readErr != nil {
			return nil, nil, pmerr.IO("cannot read %s: %v", p.Path, readErr)
		}
		if bytes.Equal(cur, out) {
			unchanged = append(unchanged, p.Path)
			continue
		}
		if writeErr := fsatomic.WriteFile(p.AbsPath, out, 0o644); writeErr != nil {
			return nil, nil, pmerr.IO("cannot write %s: %v", p.Path, writeErr)
		}
		formatted = append(formatted, p.Path)
	}
	return formatted, unchanged, nil
}

func writeFormatText(w io.Writer, formatted, unchanged []string) {
	for _, p := range formatted {
		fmt.Fprintf(w, "formatted %s\n", p)
	}
	for _, p := range unchanged {
		fmt.Fprintf(w, "unchanged %s\n", p)
	}
}

func writeFormatJSON(w io.Writer, formatted, unchanged []string) error {
	out := struct {
		Formatted []string `json:"formatted"`
		Unchanged []string `json:"unchanged"`
	}{Formatted: orEmpty(formatted), Unchanged: orEmpty(unchanged)}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
