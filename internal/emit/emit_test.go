package emit

import (
	"bytes"
	"testing"

	"github.com/TheDivic/plaintext-projects/internal/decode"
	"github.com/TheDivic/plaintext-projects/internal/model"
)

func ptr(n int) *int { return &n }

func sampleDoc() *model.Document {
	return &model.Document{
		SchemaVersion: 1,
		Project: model.Project{
			ID: "example-project", Title: "Example Project", TaskIDPrefix: "ex",
			Status: model.ProjectInProgress, Priority: ptr(1),
			Areas: []string{"knowledge-work"}, Created: "2026-07-31", Started: "2026-07-31",
		},
		Tasks: []model.Task{
			{
				ID: "ex-001", Title: "Define the accepted task schema",
				Description: "Record fields, lifecycle rules, and validation behavior.\n",
				Status:      model.TaskInProgress, Priority: ptr(1),
				Created: "2026-07-31", Started: "2026-07-31",
				Tags: []string{"project-management"},
			},
			{
				ID: "ex-002", Title: "Implement the schema validator",
				Status: model.TaskTodo, Parent: "ex-001", Created: "2026-07-31",
				Blocked: &model.Blocked{
					Reason: "The schema must receive acceptance before implementation begins.",
					Tasks:  []string{"ex-001"},
					Since:  "2026-07-31",
				},
			},
		},
	}
}

const goldenSample = `schema-version: 1

project:
  id: example-project
  title: Example Project
  task-id-prefix: ex
  status: in-progress
  priority: 1
  areas:
    - knowledge-work
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: ex-001
    title: Define the accepted task schema
    description: |
      Record fields, lifecycle rules, and validation behavior.
    status: in-progress
    priority: 1
    created: "2026-07-31"
    started: "2026-07-31"
    tags:
      - project-management

  - id: ex-002
    title: Implement the schema validator
    status: todo
    parent: ex-001
    created: "2026-07-31"
    blocked:
      reason: The schema must receive acceptance before implementation begins.
      tasks:
        - ex-001
      since: "2026-07-31"
`

func TestEmitGolden(t *testing.T) {
	got := string(Document(sampleDoc()))
	if got != goldenSample {
		t.Fatalf("emit mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, goldenSample)
	}
}

func TestEmitEmptyTasks(t *testing.T) {
	doc := &model.Document{
		SchemaVersion: 1,
		Project: model.Project{
			ID: "empty", Title: "Empty", TaskIDPrefix: "em",
			Status: model.ProjectIdea, Created: "2026-07-31",
		},
	}
	want := "schema-version: 1\n\nproject:\n  id: empty\n  title: Empty\n  task-id-prefix: em\n  status: idea\n  created: \"2026-07-31\"\n\ntasks: []\n"
	if got := string(Document(doc)); got != want {
		t.Fatalf("empty tasks emit:\n%q\nwant\n%q", got, want)
	}
}

// TestEmitIdempotent decodes emitted bytes and re-emits them; the two renderings
// must be byte-identical.
func TestEmitIdempotent(t *testing.T) {
	docs := []*model.Document{sampleDoc(), quotingDoc()}
	for _, d := range docs {
		first := Document(d)
		parsed, err := decode.Decode(first)
		if err != nil {
			t.Fatalf("decode emitted bytes: %v\n%s", err, first)
		}
		second := Document(parsed)
		if !bytes.Equal(first, second) {
			t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
		}
	}
}

func quotingDoc() *model.Document {
	return &model.Document{
		SchemaVersion: 1,
		Project: model.Project{
			ID: "quote", Title: "Quote", TaskIDPrefix: "q",
			Status: model.ProjectIdea, Created: "2026-07-31",
		},
		Tasks: []model.Task{
			{ID: "q-001", Title: "1.0", Status: model.TaskTodo, Created: "2026-07-31"},             // numeric-looking
			{ID: "q-002", Title: "Fix: the parser", Status: model.TaskTodo, Created: "2026-07-31"}, // colon-space
			{ID: "q-003", Title: "true", Status: model.TaskTodo, Created: "2026-07-31"},            // boolean-looking
		},
	}
}

func TestEmitQuotesAmbiguousTitles(t *testing.T) {
	out := string(Document(quotingDoc()))
	for _, want := range []string{`title: "1.0"`, `title: "Fix: the parser"`, `title: "true"`} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}
