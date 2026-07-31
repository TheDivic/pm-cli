package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-tasks/internal/clock"
	"github.com/TheDivic/plaintext-tasks/internal/discover"
	"github.com/TheDivic/plaintext-tasks/internal/emit"
	"github.com/TheDivic/plaintext-tasks/internal/fsatomic"
	"github.com/TheDivic/plaintext-tasks/internal/lockfile"
	"github.com/TheDivic/plaintext-tasks/internal/model"
	"github.com/TheDivic/plaintext-tasks/internal/mutate"
	"github.com/TheDivic/plaintext-tasks/internal/pmerr"
	"github.com/TheDivic/plaintext-tasks/internal/validate"
)

// ---- projects create ----

func newProjectsCreateCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var (
		id, title, prefix, path, status, due string
		priority                             int
		areas                                []string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project task file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj := model.Project{
				ID: id, Title: title, TaskIDPrefix: prefix,
				Status: model.ProjectStatus(status), Areas: areas,
				Created: clk.Today(), Due: due,
			}
			if cmd.Flags().Changed("priority") {
				proj.Priority = &priority
			}
			if proj.Status == model.ProjectInProgress {
				proj.Started = clk.Today()
			}
			doc := &model.Document{SchemaVersion: model.SchemaVersion, Project: proj, Tasks: []model.Task{}}

			target := path
			if target == "" {
				target = filepath.Join(rootOrCWD(opts), id, id+".tasks.yaml")
			}
			return createProject(cmd, opts, target, doc)
		},
	}
	f := cmd.Flags()
	f.StringVar(&id, "id", "", "project ID (required)")
	f.StringVar(&title, "title", "", "project title (required)")
	f.StringVar(&prefix, "task-id-prefix", "", "task-ID prefix (required)")
	f.StringVar(&path, "path", "", "task-file path (default <root>/<id>/<id>.tasks.yaml)")
	f.StringVar(&status, "status", "idea", "initial project status")
	f.IntVar(&priority, "priority", 0, "project priority")
	f.StringVar(&due, "due", "", "due date (YYYY-MM-DD)")
	f.StringSliceVar(&areas, "area", nil, "area slug (repeatable)")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("task-id-prefix")
	return cmd
}

func createProject(cmd *cobra.Command, opts *GlobalOptions, target string, doc *model.Document) error {
	stem := strings.TrimSuffix(filepath.Base(target), ".tasks.yaml")
	if stem != doc.Project.ID {
		return pmerr.Usage("task filename stem %q must match project id %q", stem, doc.Project.ID)
	}

	// Enforce project-ID and prefix uniqueness across the discovery root.
	ws, err := discover.Discover(rootOrCWD(opts))
	if err != nil {
		return pmerr.IO("cannot discover projects: %v", err)
	}
	for i := range ws.Projects {
		p := ws.Projects[i]
		if p.Doc == nil {
			continue
		}
		if p.Doc.Project.ID == doc.Project.ID {
			return pmerr.Validation("project id %q already exists at %s", doc.Project.ID, p.Path).WithProject(doc.Project.ID)
		}
		if p.Doc.Project.TaskIDPrefix == doc.Project.TaskIDPrefix {
			return pmerr.Validation("task-id-prefix %q already used by %s", doc.Project.TaskIDPrefix, p.Path).WithProject(doc.Project.ID)
		}
	}

	if f := validate.Document(doc); len(f) > 0 {
		return reportFindings(cmd.ErrOrStderr(), target, f)
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return pmerr.IO("bad path %s: %v", target, err)
	}
	release, err := lockfile.Acquire(abs)
	if err != nil {
		return pmerr.IO("cannot lock %s: %v", abs, err)
	}
	defer func() { _ = release() }()

	if _, err := os.Stat(abs); err == nil {
		return pmerr.Validation("refusing to replace existing file %s", target).WithFile(target)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return pmerr.IO("cannot create directory: %v", err)
	}
	if err := fsatomic.WriteFile(abs, emit.Document(doc), 0o644); err != nil {
		return pmerr.IO("cannot write %s: %v", target, err)
	}
	return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
		Kind: "created project", ID: doc.Project.ID, Status: string(doc.Project.Status), Path: target,
	})
}

// ---- projects edit ----

func newProjectsEditCmd(opts *GlobalOptions) *cobra.Command {
	var (
		e                       mutate.ProjectEdit
		title, due              string
		priority                int
		clearPriority, clearDue bool
	)
	cmd := &cobra.Command{
		Use:   "edit <project-id>",
		Short: "Edit a project's title, priority, due date, and areas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveProject(opts, args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("title") {
				e.Title = strp(title)
			}
			if cmd.Flags().Changed("priority") {
				e.Priority = &priority
			}
			e.ClearPriority = clearPriority
			if cmd.Flags().Changed("due") {
				e.Due = strp(due)
			}
			e.ClearDue = clearDue

			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.EditProject(d, e)
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "updated project", ID: args[0], Path: path,
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&title, "title", "", "new title")
	f.IntVar(&priority, "priority", 0, "set priority")
	f.BoolVar(&clearPriority, "clear-priority", false, "remove the priority")
	f.StringVar(&due, "due", "", "set due date (YYYY-MM-DD)")
	f.BoolVar(&clearDue, "clear-due", false, "remove the due date")
	f.StringSliceVar(&e.AddAreas, "add-area", nil, "add an area (repeatable)")
	f.StringSliceVar(&e.RemoveAreas, "remove-area", nil, "remove an area (repeatable)")
	return cmd
}

// ---- projects status ----

func newProjectsStatusCmd(opts *GlobalOptions, clk clock.Clock) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "status <project-id> <status>",
		Short: "Change a project's lifecycle status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveProject(opts, args[0])
			if err != nil {
				return err
			}
			status := model.ProjectStatus(args[1])
			if err := runMutation(cmd.ErrOrStderr(), path, func(d *model.Document) error {
				return mutate.ProjectStatus(d, status, reason, clk)
			}); err != nil {
				return err
			}
			return reportMutation(cmd.OutOrStdout(), opts.JSON, mutationResult{
				Kind: "project", ID: args[0], Status: args[1], Path: path,
			})
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "reason (required for blocked and cancelled)")
	return cmd
}
