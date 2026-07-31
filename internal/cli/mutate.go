package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/TheDivic/plaintext-tasks/internal/decode"
	"github.com/TheDivic/plaintext-tasks/internal/discover"
	"github.com/TheDivic/plaintext-tasks/internal/emit"
	"github.com/TheDivic/plaintext-tasks/internal/fsatomic"
	"github.com/TheDivic/plaintext-tasks/internal/lockfile"
	"github.com/TheDivic/plaintext-tasks/internal/model"
	"github.com/TheDivic/plaintext-tasks/internal/pmerr"
	"github.com/TheDivic/plaintext-tasks/internal/validate"
)

// runMutation performs the full mutation-safety sequence on path: lock, read,
// validate, apply one change, revalidate, render canonical bytes, and write
// atomically. Any error before the write leaves the original bytes unchanged.
func runMutation(stderr io.Writer, path string, apply func(*model.Document) error) error {
	release, err := lockfile.Acquire(path)
	if err != nil {
		return pmerr.IO("cannot lock %s: %v", path, err)
	}
	defer func() { _ = release() }()

	data, err := os.ReadFile(path)
	if err != nil {
		return pmerr.IO("cannot read %s: %v", path, err)
	}
	doc, err := decode.Decode(data)
	if err != nil {
		return pmerr.Validation("%v", err).WithFile(path)
	}
	if f := validate.Document(doc); len(f) > 0 {
		return reportFindings(stderr, path, f)
	}
	if err := apply(doc); err != nil {
		return pmerr.Validation("%v", err).WithFile(path).WithProject(doc.Project.ID)
	}
	if f := validate.Document(doc); len(f) > 0 {
		return reportFindings(stderr, path, f)
	}
	if err := fsatomic.WriteFile(path, emit.Document(doc), 0o644); err != nil {
		return pmerr.IO("cannot write %s: %v", path, err)
	}
	return nil
}

func reportFindings(stderr io.Writer, path string, findings []validate.Finding) error {
	fmt.Fprintf(stderr, "%s:\n", path)
	for _, f := range findings {
		fmt.Fprintf(stderr, "  - %s\n", f.String())
	}
	return pmerr.Validation("%d validation problem(s)", len(findings)).WithFile(path)
}

// resolveProject finds a discovered project by ID and returns its file path.
func resolveProject(opts *GlobalOptions, id string) (string, error) {
	ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
	if err != nil {
		return "", pmerr.IO("cannot discover projects: %v", err)
	}
	for i := range ws.Projects {
		if ws.Projects[i].ID() == id {
			return ws.Projects[i].AbsPath, nil
		}
	}
	return "", pmerr.Usage("no project with id %q under the discovery root", id).WithProject(id)
}

// resolveTask finds the project file that contains the given task ID.
func resolveTask(opts *GlobalOptions, taskID string) (path, projectID string, err error) {
	ws, err := discover.Discover(rootOrCWD(opts), opts.NoIgnore)
	if err != nil {
		return "", "", pmerr.IO("cannot discover projects: %v", err)
	}
	for i := range ws.Projects {
		p := &ws.Projects[i]
		if p.Doc == nil {
			continue
		}
		for j := range p.Doc.Tasks {
			if p.Doc.Tasks[j].ID == taskID {
				return p.AbsPath, p.Doc.Project.ID, nil
			}
		}
	}
	return "", "", pmerr.Usage("no task with id %q under the discovery root", taskID).WithTask(taskID)
}

// mutationResult is the success payload reported by a mutation command.
type mutationResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Status  string `json:"status,omitempty"`
	Project string `json:"project,omitempty"`
	Path    string `json:"path"`
}

func reportMutation(w io.Writer, jsonMode bool, r mutationResult) error {
	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	msg := fmt.Sprintf("%s %s", r.Kind, r.ID)
	if r.Status != "" {
		msg += fmt.Sprintf(" (%s)", r.Status)
	}
	fmt.Fprintf(w, "%s -> %s\n", msg, r.Path)
	return nil
}

// strp returns a pointer to s for optional edit fields.
func strp(s string) *string { return &s }
