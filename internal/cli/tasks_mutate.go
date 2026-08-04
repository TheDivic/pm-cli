package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-projects/internal/clock"
	"github.com/TheDivic/plaintext-projects/internal/model"
	"github.com/TheDivic/plaintext-projects/internal/mutate"
	"github.com/TheDivic/plaintext-projects/internal/pmerr"
)

// readDescription resolves --description-file: a path, or "-" for standard
// input. It returns provided=false when the flag was not set.
func readDescription(cmd *cobra.Command, file string) (text string, provided bool, err error) {
	if !cmd.Flags().Changed("description-file") {
		return "", false, nil
	}
	if file == "-" {
		b, rerr := io.ReadAll(cmd.InOrStdin())
		if rerr != nil {
			return "", true, pmerr.IO("cannot read description from stdin: %v", rerr)
		}
		return string(b), true, nil
	}
	b, rerr := os.ReadFile(file)
	if rerr != nil {
		return "", true, pmerr.IO("cannot read description file %s: %v", file, rerr)
	}
	return string(b), true, nil
}

// ---- tasks add ----

func newTasksAddCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var (
		project, title, descFile, status, parent, due string
		priority                                      int
		tags                                          []string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a task to a project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveProject(opts, project)
			if err != nil {
				return err
			}
			desc, _, err := readDescription(cmd, descFile)
			if err != nil {
				return err
			}
			add := mutate.TaskAdd{
				Title: title, Description: desc, Status: model.TaskStatus(status),
				Parent: parent, Due: due, Tags: tags,
			}
			if cmd.Flags().Changed("priority") {
				add.Priority = &priority
			}
			var newID string
			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				id, aerr := mutate.AddTask(d, add, clk)
				newID = id
				return aerr
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "added task", ID: newID, Project: project, Path: path,
			})
		},
	}
	f := cmd.Flags()
	f.StringVarP(&project, "project", "p", "", "project ID (required)")
	f.StringVarP(&title, "title", "t", "", "task title (required)")
	f.StringVar(&descFile, "description-file", "", "read description from a file, or - for stdin")
	f.StringVarP(&status, "status", "s", "backlog", "initial task status")
	f.IntVar(&priority, "priority", 0, "task priority")
	f.StringVar(&parent, "parent", "", "parent task ID")
	f.StringVar(&due, "due", "", "due date (YYYY-MM-DD)")
	f.StringSliceVarP(&tags, "tag", "g", nil, "tag (repeatable)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

// ---- tasks edit ----

func newTasksEditCmd(opts *GlobalOptions) *cobra.Command {
	var (
		e                                    mutate.TaskEdit
		title, descFile, due, parent         string
		priority                             int
		clearPriority, clearDue, clearParent bool
	)
	cmd := &cobra.Command{
		Use:   "edit <task-id>...",
		Short: "Edit the title, description, priority, due date, tags, and parent of one or more tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("title") {
				// A title names one specific outcome, so applying one to several
				// tasks is a mistake rather than a bulk edit.
				if len(args) > 1 {
					return pmerr.Usage("--title applies to a single task; got %d task IDs", len(args))
				}
				e.Title = strp(title)
			}
			if desc, provided, derr := readDescription(cmd, descFile); derr != nil {
				return derr
			} else if provided {
				e.Description = strp(desc)
			}
			if cmd.Flags().Changed("priority") {
				e.Priority = &priority
			}
			e.ClearPriority = clearPriority
			if cmd.Flags().Changed("due") {
				e.Due = strp(due)
			}
			e.ClearDue = clearDue
			if cmd.Flags().Changed("parent") {
				e.Parent = strp(parent)
			}
			e.ClearParent = clearParent

			return runTaskBatch(opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), args, "updated task", "",
				func(d *model.Document, id string) error {
					return mutate.EditTask(d, id, e)
				})
		},
	}
	f := cmd.Flags()
	f.StringVarP(&title, "title", "t", "", "new title")
	f.StringVar(&descFile, "description-file", "", "read description from a file, or - for stdin")
	f.IntVar(&priority, "priority", 0, "set priority")
	f.BoolVar(&clearPriority, "clear-priority", false, "remove the priority")
	f.StringVar(&due, "due", "", "set due date (YYYY-MM-DD)")
	f.BoolVar(&clearDue, "clear-due", false, "remove the due date")
	f.StringSliceVar(&e.AddTags, "add-tag", nil, "add a tag (repeatable)")
	f.StringSliceVar(&e.RemoveTags, "remove-tag", nil, "remove a tag (repeatable)")
	f.StringVar(&parent, "parent", "", "set parent task ID")
	f.BoolVar(&clearParent, "clear-parent", false, "remove the parent")
	return cmd
}

// ---- tasks delete ----

func newTasksDeleteCmd(opts *GlobalOptions) *cobra.Command {
	var cascade bool
	cmd := &cobra.Command{
		Use:   "delete <task-id>...",
		Short: "Delete one or more tasks",
		Long: "Delete one or more tasks.\n\n" +
			"Deleting removes the record and its history from the file; Git is the only\n" +
			"way back. To retire work while keeping the outcome legible, cancel it\n" +
			"instead: pm tasks status <task-id> cancelled --reason \"<why>\".\n\n" +
			"A delete is refused when a task outside it points at one inside it, as a\n" +
			"child or as a blocker. --cascade removes the whole subtree and drops those\n" +
			"references from the tasks that remain.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolveTaskBatch(opts, args)
			if err != nil {
				return err
			}
			results, err := applyToTaskFiles(cmd.ErrOrStderr(), targets,
				func(d *model.Document, ids []string) ([]string, error) {
					return mutate.DeleteTasks(d, ids, cascade)
				})
			if err != nil {
				return err
			}
			return reportTaskMutations(cmd.OutOrStdout(), opts.JSON, "deleted task", "", results)
		},
	}
	cmd.Flags().BoolVar(&cascade, "cascade", false,
		"also delete descendants and drop references from the tasks that remain")
	return cmd
}

// ---- tasks status ----

func newTasksStatusCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "status <task-id>... <status>",
		Short: "Change the lifecycle status of one or more tasks",
		Long: "Change the lifecycle status of one or more tasks.\n\n" +
			"The final argument is the target status; every argument before it is a\n" +
			"task ID. All named tasks move to the same status.",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, status := args[:len(args)-1], model.TaskStatus(args[len(args)-1])
			return runTaskBatch(opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), ids, "task", string(status),
				func(d *model.Document, id string) error {
					return mutate.TaskStatus(d, id, status, reason, clk)
				})
		},
	}
	cmd.Flags().StringVarP(&reason, "reason", "r", "", "reason (required for cancelled)")
	return cmd
}

// ---- tasks block / unblock ----

func newTasksBlockCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var (
		reason   string
		blockers []string
	)
	cmd := &cobra.Command{
		Use:   "block <task-id>...",
		Short: "Record a blocking condition on one or more tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskBatch(opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), args, "blocked task", "",
				func(d *model.Document, id string) error {
					return mutate.BlockTask(d, id, reason, blockers, clk)
				})
		},
	}
	cmd.Flags().StringVarP(&reason, "reason", "r", "", "reason for the blockage (required)")
	cmd.Flags().StringSliceVar(&blockers, "task", nil, "blocking task ID (repeatable)")
	_ = cmd.MarkFlagRequired("reason")
	return cmd
}

func newTasksUnblockCmd(opts *GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "unblock <task-id>...",
		Short: "Remove the blocking record from one or more tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskBatch(opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), args, "unblocked task", "",
				func(d *model.Document, id string) error {
					return mutate.UnblockTask(d, id)
				})
		},
	}
}
