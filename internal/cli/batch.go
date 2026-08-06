package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TheDivic/pm-cli/internal/discover"
	"github.com/TheDivic/pm-cli/internal/model"
	"github.com/TheDivic/pm-cli/internal/pmerr"
)

// taskTarget is one resolved task ID and the absolute path of the file holding
// it.
type taskTarget struct {
	ID   string
	Path string
}

// taskResult identifies one task a batch mutation actually changed.
type taskResult struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// batchResult is the machine-readable payload for a mutation that touched more
// than one task. A single-task mutation keeps the flat mutationResult shape, so
// existing agents and scripts are unaffected by batch support.
type batchResult struct {
	Kind   string       `json:"kind"`
	Status string       `json:"status,omitempty"`
	Count  int          `json:"count"`
	Tasks  []taskResult `json:"tasks"`
}

// resolveTaskBatch resolves every task ID in one discovery walk. Resolving up
// front means an unknown ID fails the command before anything is written,
// instead of after the earlier tasks have already changed. Repeated IDs collapse
// to one entry, preserving first-seen order.
func resolveTaskBatch(opts *GlobalOptions, ids []string) ([]taskTarget, error) {
	ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
	if err != nil {
		return nil, pmerr.IO("cannot discover projects: %v", err)
	}
	byID := taskPaths(ws)

	seen := make(map[string]bool, len(ids))
	out := make([]taskTarget, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		path, ok := byID[id]
		if !ok {
			return nil, pmerr.Usage("no task with id %q under the discovery root", id).WithTask(id)
		}
		out = append(out, taskTarget{ID: id, Path: path})
	}
	return out, nil
}

// taskPaths maps every discovered task ID to its file. Task IDs are unique
// across a discovery root, so the mapping is unambiguous.
func taskPaths(ws *discover.Workspace) map[string]string {
	out := map[string]string{}
	for i := range ws.Projects {
		p := &ws.Projects[i]
		if p.Doc == nil {
			continue
		}
		for j := range p.Doc.Tasks {
			out[p.Doc.Tasks[j].ID] = p.AbsPath
		}
	}
	return out
}

// applyToTaskFiles runs apply once per file, handing it every targeted ID in
// that file, so each file passes through the mutation envelope (lock, validate,
// write) exactly once. apply returns the IDs it actually changed, which may
// exceed the requested ones — a cascading delete also removes descendants.
//
// Within a file the change is all-or-nothing: one failing task leaves that whole
// file untouched. Across files it is not, because each file is its own atomic
// write. When a later file fails, the tasks already committed are named on
// stderr rather than left for the caller to work out.
func applyToTaskFiles(stderr io.Writer, targets []taskTarget, apply func(*model.Document, []string) ([]string, error)) ([]taskResult, error) {
	var order []string
	byPath := map[string][]string{}
	for _, t := range targets {
		if _, ok := byPath[t.Path]; !ok {
			order = append(order, t.Path)
		}
		byPath[t.Path] = append(byPath[t.Path], t.ID)
	}

	done := make([]taskResult, 0, len(targets))
	for _, path := range order {
		var changed []string
		err := runMutation(stderr, path, func(d *model.Document) error {
			var aerr error
			changed, aerr = apply(d, byPath[path])
			return aerr
		})
		if err != nil {
			if len(done) > 0 {
				fmt.Fprintf(stderr, "note: already updated %s\n", joinComma(resultIDs(done)))
			}
			return done, err
		}
		for _, id := range changed {
			done = append(done, taskResult{ID: id, Path: path})
		}
	}
	return done, nil
}

// applyToTasks runs a per-task change over every target, one file at a time.
func applyToTasks(stderr io.Writer, targets []taskTarget, apply func(*model.Document, string) error) ([]taskResult, error) {
	batch := len(targets) > 1
	return applyToTaskFiles(stderr, targets, func(d *model.Document, ids []string) ([]string, error) {
		for _, id := range ids {
			if err := apply(d, id); err != nil {
				if batch {
					// Name the offending task; the command covers several.
					return nil, fmt.Errorf("%s: %w", id, err)
				}
				return nil, err
			}
		}
		return ids, nil
	})
}

func resultIDs(rs []taskResult) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.ID)
	}
	return out
}

// reportTaskMutations renders the outcome of a possibly batched task mutation.
// One task keeps the single-result shape; several emit one human line each and a
// batch object in JSON mode.
func reportTaskMutations(w io.Writer, jsonMode bool, kind, status string, results []taskResult) error {
	if len(results) == 1 {
		return reportMutation(w, jsonMode, mutationResult{
			Kind: kind, ID: results[0].ID, Status: status, Path: results[0].Path,
		})
	}
	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(batchResult{Kind: kind, Status: status, Count: len(results), Tasks: results})
	}
	for _, r := range results {
		writeMutationLine(w, mutationResult{Kind: kind, ID: r.ID, Status: status, Path: r.Path})
	}
	return nil
}

// runTaskBatch is the whole batch path for a task mutation command: resolve the
// IDs, apply the change file by file, and report the result.
func runTaskBatch(opts *GlobalOptions, stdout, stderr io.Writer, ids []string, kind, status string,
	apply func(*model.Document, string) error) error {
	targets, err := resolveTaskBatch(opts, ids)
	if err != nil {
		return err
	}
	results, err := applyToTasks(stderr, targets, apply)
	if err != nil {
		return err
	}
	return reportTaskMutations(stdout, opts.JSON, kind, status, results)
}
