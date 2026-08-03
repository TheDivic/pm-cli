package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-projects/internal/clock"
	"github.com/TheDivic/plaintext-projects/internal/discover"
	"github.com/TheDivic/plaintext-projects/internal/model"
	"github.com/TheDivic/plaintext-projects/internal/pmerr"
	"github.com/TheDivic/plaintext-projects/internal/query"
)

func newTasksCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Inspect and manage tasks",
	}
	cmd.AddCommand(newTasksListCmd(opts))
	cmd.AddCommand(newTasksShowCmd(opts))
	cmd.AddCommand(newTasksAddCmd(opts, clk))
	cmd.AddCommand(newTasksEditCmd(opts))
	cmd.AddCommand(newTasksStatusCmd(opts, clk))
	cmd.AddCommand(newTasksBlockCmd(opts, clk))
	cmd.AddCommand(newTasksUnblockCmd(opts))
	return cmd
}

func loadRefs(opts *GlobalOptions) ([]query.TaskRef, *discover.Workspace, error) {
	ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
	if err != nil {
		return nil, nil, pmerr.IO("cannot discover projects: %v", err)
	}
	return query.Flatten(ws), ws, nil
}

// openTaskStatuses are the non-terminal statuses shown by tasks list by
// default (done and cancelled are hidden unless --all or an explicit --status).
var openTaskStatuses = []string{
	string(model.TaskBacklog),
	string(model.TaskTodo),
	string(model.TaskInProgress),
	string(model.TaskInReview),
}

// ---- tasks list ----

func newTasksListCmd(opts *GlobalOptions) *cobra.Command {
	var (
		f         query.TaskFilter
		all       bool
		blocked   bool
		dueBefore string
		dueOn     string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks across projects, with optional filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			refs, _, err := loadRefs(opts)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("blocked") {
				f.Blocked = &blocked
			}
			f.HasParent = cmd.Flags().Changed("parent")
			f.DueBefore = dueBefore
			f.DueOn = dueOn
			// By default the overview hides terminal tasks; an explicit --status
			// or --all opts back in.
			if !all && !cmd.Flags().Changed("status") {
				f.Statuses = openTaskStatuses
			}
			result := query.Filter(refs, f)
			query.SortForList(result)

			if opts.JSON {
				return writeTasksJSON(cmd.OutOrStdout(), result)
			}
			writeTasksText(cmd.OutOrStdout(), result)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&all, "all", "a", false, "include done and cancelled tasks")
	flags.StringSliceVarP(&f.Projects, "project", "p", nil, "filter by project ID (repeatable)")
	flags.StringSliceVarP(&f.Statuses, "status", "s", nil, "filter by task status (repeatable)")
	flags.StringSliceVarP(&f.Tags, "tag", "g", nil, "filter by tag (repeatable)")
	flags.StringSliceVar(&f.Areas, "area", nil, "filter by project area (repeatable)")
	flags.StringVar(&f.Parent, "parent", "", "filter by parent task ID")
	flags.BoolVar(&blocked, "blocked", false, "filter by whether a task is blocked")
	flags.StringVar(&dueBefore, "due-before", "", "filter to tasks due before YYYY-MM-DD")
	flags.StringVar(&dueOn, "due-on", "", "filter to tasks due on YYYY-MM-DD")
	return cmd
}

type taskSummary struct {
	ID       string   `json:"id"`
	Project  string   `json:"project"`
	Status   string   `json:"status"`
	Priority *int     `json:"priority,omitempty"`
	Title    string   `json:"title"`
	Parent   string   `json:"parent,omitempty"`
	Due      string   `json:"due,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Blocked  bool     `json:"blocked,omitempty"`
}

func summarize(r query.TaskRef) taskSummary {
	return taskSummary{
		ID:       r.Task.ID,
		Project:  r.Project.ID,
		Status:   string(r.Task.Status),
		Priority: r.Task.Priority,
		Title:    r.Task.Title,
		Parent:   r.Task.Parent,
		Due:      r.Task.Due,
		Tags:     r.Task.Tags,
		Blocked:  r.Task.Blocked != nil,
	}
}

func writeTasksJSON(w io.Writer, refs []query.TaskRef) error {
	out := struct {
		Tasks []taskSummary `json:"tasks"`
	}{Tasks: []taskSummary{}}
	for _, r := range refs {
		out.Tasks = append(out.Tasks, summarize(r))
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTasksText(w io.Writer, refs []query.TaskRef) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tPRIO\tPROJECT\tTITLE")
	for _, r := range refs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.Task.ID, r.Task.Status, priorityLabel(r.Task.Priority), r.Project.ID, r.Task.Title)
	}
	_ = tw.Flush()
}

// ---- tasks show ----

func newTasksShowCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show every stored field of one task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, _, err := loadRefs(opts)
			if err != nil {
				return err
			}
			ref := query.Find(refs, args[0])
			if ref == nil {
				return pmerr.Usage("no task with id %q under the discovery root", args[0]).WithTask(args[0])
			}
			if opts.JSON {
				return writeTaskDetailJSON(cmd.OutOrStdout(), ref)
			}
			writeTaskDetailText(cmd.OutOrStdout(), ref)
			return nil
		},
	}
}

func writeTaskDetailJSON(w io.Writer, r *query.TaskRef) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(taskDetail(r))
}

func writeTaskDetailText(w io.Writer, r *query.TaskRef) {
	t := r.Task
	field(w, "ID", t.ID)
	field(w, "Project", r.Project.ID)
	field(w, "Title", t.Title)
	field(w, "Status", string(t.Status))
	field(w, "Priority", priorityLabel(t.Priority))
	optField(w, "Parent", t.Parent)
	field(w, "Created", t.Created)
	optField(w, "Started", t.Started)
	optField(w, "Due", t.Due)
	optField(w, "Completed", t.Completed)
	if len(t.Tags) > 0 {
		field(w, "Tags", joinComma(t.Tags))
	}
	if t.Blocked != nil {
		field(w, "Blocked", fmt.Sprintf("since %s: %s", t.Blocked.Since, t.Blocked.Reason))
		if len(t.Blocked.Tasks) > 0 {
			field(w, "  by", joinComma(t.Blocked.Tasks))
		}
	}
	if t.Cancellation != nil {
		field(w, "Cancelled", fmt.Sprintf("%s: %s", t.Cancellation.Date, t.Cancellation.Reason))
	}
	if t.Description != "" {
		fmt.Fprintf(w, "\n%s\n", t.Description)
	}
}

// taskDetailJSON is the full machine-readable task record.
type taskDetailJSON struct {
	ID           string              `json:"id"`
	Project      string              `json:"project"`
	Path         string              `json:"path"`
	Title        string              `json:"title"`
	Description  string              `json:"description,omitempty"`
	Status       string              `json:"status"`
	Priority     *int                `json:"priority,omitempty"`
	Parent       string              `json:"parent,omitempty"`
	Created      string              `json:"created"`
	Started      string              `json:"started,omitempty"`
	Due          string              `json:"due,omitempty"`
	Tags         []string            `json:"tags,omitempty"`
	Blocked      *model.Blocked      `json:"blocked,omitempty"`
	Cancellation *model.Cancellation `json:"cancellation,omitempty"`
	Completed    string              `json:"completed,omitempty"`
}

func taskDetail(r *query.TaskRef) taskDetailJSON {
	t := r.Task
	return taskDetailJSON{
		ID:           t.ID,
		Project:      r.Project.ID,
		Path:         r.Path,
		Title:        t.Title,
		Description:  t.Description,
		Status:       string(t.Status),
		Priority:     t.Priority,
		Parent:       t.Parent,
		Created:      t.Created,
		Started:      t.Started,
		Due:          t.Due,
		Tags:         t.Tags,
		Blocked:      t.Blocked,
		Cancellation: t.Cancellation,
		Completed:    t.Completed,
	}
}
