package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validProject = `schema-version: 1

project:
  id: demo
  title: Demo Project
  task-id-prefix: dm
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: First
    status: todo
    created: "2026-07-31"
`

const invalidProject = `schema-version: 1

project:
  id: broken
  title: Broken
  task-id-prefix: bk
  status: nonsense
  created: "2026-07-31"

tasks: []
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectsListText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, stderr := run("--root", root, "projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "demo") || !strings.Contains(stdout, "Demo Project") {
		t.Fatalf("list output missing project: %q", stdout)
	}
	if !strings.Contains(stdout, "PROGRESS") || !strings.Contains(stdout, "0%") {
		t.Fatalf("list output missing progress column: %q", stdout)
	}
}

func TestProjectsListJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, _ := run("--json", "--root", root, "projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Projects []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(out.Projects) != 1 || out.Projects[0].ID != "demo" {
		t.Fatalf("unexpected projects: %+v", out.Projects)
	}
}

func listProjectFile(id, status, prio, created string) string {
	p := "schema-version: 1\n\nproject:\n  id: " + id + "\n  title: " + id + "\n  task-id-prefix: " + id[:1]
	p += "\n  status: " + status + "\n"
	if prio != "" {
		p += "  priority: " + prio + "\n"
	}
	p += "  created: \"" + created + "\"\n"
	if status == "in-progress" {
		p += "  started: \"" + created + "\"\n"
	}
	p += "\ntasks: []\n"
	return p
}

func listIDs(t *testing.T, root string, extra ...string) []string {
	t.Helper()
	args := append([]string{"--json", "--root", root, "projects", "list"}, extra...)
	code, stdout, stderr := run(args...)
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	var out struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var got []string
	for _, p := range out.Projects {
		got = append(got, p.ID)
	}
	return got
}

func TestProjectsListSortOrder(t *testing.T) {
	root := t.TempDir()
	// Same status; same priority with different created; a lower priority; none.
	writeFile(t, filepath.Join(root, "a.tasks.yaml"), listProjectFile("a", "idea", "1", "2026-02-01"))
	writeFile(t, filepath.Join(root, "c.tasks.yaml"), listProjectFile("c", "idea", "1", "2026-01-01"))
	writeFile(t, filepath.Join(root, "b.tasks.yaml"), listProjectFile("b", "idea", "2", "2026-01-01"))
	writeFile(t, filepath.Join(root, "z.tasks.yaml"), listProjectFile("z", "idea", "", "2026-01-01"))

	got := listIDs(t, root)
	// c (p1, Jan) < a (p1, Feb) < b (p2) < z (no priority).
	want := []string{"c", "a", "b", "z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestProjectsListInProgressFirst(t *testing.T) {
	root := t.TempDir()
	// An in-progress project with no priority must precede a prioritized idea.
	writeFile(t, filepath.Join(root, "prog.tasks.yaml"), listProjectFile("prog", "in-progress", "", "2026-03-01"))
	writeFile(t, filepath.Join(root, "idea.tasks.yaml"), listProjectFile("idea", "idea", "1", "2026-01-01"))
	// A second in-progress with a priority sorts ahead of the unprioritized one.
	writeFile(t, filepath.Join(root, "prog2.tasks.yaml"), listProjectFile("prog2", "in-progress", "1", "2026-05-01"))

	got := listIDs(t, root)
	want := []string{"prog2", "prog", "idea"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func projFileArea(id, status, prio, area string) string {
	s := "schema-version: 1\n\nproject:\n  id: " + id + "\n  title: " + id +
		"\n  task-id-prefix: " + id[:1] + "\n  status: " + status + "\n"
	if prio != "" {
		s += "  priority: " + prio + "\n"
	}
	s += "  areas:\n    - " + area + "\n  created: \"2026-07-31\"\n"
	if status == "in-progress" {
		s += "  started: \"2026-07-31\"\n"
	}
	return s + "\ntasks: []\n"
}

func TestProjectsListFilters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.tasks.yaml"), projFileArea("a", "idea", "1", "alpha"))
	writeFile(t, filepath.Join(root, "b.tasks.yaml"), projFileArea("b", "done", "", "beta"))
	writeFile(t, filepath.Join(root, "c.tasks.yaml"), projFileArea("c", "idea", "2", "alpha"))

	eq := func(got, want []string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	// idea status: a (prio 1) then c (prio 2).
	eq(listIDs(t, root, "--status", "idea"), []string{"a", "c"})
	// priority 1 only.
	eq(listIDs(t, root, "--priority", "1"), []string{"a"})
	// area beta only. b is done, so it needs --all to show up.
	eq(listIDs(t, root, "--all", "--area", "beta"), []string{"b"})
	eq(listIDs(t, root, "--area", "beta"), nil)
	// AND across filters: idea AND priority 2 -> c.
	eq(listIDs(t, root, "--status", "idea", "--priority", "2"), []string{"c"})
	// OR within a filter: priority 1 or 2 -> a, c.
	eq(listIDs(t, root, "--priority", "1", "--priority", "2"), []string{"a", "c"})
}

func TestRootFromEnvVar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	t.Setenv("PM_ROOT", root)

	// No --root: PM_ROOT supplies the discovery root.
	code, stdout, stderr := run("projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "demo") {
		t.Fatalf("PM_ROOT not used: %q", stdout)
	}

	// An explicit --root overrides PM_ROOT.
	empty := t.TempDir()
	_, stdout2, _ := run("--root", empty, "projects", "list")
	if strings.Contains(stdout2, "demo") {
		t.Fatalf("--root should override PM_ROOT: %q", stdout2)
	}
}

func TestProjectsValidateValid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, _ := run("--root", root, "projects", "validate")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "valid") {
		t.Fatalf("unexpected output: %q", stdout)
	}
}

func TestProjectsValidateInvalidExitsOne(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "broken", "broken.tasks.yaml"), invalidProject)

	code, stdout, _ := run("--root", root, "projects", "validate")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stdout, "status") {
		t.Fatalf("expected a status finding in output: %q", stdout)
	}
}

func TestProjectsValidateUnknownIDIsUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, _, stderr := run("--root", root, "projects", "validate", "nope")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
}

// projectInReview exercises the in-review project status added alongside the
// task status of the same name.
const projectInReview = `schema-version: 1

project:
  id: review
  title: Under Review
  task-id-prefix: rv
  status: in-review
  created: "2026-07-31"
  started: "2026-07-31"

tasks: []
`

func TestProjectInReviewValidates(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "review", "review.tasks.yaml"), projectInReview)

	if code, stdout, stderr := run("--root", root, "projects", "validate"); code != 0 {
		t.Fatalf("exit = %d, want 0 (stdout: %s stderr: %s)", code, stdout, stderr)
	}
	if code, _, stderr := run("--root", root, "projects", "status", "review", "in-review"); code != 0 {
		t.Fatalf("status transition exit = %d: %s", code, stderr)
	}
}

func TestProjectsListPutsInReviewFirst(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject) // in-progress
	writeFile(t, filepath.Join(root, "review", "review.tasks.yaml"), projectInReview)

	_, stdout, _ := run("--root", root, "projects", "list")
	review, progress := strings.Index(stdout, "review"), strings.Index(stdout, "demo")
	if review < 0 || progress < 0 {
		t.Fatalf("both projects should be listed: %q", stdout)
	}
	if review > progress {
		t.Fatalf("in-review should sort before in-progress:\n%s", stdout)
	}
}

func TestProjectsStatusRejectsUnknownStatus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	if code, _, _ := run("--root", root, "projects", "status", "demo", "in-revue"); code != 1 {
		t.Fatalf("typo status: exit = %d, want 1", code)
	}
}

const projectMarkdown = `# Demo Project

A short description with **emphasis** and ` + "`code`" + `.

## Decisions

- [x] Picked YAML
- [ ] Picked a renderer
`

func TestProjectsShowReportsTheDocumentPathOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	code, stdout, stderr := run("--root", root, "projects", "show", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Prefix:") {
		t.Fatalf("project fields missing: %s", stdout)
	}
	if !strings.Contains(stdout, "Doc:") || !strings.Contains(stdout, "demo.md") {
		t.Fatalf("doc path missing: %s", stdout)
	}
	// show points at the document; it does not render it. That's `projects doc`.
	for _, wantAbsent := range []string{"Decisions", "Picked YAML", "──"} {
		if strings.Contains(stdout, wantAbsent) {
			t.Fatalf("show should not render the document, found %q:\n%s", wantAbsent, stdout)
		}
	}
}

func TestProjectsShowWithoutADocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, stderr := run("--root", root, "projects", "show", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout, "Doc:") {
		t.Fatalf("no document should mean no Doc field:\n%s", stdout)
	}
}

func TestProjectsShowJSONCarriesOnlyTheDocumentPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	_, stdout, _ := run("--json", "--root", root, "projects", "show", "demo")
	var d struct {
		DocPath string `json:"doc_path"`
	}
	if err := json.Unmarshal([]byte(stdout), &d); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, stdout)
	}
	if d.DocPath != filepath.Join("demo", "demo.md") {
		t.Fatalf("doc_path = %q", d.DocPath)
	}
	// The content moved to `projects doc`; show's JSON no longer carries it.
	if strings.Contains(stdout, `"doc"`) {
		t.Fatalf("show JSON should not carry the document source: %s", stdout)
	}
}

func TestProjectsShowJSONOmitsAMissingDocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	_, stdout, _ := run("--json", "--root", root, "projects", "show", "demo")
	if strings.Contains(stdout, "doc_path") {
		t.Fatalf("absent document should be omitted: %s", stdout)
	}
}

func TestProjectsDocRendersTheProjectDocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	code, stdout, stderr := run("--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{"Demo Project", "Decisions", "[✓] Picked YAML", "[ ] Picked a renderer"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("rendered document missing %q:\n%s", want, stdout)
		}
	}
	// The project record itself is not repeated here; that's `projects show`.
	if strings.Contains(stdout, "Prefix:") {
		t.Fatalf("doc output should not repeat project fields:\n%s", stdout)
	}
	// Markup is consumed, and a non-terminal writer gets no escape codes.
	if strings.Contains(stdout, "**") || strings.Contains(stdout, "\x1b") {
		t.Fatalf("output should be plain rendered text:\n%q", stdout)
	}
}

func TestProjectsDocJSONCarriesTheDocumentSource(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	_, stdout, _ := run("--json", "--root", root, "projects", "doc", "demo")
	var d struct {
		ProjectID string `json:"project_id"`
		DocPath   string `json:"doc_path"`
		Doc       string `json:"doc"`
	}
	if err := json.Unmarshal([]byte(stdout), &d); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, stdout)
	}
	if d.ProjectID != "demo" {
		t.Fatalf("project_id = %q", d.ProjectID)
	}
	if d.DocPath != filepath.Join("demo", "demo.md") {
		t.Fatalf("doc_path = %q", d.DocPath)
	}
	// JSON carries the source, not the terminal rendering.
	if !strings.Contains(d.Doc, "# Demo Project") || !strings.Contains(d.Doc, "**emphasis**") {
		t.Fatalf("doc should be the raw Markdown: %q", d.Doc)
	}
}

func TestProjectsDocWithoutADocumentIsAUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, _, stderr := run("--root", root, "projects", "doc", "demo")
	if code != 2 {
		t.Fatalf("exit %d, want 2 (usage): %s", code, stderr)
	}
}

func TestProjectsDocUnknownProjectIsAUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, _, stderr := run("--root", root, "projects", "doc", "nope")
	if code != 2 {
		t.Fatalf("exit %d, want 2 (usage): %s", code, stderr)
	}
}

func TestProjectsDocColorAlwaysStylesEvenWithoutATerminal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	// The test harness writes to a bytes.Buffer, never a terminal, so this is
	// the only way to observe --color's override of the auto-detected default.
	code, stdout, stderr := run("--color", "always", "--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "\x1b[") {
		t.Fatalf("--color=always should style output even off a terminal:\n%q", stdout)
	}
}

func TestProjectsDocColorNeverSuppressesStyling(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	code, stdout, stderr := run("--color", "never", "--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("--color=never should never style output:\n%q", stdout)
	}
}

func TestProjectsDocColorAutoStaysPlainWithoutATerminal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	// No --color given: the default is "auto", which is the pre-existing
	// terminal-detection behavior.
	code, stdout, stderr := run("--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("--color=auto (default) should stay plain off a terminal:\n%q", stdout)
	}
}

func TestProjectsDocInvalidColorIsAUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	code, _, stderr := run("--color", "bogus", "--root", root, "projects", "doc", "demo")
	if code != 2 {
		t.Fatalf("exit %d, want 2 (usage): %s", code, stderr)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Fatalf("error should name the bad value: %s", stderr)
	}
}

func TestProjectsDocColorEnvVarSetsTheDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	t.Setenv("PM_COLOR", "always")
	code, stdout, stderr := run("--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "\x1b[") {
		t.Fatalf("$PM_COLOR=always should style output without --color:\n%q", stdout)
	}
}

func TestProjectsDocColorFlagOverridesEnvVar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	t.Setenv("PM_COLOR", "always")
	code, stdout, stderr := run("--color", "never", "--root", root, "projects", "doc", "demo")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("--color=never should override $PM_COLOR=always:\n%q", stdout)
	}
}

func TestProjectsDocInvalidColorEnvVarIsAUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)
	writeFile(t, filepath.Join(root, "demo", "demo.md"), projectMarkdown)

	t.Setenv("PM_COLOR", "bogus")
	code, _, stderr := run("--root", root, "projects", "doc", "demo")
	if code != 2 {
		t.Fatalf("exit %d, want 2 (usage): %s", code, stderr)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Fatalf("error should name the bad value: %s", stderr)
	}
}

func TestProjectsListHidesTerminalProjectsByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "live.tasks.yaml"), projFileArea("live", "in-progress", "", "alpha"))
	writeFile(t, filepath.Join(root, "shipped.tasks.yaml"), projFileArea("shipped", "done", "", "alpha"))
	writeFile(t, filepath.Join(root, "dropped.tasks.yaml"), projFileArea("dropped", "cancelled", "", "alpha"))
	writeFile(t, filepath.Join(root, "idea.tasks.yaml"), projFileArea("idea", "idea", "", "alpha"))

	// Finished work accumulates without bound, so it is hidden by default.
	got := listIDs(t, root)
	if strings.Join(got, ",") != "live,idea" {
		t.Fatalf("default listing = %v, want [live idea]", got)
	}

	// -a/--all brings it back.
	all := listIDs(t, root, "--all")
	if len(all) != 4 {
		t.Fatalf("--all listing = %v, want 4 projects", all)
	}
	if short := listIDs(t, root, "-a"); len(short) != 4 {
		t.Fatalf("-a listing = %v, want 4 projects", short)
	}

	// An explicit --status overrides the default, exactly as for tasks list.
	if got := listIDs(t, root, "--status", "done"); strings.Join(got, ",") != "shipped" {
		t.Fatalf("--status done = %v, want [shipped]", got)
	}
	if got := listIDs(t, root, "-s", "cancelled"); strings.Join(got, ",") != "dropped" {
		t.Fatalf("-s cancelled = %v, want [dropped]", got)
	}
}
