package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TheDivic/pm-cli/internal/clock"
	"github.com/TheDivic/pm-cli/internal/discover"
	"github.com/TheDivic/pm-cli/internal/mdrender"
	"github.com/TheDivic/pm-cli/internal/model"
	"github.com/TheDivic/pm-cli/internal/pmerr"
	"github.com/TheDivic/pm-cli/internal/validate"
)

// rootEnvVar names the environment variable that supplies a default discovery
// root when --root is not given.
const rootEnvVar = "PM_ROOT"

func newProjectsCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Inspect and manage projects",
	}
	cmd.AddCommand(newProjectsListCmd(opts))
	cmd.AddCommand(newProjectsShowCmd(opts))
	cmd.AddCommand(newProjectsValidateCmd(opts))
	cmd.AddCommand(newProjectsFormatCmd(opts))
	cmd.AddCommand(newProjectsCreateCmd(opts, clk))
	cmd.AddCommand(newProjectsEditCmd(opts))
	cmd.AddCommand(newProjectsStatusCmd(opts, clk))
	return cmd
}

// ---- projects show ----

func newProjectsShowCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show <project-id>",
		Short: "Show a project's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
			if err != nil {
				return pmerr.IO("cannot discover projects: %v", err)
			}
			var target *discover.Project
			for i := range ws.Projects {
				if ws.Projects[i].ID() == args[0] {
					target = &ws.Projects[i]
					break
				}
			}
			if target == nil {
				return pmerr.Usage("no project with id %q under the discovery root", args[0]).WithProject(args[0])
			}
			if opts.JSON {
				return writeProjectShowJSON(cmd.OutOrStdout(), target)
			}
			writeProjectShowText(cmd.OutOrStdout(), target)
			writeProjectDoc(cmd.OutOrStdout(), target)
			return nil
		},
	}
}

type projectDetail struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	TaskIDPrefix string              `json:"task_id_prefix"`
	Status       string              `json:"status"`
	Priority     *int                `json:"priority,omitempty"`
	Areas        []string            `json:"areas,omitempty"`
	Created      string              `json:"created"`
	Started      string              `json:"started,omitempty"`
	Due          string              `json:"due,omitempty"`
	Blocked      *model.Blocked      `json:"blocked,omitempty"`
	Cancellation *model.Cancellation `json:"cancellation,omitempty"`
	Completed    string              `json:"completed,omitempty"`
	Path         string              `json:"path"`
	DocPath      string              `json:"doc_path,omitempty"`
	Doc          string              `json:"doc,omitempty"`
	TaskCounts   map[string]int      `json:"task_counts"`
}

func taskCounts(doc *model.Document) map[string]int {
	counts := map[string]int{"total": len(doc.Tasks)}
	for i := range doc.Tasks {
		counts[string(doc.Tasks[i].Status)]++
	}
	return counts
}

// countableTotal returns the number of tasks that count toward completion. Every
// task is expected to end up either done or cancelled, so only cancelled ones
// are excluded: they are work that will never be finished, and leaving them in
// the denominator would cap a project below 100% forever.
func countableTotal(counts map[string]int) int {
	return counts["total"] - counts[string(model.TaskCancelled)]
}

func writeProjectShowJSON(w io.Writer, p *discover.Project) error {
	pr := p.Doc.Project
	// JSON carries the document as stored: rendering is for terminals, and a
	// consumer that wants Markdown wants the source.
	docPath, doc, _ := projectDoc(p)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(projectDetail{
		ID:           pr.ID,
		Title:        pr.Title,
		TaskIDPrefix: pr.TaskIDPrefix,
		Status:       string(pr.Status),
		Priority:     pr.Priority,
		Areas:        pr.Areas,
		Created:      pr.Created,
		Started:      pr.Started,
		Due:          pr.Due,
		Blocked:      pr.Blocked,
		Cancellation: pr.Cancellation,
		Completed:    pr.Completed,
		Path:         p.Path,
		DocPath:      docPath,
		Doc:          doc,
		TaskCounts:   taskCounts(p.Doc),
	})
}

func writeProjectShowText(w io.Writer, p *discover.Project) {
	pr := p.Doc.Project
	field(w, "ID", pr.ID)
	field(w, "Title", pr.Title)
	field(w, "Prefix", pr.TaskIDPrefix)
	field(w, "Status", string(pr.Status))
	field(w, "Priority", priorityLabel(pr.Priority))
	if len(pr.Areas) > 0 {
		field(w, "Areas", joinComma(pr.Areas))
	}
	field(w, "Created", pr.Created)
	optField(w, "Started", pr.Started)
	optField(w, "Due", pr.Due)
	optField(w, "Completed", pr.Completed)
	if pr.Blocked != nil {
		field(w, "Blocked", fmt.Sprintf("since %s: %s", pr.Blocked.Since, pr.Blocked.Reason))
	}
	if pr.Cancellation != nil {
		field(w, "Cancelled", fmt.Sprintf("%s: %s", pr.Cancellation.Date, pr.Cancellation.Reason))
	}
	field(w, "Path", p.Path)
	counts := taskCounts(p.Doc)
	cancelled := counts[string(model.TaskCancelled)]
	countable := countableTotal(counts)
	switch {
	case counts["total"] == 0:
		field(w, "Tasks", "none")
		return
	case countable == 0:
		// Every task was cancelled, so there is no ratio to report.
		field(w, "Tasks", fmt.Sprintf("none countable (%d cancelled)", cancelled))
	default:
		field(w, "Tasks", progressBar(counts[string(model.TaskDone)], countable, cancelled, 20))
	}
	for _, s := range taskStatusOrder {
		if n := counts[string(s)]; n > 0 {
			fmt.Fprintf(w, "  %-13s%d\n", string(s)+":", n)
		}
	}
}

// projectDoc locates a project's Markdown document, which by convention sits
// beside the task file under the same basename. It returns the path relative to
// the discovery root, the contents, and whether it was found.
func projectDoc(p *discover.Project) (relPath, content string, ok bool) {
	if p.Doc == nil {
		return "", "", false
	}
	name := p.Doc.Project.ID + ".md"
	data, err := os.ReadFile(filepath.Join(filepath.Dir(p.AbsPath), name))
	if err != nil {
		return "", "", false // no document is the normal case, not an error
	}
	return filepath.Join(filepath.Dir(p.Path), name), string(data), true
}

// writeProjectDoc appends the project's rendered Markdown document, so `projects
// show` reports what the project *is* alongside its state. A missing document is
// silently skipped.
func writeProjectDoc(w io.Writer, p *discover.Project) {
	rel, content, ok := projectDoc(p)
	if !ok || strings.TrimSpace(content) == "" {
		return
	}
	fmt.Fprintf(w, "\n%s\n\n%s\n", docSeparator(rel), mdrender.Render(content, useColor(w)))
}

// docSeparator draws a labeled rule so the document is clearly a different kind
// of content from the fields above it.
func docSeparator(label string) string {
	const width = 72
	prefix := "── " + label + " "
	if n := width - len([]rune(prefix)); n > 0 {
		return prefix + strings.Repeat("─", n)
	}
	return prefix
}

// taskStatusOrder lists task statuses in lifecycle order for progress display.
var taskStatusOrder = []model.TaskStatus{
	model.TaskBacklog,
	model.TaskTodo,
	model.TaskInProgress,
	model.TaskInReview,
	model.TaskDone,
	model.TaskCancelled,
}

// rootOrCWD resolves the discovery root: the --root flag takes precedence, then
// the PM_ROOT environment variable, then the current working directory.
func rootOrCWD(opts *GlobalOptions) string {
	if opts.Root != "" {
		return opts.Root
	}
	if env := os.Getenv(rootEnvVar); env != "" {
		return env
	}
	return "."
}

// ---- projects list ----

// openProjectStatuses are the non-terminal project statuses shown by `projects
// list` by default. Done and cancelled projects are history: they accumulate
// without bound and would eventually bury the work in flight.
var openProjectStatuses = []string{
	string(model.ProjectIdea),
	string(model.ProjectTodo),
	string(model.ProjectInProgress),
	string(model.ProjectInReview),
	string(model.ProjectBlocked),
}

func newProjectsListCmd(opts *GlobalOptions) *cobra.Command {
	var (
		f   projectFilter
		all bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered projects, with optional filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
			if err != nil {
				return pmerr.IO("cannot discover projects: %v", err)
			}
			// Mirrors tasks list: hide finished work unless asked, and let an
			// explicit --status say exactly what to show.
			if !all && !cmd.Flags().Changed("status") {
				f.Statuses = openProjectStatuses
			}
			ordered := filterProjects(listOrder(ws), f)
			if opts.JSON {
				return writeProjectsListJSON(cmd.OutOrStdout(), ordered)
			}
			return writeProjectsListText(cmd.OutOrStdout(), cmd.ErrOrStderr(), ordered, ws)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&all, "all", "a", false, "include done and cancelled projects")
	flags.StringSliceVarP(&f.Statuses, "status", "s", nil, "filter by project status (repeatable)")
	flags.IntSliceVar(&f.Priorities, "priority", nil, "filter by priority (repeatable)")
	flags.StringSliceVar(&f.Areas, "area", nil, "filter by area (repeatable)")
	return cmd
}

// projectFilter is a conjunction of project filters; slice fields are
// disjunctions, matching the tasks list semantics.
type projectFilter struct {
	Statuses   []string
	Priorities []int
	Areas      []string
}

func filterProjects(projects []*discover.Project, f projectFilter) []*discover.Project {
	if len(f.Statuses) == 0 && len(f.Priorities) == 0 && len(f.Areas) == 0 {
		return projects
	}
	out := make([]*discover.Project, 0, len(projects))
	for _, p := range projects {
		if f.match(&p.Doc.Project) {
			out = append(out, p)
		}
	}
	return out
}

func (f projectFilter) match(p *model.Project) bool {
	if len(f.Statuses) > 0 && !containsString(f.Statuses, string(p.Status)) {
		return false
	}
	if len(f.Priorities) > 0 && (p.Priority == nil || !containsInt(f.Priorities, *p.Priority)) {
		return false
	}
	if len(f.Areas) > 0 && !overlapString(f.Areas, p.Areas) {
		return false
	}
	return true
}

func containsString(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func containsInt(set []int, v int) bool {
	for _, n := range set {
		if n == v {
			return true
		}
	}
	return false
}

func overlapString(want, have []string) bool {
	for _, w := range want {
		if containsString(have, w) {
			return true
		}
	}
	return false
}

type projectSummary struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Status     string         `json:"status"`
	Priority   *int           `json:"priority,omitempty"`
	Areas      []string       `json:"areas,omitempty"`
	TaskCounts map[string]int `json:"task_counts"`
}

// projectStatusRank orders projects for list display: active work first, with
// in-review ahead of in-progress because it is closer to completion, then every
// other status. This mirrors the task list's status grouping.
func projectStatusRank(s model.ProjectStatus) int {
	switch s {
	case model.ProjectInReview:
		return 0
	case model.ProjectInProgress:
		return 1
	default:
		return 2
	}
}

// listOrder returns the successfully decoded projects sorted for display:
// in-review projects first, then in-progress, then all others. Within each
// group, prioritized projects come first (lowest number = highest priority),
// projects without a priority last, with ties broken by creation date (oldest
// first) and then project ID for deterministic output.
func listOrder(ws *discover.Workspace) []*discover.Project {
	out := make([]*discover.Project, 0, len(ws.Projects))
	for i := range ws.Projects {
		if ws.Projects[i].Doc != nil {
			out = append(out, &ws.Projects[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Doc.Project, out[j].Doc.Project
		if ra, rb := projectStatusRank(a.Status), projectStatusRank(b.Status); ra != rb {
			return ra < rb // active projects sort first
		}
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
			ID:         p.Doc.Project.ID,
			Title:      p.Doc.Project.Title,
			Status:     string(p.Doc.Project.Status),
			Priority:   p.Doc.Project.Priority,
			Areas:      p.Doc.Project.Areas,
			TaskCounts: taskCounts(p.Doc),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeProjectsListText(stdout, stderr io.Writer, ordered []*discover.Project, ws *discover.Workspace) error {
	tw := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	// PROGRESS is last so its multibyte bar cannot skew tabwriter's alignment
	// of the preceding columns.
	fmt.Fprintln(tw, "ID\tTITLE\tSTATUS\tPRIO\tCREATED\tPROGRESS")
	for _, p := range ordered {
		c := taskCounts(p.Doc)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Doc.Project.ID, p.Doc.Project.Title, p.Doc.Project.Status,
			priorityLabel(p.Doc.Project.Priority), dateLabel(p.Doc.Project.Created),
			miniProgress(c[string(model.TaskDone)], countableTotal(c), 10))
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
			ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
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
