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
		Use:   "edit <task-id>",
		Short: "Edit a task's title, description, priority, due date, tags, and parent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveTask(opts, args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("title") {
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

			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.EditTask(d, args[0], e)
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "updated task", ID: args[0], Path: path,
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

// ---- tasks status ----

func newTasksStatusCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "status <task-id> <status>",
		Short: "Change a task's lifecycle status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveTask(opts, args[0])
			if err != nil {
				return err
			}
			status := model.TaskStatus(args[1])
			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.TaskStatus(d, args[0], status, reason, clk)
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "task", ID: args[0], Status: args[1], Path: path,
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
		Use:   "block <task-id>",
		Short: "Record a blocking condition on a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveTask(opts, args[0])
			if err != nil {
				return err
			}
			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.BlockTask(d, args[0], reason, blockers, clk)
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "blocked task", ID: args[0], Path: path,
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
		Use:   "unblock <task-id>",
		Short: "Remove a task's blocking record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveTask(opts, args[0])
			if err != nil {
				return err
			}
			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.UnblockTask(d, args[0])
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "unblocked task", ID: args[0], Path: path,
			})
		},
	}
}
