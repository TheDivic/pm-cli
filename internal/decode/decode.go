// Package decode turns validated task-file bytes into the typed model. It runs
// the restricted-profile check first, then decodes with unknown-field
// rejection. Structural presence of the three top-level fields is required
// here; field-level semantics are the validator's job.
package decode

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/TheDivic/pm-cli/internal/model"
	"github.com/TheDivic/pm-cli/internal/yamlprofile"
)

// documentYAML mirrors the on-disk schema for decoding. Pointers on the three
// top-level fields let us tell "absent" from a zero value.
type documentYAML struct {
	SchemaVersion *int         `yaml:"schema-version"`
	Project       *projectYAML `yaml:"project"`
	Tasks         *[]taskYAML  `yaml:"tasks"`
}

type projectYAML struct {
	ID           string            `yaml:"id"`
	Title        string            `yaml:"title"`
	TaskIDPrefix string            `yaml:"task-id-prefix"`
	Status       string            `yaml:"status"`
	Priority     *int              `yaml:"priority"`
	Areas        []string          `yaml:"areas"`
	Created      string            `yaml:"created"`
	Started      string            `yaml:"started"`
	Due          string            `yaml:"due"`
	Blocked      *blockedYAML      `yaml:"blocked"`
	Cancellation *cancellationYAML `yaml:"cancellation"`
	Completed    string            `yaml:"completed"`
}

type taskYAML struct {
	ID           string            `yaml:"id"`
	Title        string            `yaml:"title"`
	Description  string            `yaml:"description"`
	Status       string            `yaml:"status"`
	Priority     *int              `yaml:"priority"`
	Parent       string            `yaml:"parent"`
	Created      string            `yaml:"created"`
	Started      string            `yaml:"started"`
	Due          string            `yaml:"due"`
	Tags         []string          `yaml:"tags"`
	Blocked      *blockedYAML      `yaml:"blocked"`
	Cancellation *cancellationYAML `yaml:"cancellation"`
	Completed    string            `yaml:"completed"`
}

type blockedYAML struct {
	Reason string   `yaml:"reason"`
	Tasks  []string `yaml:"tasks"`
	Since  string   `yaml:"since"`
}

type cancellationYAML struct {
	Reason string `yaml:"reason"`
	Date   string `yaml:"date"`
}

// Decode enforces the restricted profile, decodes into the typed model with
// unknown-field rejection, and requires the three top-level fields to be
// present. Field-level validation happens separately.
func Decode(data []byte) (*model.Document, error) {
	if _, err := yamlprofile.Load(data); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var raw documentYAML
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid task document: %w", err)
	}

	if raw.SchemaVersion == nil {
		return nil, fmt.Errorf("missing required field: schema-version")
	}
	if raw.Project == nil {
		return nil, fmt.Errorf("missing required field: project")
	}
	if raw.Tasks == nil {
		return nil, fmt.Errorf("missing required field: tasks (use an empty list)")
	}

	return convert(&raw), nil
}

func convert(raw *documentYAML) *model.Document {
	doc := &model.Document{
		SchemaVersion: *raw.SchemaVersion,
		Project:       convertProject(raw.Project),
		Tasks:         make([]model.Task, 0, len(*raw.Tasks)),
	}
	for i := range *raw.Tasks {
		doc.Tasks = append(doc.Tasks, convertTask(&(*raw.Tasks)[i]))
	}
	return doc
}

func convertProject(p *projectYAML) model.Project {
	return model.Project{
		ID:           p.ID,
		Title:        p.Title,
		TaskIDPrefix: p.TaskIDPrefix,
		Status:       model.ProjectStatus(p.Status),
		Priority:     p.Priority,
		Areas:        p.Areas,
		Created:      p.Created,
		Started:      p.Started,
		Due:          p.Due,
		Blocked:      convertBlocked(p.Blocked),
		Cancellation: convertCancellation(p.Cancellation),
		Completed:    p.Completed,
	}
}

func convertTask(t *taskYAML) model.Task {
	return model.Task{
		ID:           t.ID,
		Title:        t.Title,
		Description:  t.Description,
		Status:       model.TaskStatus(t.Status),
		Priority:     t.Priority,
		Parent:       t.Parent,
		Created:      t.Created,
		Started:      t.Started,
		Due:          t.Due,
		Tags:         t.Tags,
		Blocked:      convertBlocked(t.Blocked),
		Cancellation: convertCancellation(t.Cancellation),
		Completed:    t.Completed,
	}
}

func convertBlocked(b *blockedYAML) *model.Blocked {
	if b == nil {
		return nil
	}
	return &model.Blocked{Reason: b.Reason, Tasks: b.Tasks, Since: b.Since}
}

func convertCancellation(c *cancellationYAML) *model.Cancellation {
	if c == nil {
		return nil
	}
	return &model.Cancellation{Reason: c.Reason, Date: c.Date}
}
