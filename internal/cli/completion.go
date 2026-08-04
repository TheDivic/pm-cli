package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TheDivic/plaintext-projects/internal/discover"
	"github.com/TheDivic/plaintext-projects/internal/model"
)

// Shell completion resolves IDs, statuses, and tags from the live discovery
// root, so suggestions always reflect the files in front of the user rather than
// a hardcoded list.
//
// Every completer degrades to "no suggestions" on error. A completion request is
// a keystroke, not a command: a broken task file must not print diagnostics into
// the user's prompt or make the shell beep.

// completionFunc is the signature cobra expects for dynamic completion.
type completionFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

// noFileComp is the directive shared by every completer here. None of these
// arguments are paths, so falling back to filename completion would only add
// noise.
const noFileComp = cobra.ShellCompDirectiveNoFileComp

// completeProjectIDs suggests discovered project IDs, described by their titles.
func completeProjectIDs(opts *GlobalOptions) completionFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
		if err != nil {
			return nil, noFileComp
		}
		var out []string
		for i := range ws.Projects {
			p := &ws.Projects[i]
			if p.Doc == nil || !strings.HasPrefix(p.Doc.Project.ID, toComplete) {
				continue
			}
			out = append(out, p.Doc.Project.ID+"\t"+p.Doc.Project.Title)
		}
		sort.Strings(out)
		return out, noFileComp
	}
}

// completeTaskIDs suggests task IDs, described by their status and title so the
// right one is recognizable without running `tasks show` first. Tasks already
// named on the command line drop out, which matters for the batch commands.
func completeTaskIDs(opts *GlobalOptions) completionFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
		if err != nil {
			return nil, noFileComp
		}
		used := make(map[string]bool, len(args))
		for _, a := range args {
			used[a] = true
		}
		var out []string
		for i := range ws.Projects {
			p := &ws.Projects[i]
			if p.Doc == nil {
				continue
			}
			for j := range p.Doc.Tasks {
				t := &p.Doc.Tasks[j]
				if used[t.ID] || !strings.HasPrefix(t.ID, toComplete) {
					continue
				}
				out = append(out, t.ID+"\t["+string(t.Status)+"] "+t.Title)
			}
		}
		sort.Strings(out)
		return out, noFileComp
	}
}

// completeTags suggests the tag vocabulary already in use, so completion nudges
// toward reusing a tag rather than inventing a near-duplicate.
func completeTags(opts *GlobalOptions) completionFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		refs, _, err := loadRefs(opts)
		if err != nil {
			return nil, noFileComp
		}
		seen := map[string]bool{}
		var out []string
		for _, r := range refs {
			for _, tag := range r.Task.Tags {
				if !seen[tag] && strings.HasPrefix(tag, toComplete) {
					seen[tag] = true
					out = append(out, tag)
				}
			}
		}
		sort.Strings(out)
		return out, noFileComp
	}
}

// completeAreas suggests the area slugs projects already reference.
func completeAreas(opts *GlobalOptions) completionFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
		if err != nil {
			return nil, noFileComp
		}
		seen := map[string]bool{}
		var out []string
		for i := range ws.Projects {
			if ws.Projects[i].Doc == nil {
				continue
			}
			for _, a := range ws.Projects[i].Doc.Project.Areas {
				if !seen[a] && strings.HasPrefix(a, toComplete) {
					seen[a] = true
					out = append(out, a)
				}
			}
		}
		sort.Strings(out)
		return out, noFileComp
	}
}

// completeTaskStatuses and completeProjectStatuses suggest the fixed lifecycle
// vocabularies.
func completeTaskStatuses(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return prefixed(taskStatusNames(), toComplete), noFileComp
}

func completeProjectStatuses(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return prefixed(projectStatusNames(), toComplete), noFileComp
}

// completeTaskIDsThenStatus drives `tasks status <task-id>... <status>`, where
// the final argument is a status and every earlier one is a task ID. Both are
// offered from the second position on, because only the user knows whether the
// ID list is finished.
func completeTaskIDsThenStatus(opts *GlobalOptions) completionFunc {
	ids := completeTaskIDs(opts)
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		out, _ := ids(cmd, args, toComplete)
		if len(args) > 0 {
			out = append(out, prefixed(taskStatusNames(), toComplete)...)
		}
		return out, noFileComp
	}
}

// completeProjectIDThenStatus drives `projects status <project-id> <status>`.
func completeProjectIDThenStatus(opts *GlobalOptions) completionFunc {
	ids := completeProjectIDs(opts)
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return ids(cmd, args, toComplete)
		}
		if len(args) == 1 {
			return prefixed(projectStatusNames(), toComplete), noFileComp
		}
		return nil, noFileComp
	}
}

func taskStatusNames() []string {
	out := make([]string, 0, len(model.TaskStatuses))
	for _, s := range model.TaskStatuses {
		out = append(out, string(s))
	}
	return out
}

func projectStatusNames() []string {
	out := make([]string, 0, len(model.ProjectStatuses))
	for _, s := range model.ProjectStatuses {
		out = append(out, string(s))
	}
	return out
}

func prefixed(values []string, toComplete string) []string {
	var out []string
	for _, v := range values {
		if strings.HasPrefix(v, toComplete) {
			out = append(out, v)
		}
	}
	return out
}

// registerFlagCompletion wires a completer to a flag, ignoring the error cobra
// returns only when the flag does not exist — a wiring bug that the command
// tests catch, not a runtime condition.
func registerFlagCompletion(cmd *cobra.Command, flag string, fn completionFunc) {
	_ = cmd.RegisterFlagCompletionFunc(flag, fn)
}

// completeCommandTree attaches every completer to the built command tree. Doing
// it in one place keeps the command constructors readable and makes it obvious
// which arguments have completion and which do not.
func completeCommandTree(root *cobra.Command, opts *GlobalOptions) {
	projectIDs := completeProjectIDs(opts)
	taskIDs := completeTaskIDs(opts)
	tags := completeTags(opts)
	areas := completeAreas(opts)

	// Global flags.
	registerFlagCompletion(root, "root", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	})

	if projects, _, err := root.Find([]string{"projects"}); err == nil {
		for _, name := range []string{"show", "edit", "validate", "format"} {
			if c, _, ferr := projects.Find([]string{name}); ferr == nil {
				c.ValidArgsFunction = projectIDs
			}
		}
		if c, _, ferr := projects.Find([]string{"status"}); ferr == nil {
			c.ValidArgsFunction = completeProjectIDThenStatus(opts)
		}
		if c, _, ferr := projects.Find([]string{"list"}); ferr == nil {
			registerFlagCompletion(c, "status", completeProjectStatuses)
			registerFlagCompletion(c, "area", areas)
		}
		for _, name := range []string{"create", "edit"} {
			if c, _, ferr := projects.Find([]string{name}); ferr == nil {
				registerFlagCompletion(c, "status", completeProjectStatuses)
				registerFlagCompletion(c, "area", areas)
				registerFlagCompletion(c, "add-area", areas)
				registerFlagCompletion(c, "remove-area", areas)
			}
		}
	}

	if tasks, _, err := root.Find([]string{"tasks"}); err == nil {
		for _, name := range []string{"show", "edit", "block", "unblock", "delete"} {
			if c, _, ferr := tasks.Find([]string{name}); ferr == nil {
				c.ValidArgsFunction = taskIDs
			}
		}
		if c, _, ferr := tasks.Find([]string{"status"}); ferr == nil {
			c.ValidArgsFunction = completeTaskIDsThenStatus(opts)
		}
		if c, _, ferr := tasks.Find([]string{"list"}); ferr == nil {
			registerFlagCompletion(c, "project", projectIDs)
			registerFlagCompletion(c, "status", completeTaskStatuses)
			registerFlagCompletion(c, "tag", tags)
			registerFlagCompletion(c, "area", areas)
			registerFlagCompletion(c, "parent", taskIDs)
		}
		if c, _, ferr := tasks.Find([]string{"add"}); ferr == nil {
			registerFlagCompletion(c, "project", projectIDs)
			registerFlagCompletion(c, "status", completeTaskStatuses)
			registerFlagCompletion(c, "tag", tags)
			registerFlagCompletion(c, "parent", taskIDs)
		}
		if c, _, ferr := tasks.Find([]string{"edit"}); ferr == nil {
			registerFlagCompletion(c, "add-tag", tags)
			registerFlagCompletion(c, "remove-tag", tags)
			registerFlagCompletion(c, "parent", taskIDs)
		}
		if c, _, ferr := tasks.Find([]string{"block"}); ferr == nil {
			registerFlagCompletion(c, "task", taskIDs)
		}
	}
}
