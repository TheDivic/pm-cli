// Package discover walks a task root, loads every *.tasks.yaml file through the
// decode and validate pipeline, and reports project-ID and task-ID-prefix
// conflicts across the discovered set.
//
// Discovery honors .gitignore rules from the root downward (unless noIgnore is
// set), always skips .git, and does not follow directory symlinks.
package discover

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TheDivic/pm-cli/internal/decode"
	"github.com/TheDivic/pm-cli/internal/ignore"
	"github.com/TheDivic/pm-cli/internal/model"
	"github.com/TheDivic/pm-cli/internal/validate"
)

// suffix identifies task files.
const suffix = ".tasks.yaml"

// Project is one discovered task file and the result of loading it.
type Project struct {
	// Path is relative to the discovery root.
	Path string
	// AbsPath is the absolute file path.
	AbsPath string
	// Doc is the decoded document, or nil when decoding failed.
	Doc *model.Document
	// LoadErr is set when the file could not be read or decoded.
	LoadErr error
	// Findings holds semantic validation problems (empty when Doc is valid).
	Findings []validate.Finding
}

// ID returns the project ID, or an empty string when the file failed to decode.
func (p *Project) ID() string {
	if p.Doc == nil {
		return ""
	}
	return p.Doc.Project.ID
}

// Workspace is the result of discovering a root.
type Workspace struct {
	Root string
	// Projects are sorted by project ID (files that failed to decode sort last
	// by path).
	Projects []Project
	// Conflicts lists project-ID and task-ID-prefix collisions across files.
	Conflicts []Conflict
}

// Conflict is a duplicate project ID or task-ID prefix across discovered files.
type Conflict struct {
	Kind  string // "project id" or "task-id-prefix"
	Value string
	Paths []string
}

// Discover walks root, loads each task file, and records cross-file conflicts.
// When noIgnore is false, .gitignore rules from the root down exclude matching
// files and directories. The returned error is only for a root that cannot be
// walked; per-file problems live on each Project.
func Discover(root string, noIgnore bool) (*Workspace, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var matcher *ignore.Matcher
	if !noIgnore {
		if matcher, err = ignore.New(absRoot); err != nil {
			return nil, err
		}
	}

	ws := &Workspace{Root: absRoot}
	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(absRoot, path)
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if matcher.Ignored(rel, true) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), suffix) {
			return nil
		}
		if matcher.Ignored(rel, false) {
			return nil
		}
		ws.Projects = append(ws.Projects, load(absRoot, path))
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sortProjects(ws.Projects)
	ws.Conflicts = findConflicts(ws.Projects)
	return ws, nil
}

// load reads and decodes a single task file and runs validation.
func load(root, path string) Project {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	p := Project{Path: rel, AbsPath: path}

	data, err := os.ReadFile(path)
	if err != nil {
		p.LoadErr = err
		return p
	}
	doc, err := decode.Decode(data)
	if err != nil {
		p.LoadErr = err
		return p
	}
	p.Doc = doc
	p.Findings = validate.Document(doc)

	// The filename stem must match project.id.
	stem := strings.TrimSuffix(filepath.Base(path), suffix)
	if doc.Project.ID != "" && doc.Project.ID != stem {
		p.Findings = append(p.Findings, validate.Finding{
			Field:   "project.id",
			Message: "must match the task filename stem " + strconvQuote(stem),
		})
	}

	// Each area slug must resolve to areas/<slug>.md under the discovery root.
	for i, area := range doc.Project.Areas {
		if _, statErr := os.Stat(filepath.Join(root, "areas", area+".md")); statErr != nil {
			p.Findings = append(p.Findings, validate.Finding{
				Field:   fmt.Sprintf("project.areas[%d]", i),
				Message: fmt.Sprintf("area %q does not resolve to areas/%s.md under the root", area, area),
			})
		}
	}
	return p
}

func sortProjects(ps []Project) {
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i].ID(), ps[j].ID()
		if a == "" || b == "" {
			if a == b {
				return ps[i].Path < ps[j].Path
			}
			return a != "" // non-empty IDs sort before empty ones
		}
		if a == b {
			return ps[i].Path < ps[j].Path
		}
		return a < b
	})
}

func findConflicts(ps []Project) []Conflict {
	ids := map[string][]string{}
	prefixes := map[string][]string{}
	for i := range ps {
		if ps[i].Doc == nil {
			continue
		}
		ids[ps[i].Doc.Project.ID] = append(ids[ps[i].Doc.Project.ID], ps[i].Path)
		prefixes[ps[i].Doc.Project.TaskIDPrefix] = append(prefixes[ps[i].Doc.Project.TaskIDPrefix], ps[i].Path)
	}

	var conflicts []Conflict
	conflicts = append(conflicts, collectDupes("project id", ids)...)
	conflicts = append(conflicts, collectDupes("task-id-prefix", prefixes)...)
	return conflicts
}

func collectDupes(kind string, m map[string][]string) []Conflict {
	var out []Conflict
	for value, paths := range m {
		if value != "" && len(paths) > 1 {
			sort.Strings(paths)
			out = append(out, Conflict{Kind: kind, Value: value, Paths: paths})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out
}

func strconvQuote(s string) string { return `"` + s + `"` }
