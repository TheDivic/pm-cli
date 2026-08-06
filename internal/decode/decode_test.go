package decode

import (
	"strings"
	"testing"

	"github.com/TheDivic/pm-cli/internal/model"
)

const minimal = `schema-version: 1

project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: in-progress
  priority: 2
  areas:
    - knowledge-work
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: First task
    description: |
      A multiline
      description.
    status: todo
    created: "2026-07-31"
    tags:
      - research
`

func TestDecodeMinimal(t *testing.T) {
	doc, err := Decode([]byte(minimal))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if doc.Project.ID != "demo" || doc.Project.Status != model.ProjectInProgress {
		t.Fatalf("unexpected project: %+v", doc.Project)
	}
	if doc.Project.Priority == nil || *doc.Project.Priority != 2 {
		t.Fatalf("priority not decoded: %+v", doc.Project.Priority)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].ID != "dm-001" {
		t.Fatalf("unexpected tasks: %+v", doc.Tasks)
	}
	if !strings.Contains(doc.Tasks[0].Description, "multiline") {
		t.Fatalf("description not decoded: %q", doc.Tasks[0].Description)
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	src := strings.Replace(minimal, "  status: in-progress\n", "  status: in-progress\n  bogus: 1\n", 1)
	if _, err := Decode([]byte(src)); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestDecodeRequiresTopLevelFields(t *testing.T) {
	cases := map[string]string{
		"no schema":  "project:\n  id: demo\ntasks: []\n",
		"no project": "schema-version: 1\ntasks: []\n",
		"no tasks":   "schema-version: 1\nproject:\n  id: demo\n",
	}
	for name, src := range cases {
		if _, err := Decode([]byte(src)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDecodeRejectsWrongPriorityType(t *testing.T) {
	src := strings.Replace(minimal, "  priority: 2\n", "  priority: high\n", 1)
	if _, err := Decode([]byte(src)); err == nil {
		t.Fatal("expected type error for non-integer priority")
	}
}
