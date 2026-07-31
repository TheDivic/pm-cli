package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-tasks/internal/discover"
	"github.com/TheDivic/plaintext-tasks/internal/pmerr"
	"github.com/TheDivic/plaintext-tasks/internal/validate"
)

func newProjectsCmd(opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Inspect and manage projects",
	}
	cmd.AddCommand(newProjectsListCmd(opts))
	cmd.AddCommand(newProjectsValidateCmd(opts))
	return cmd
}

func rootOrCWD(opts *GlobalOptions) string {
	if opts.Root == "" {
		return "."
	}
	return opts.Root
}

// ---- projects list ----

func newProjectsListCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List discovered projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := discover.Discover(rootOrCWD(opts))
			if err != nil {
				return pmerr.IO("cannot discover projects: %v", err)
			}
			ordered := listOrder(ws)
			if opts.JSON {
				return writeProjectsListJSON(cmd.OutOrStdout(), ordered)
			}
			return writeProjectsListText(cmd.OutOrStdout(), cmd.ErrOrStderr(), ordered, ws)
		},
	}
}

type projectSummary struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Priority *int     `json:"priority,omitempty"`
	Areas    []string `json:"areas,omitempty"`
}

// listOrder returns the successfully decoded projects sorted for display:
// prioritized projects first (lowest number = highest priority), projects
// without a priority last, ties broken by creation date (oldest first) and then
// project ID for deterministic output.
func listOrder(ws *discover.Workspace) []*discover.Project {
	out := make([]*discover.Project, 0, len(ws.Projects))
	for i := range ws.Projects {
		if ws.Projects[i].Doc != nil {
			out = append(out, &ws.Projects[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Doc.Project, out[j].Doc.Project
		if (a.Priority == nil) != (b.Priority == nil) {
			return a.Priority != nil // prioritized projects sort first
		}
		if a.Priority != nil && b.Priority != nil && *a.Priority != *b.Priority {
			return *a.Priority < *b.Priority
		}
		if a.Created != b.Created {
			return a.Created < b.Created // YYYY-MM-DD sorts chronologically
		}
		return a.ID < b.ID
	})
	return out
}

func writeProjectsListJSON(w io.Writer, ordered []*discover.Project) error {
	out := struct {
		Projects []projectSummary `json:"projects"`
	}{Projects: []projectSummary{}}
	for _, p := range ordered {
		out.Projects = append(out.Projects, projectSummary{
			ID:       p.Doc.Project.ID,
			Title:    p.Doc.Project.Title,
			Status:   string(p.Doc.Project.Status),
			Priority: p.Doc.Project.Priority,
			Areas:    p.Doc.Project.Areas,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeProjectsListText(stdout, stderr io.Writer, ordered []*discover.Project, ws *discover.Workspace) error {
	tw := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tSTATUS\tPRIO\tCREATED")
	for _, p := range ordered {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			p.Doc.Project.ID, p.Doc.Project.Title, p.Doc.Project.Status,
			priorityLabel(p.Doc.Project.Priority), dateLabel(p.Doc.Project.Created))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	// Load failures and conflicts are surfaced as non-fatal warnings; use
	// `projects validate` for a full report.
	for i := range ws.Projects {
		if p := &ws.Projects[i]; p.LoadErr != nil {
			fmt.Fprintf(stderr, "warning: %s: %v\n", p.Path, p.LoadErr)
		}
	}
	for _, c := range ws.Conflicts {
		fmt.Fprintf(stderr, "warning: duplicate %s %q in %v\n", c.Kind, c.Value, c.Paths)
	}
	return nil
}

// ---- projects validate ----

func newProjectsValidateCmd(opts *GlobalOptions) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "validate [project-id]",
		Short: "Validate discovered project task files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := discover.Discover(rootOrCWD(opts))
			if err != nil {
				return pmerr.IO("cannot discover projects: %v", err)
			}
			targets, err := selectTargets(ws, args, all)
			if err != nil {
				return err
			}
			report := buildValidationReport(ws, targets, len(args) == 0 || all)
			if opts.JSON {
				if err := writeValidationJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				writeValidationText(cmd.OutOrStdout(), report)
			}
			if !report.Valid {
				return pmerr.Validation("validation found %d problem(s)", report.problemCount())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "validate every discovered project (the default)")
	return cmd
}

func selectTargets(ws *discover.Workspace, args []string, _ bool) ([]*discover.Project, error) {
	if len(args) == 0 {
		out := make([]*discover.Project, 0, len(ws.Projects))
		for i := range ws.Projects {
			out = append(out, &ws.Projects[i])
		}
		return out, nil
	}
	id := args[0]
	for i := range ws.Projects {
		if ws.Projects[i].ID() == id {
			return []*discover.Project{&ws.Projects[i]}, nil
		}
	}
	return nil, pmerr.Usage("no project with id %q under the discovery root", id).WithProject(id)
}

type fileReport struct {
	Path    string             `json:"path"`
	Project string             `json:"project,omitempty"`
	LoadErr string             `json:"load_error,omitempty"`
	Errors  []validate.Finding `json:"errors,omitempty"`
}

type validationReport struct {
	Valid     bool                `json:"valid"`
	Files     []fileReport        `json:"files"`
	Conflicts []discover.Conflict `json:"conflicts,omitempty"`
}

func (r *validationReport) problemCount() int {
	n := len(r.Conflicts)
	for _, f := range r.Files {
		if f.LoadErr != "" {
			n++
		}
		n += len(f.Errors)
	}
	return n
}

func buildValidationReport(ws *discover.Workspace, targets []*discover.Project, includeConflicts bool) *validationReport {
	r := &validationReport{Valid: true, Files: []fileReport{}}
	for _, p := range targets {
		fr := fileReport{Path: p.Path, Project: p.ID()}
		if p.LoadErr != nil {
			fr.LoadErr = p.LoadErr.Error()
			r.Valid = false
		}
		if len(p.Findings) > 0 {
			fr.Errors = p.Findings
			r.Valid = false
		}
		r.Files = append(r.Files, fr)
	}
	if includeConflicts && len(ws.Conflicts) > 0 {
		r.Conflicts = ws.Conflicts
		r.Valid = false
	}
	return r
}

func writeValidationJSON(w io.Writer, r *validationReport) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func writeValidationText(w io.Writer, r *validationReport) {
	if r.Valid {
		fmt.Fprintf(w, "%d project(s) valid\n", len(r.Files))
		return
	}
	for _, f := range r.Files {
		if f.LoadErr == "" && len(f.Errors) == 0 {
			continue
		}
		fmt.Fprintf(w, "%s:\n", f.Path)
		if f.LoadErr != "" {
			fmt.Fprintf(w, "  - %s\n", f.LoadErr)
		}
		for _, e := range f.Errors {
			fmt.Fprintf(w, "  - %s\n", e.String())
		}
	}
	for _, c := range r.Conflicts {
		fmt.Fprintf(w, "duplicate %s %q in %v\n", c.Kind, c.Value, c.Paths)
	}
}

func priorityLabel(p *int) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *p)
}

func dateLabel(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
