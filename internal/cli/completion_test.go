package cli

import (
	"strings"
	"testing"
)

// complete drives cobra's hidden __complete command, which is exactly what the
// generated shell scripts call.
func complete(t *testing.T, args ...string) []string {
	t.Helper()
	code, stdout, stderr := run(append([]string{"__complete"}, args...)...)
	if code != 0 {
		t.Fatalf("__complete exit %d: %s", code, stderr)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line == "" || strings.HasPrefix(line, ":") {
			continue // the trailing directive line
		}
		out = append(out, strings.SplitN(line, "\t", 2)[0])
	}
	return out
}

func has(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestCompletionCommandIsAvailable(t *testing.T) {
	code, stdout, stderr := run("completion", "bash")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "pm") {
		t.Fatalf("bash completion script looks empty: %.200q", stdout)
	}
	for _, shell := range []string{"zsh", "fish"} {
		if code, _, stderr := run("completion", shell); code != 0 {
			t.Fatalf("%s completion exit %d: %s", shell, code, stderr)
		}
	}
}

func TestCompleteTaskAndProjectIDs(t *testing.T) {
	root := seedTwoProjects(t)

	tasks := complete(t, "--root", root, "tasks", "show", "")
	for _, id := range []string{"al-001", "al-002", "bt-001", "bt-002"} {
		if !has(tasks, id) {
			t.Fatalf("task completion missing %s: %v", id, tasks)
		}
	}

	// A prefix narrows the list.
	if got := complete(t, "--root", root, "tasks", "show", "bt-"); len(got) != 2 || !has(got, "bt-001") {
		t.Fatalf("prefix completion = %v", got)
	}

	projects := complete(t, "--root", root, "projects", "show", "")
	if !has(projects, "alpha") || !has(projects, "beta") {
		t.Fatalf("project completion = %v", projects)
	}

	docProjects := complete(t, "--root", root, "projects", "doc", "")
	if !has(docProjects, "alpha") || !has(docProjects, "beta") {
		t.Fatalf("project doc completion = %v", docProjects)
	}
}

func TestCompleteStatusPositions(t *testing.T) {
	root := seedTwoProjects(t)

	// First argument of `tasks status` is a task ID, not a status.
	first := complete(t, "--root", root, "tasks", "status", "")
	if has(first, "in-progress") {
		t.Fatalf("statuses offered in the ID position: %v", first)
	}
	// From the second argument on, both are valid.
	second := complete(t, "--root", root, "tasks", "status", "al-001", "")
	if !has(second, "in-progress") || !has(second, "al-002") {
		t.Fatalf("second position should offer statuses and more IDs: %v", second)
	}
	// An ID already named is not offered again.
	if has(second, "al-001") {
		t.Fatalf("already-named task offered again: %v", second)
	}

	// Projects have their own vocabulary, including the new in-review status.
	proj := complete(t, "--root", root, "projects", "status", "alpha", "")
	if !has(proj, "in-review") || !has(proj, "idea") {
		t.Fatalf("project statuses = %v", proj)
	}
	if has(proj, "backlog") {
		t.Fatalf("backlog is a task status, not a project one: %v", proj)
	}
}

func TestCompleteFlagValues(t *testing.T) {
	root := seedTwoProjects(t)
	run("--root", root, "tasks", "edit", "al-001", "--add-tag", "research")

	if got := complete(t, "--root", root, "tasks", "list", "--tag", ""); !has(got, "research") {
		t.Fatalf("tag completion = %v", got)
	}
	if got := complete(t, "--root", root, "tasks", "list", "--project", ""); !has(got, "alpha") {
		t.Fatalf("project flag completion = %v", got)
	}
	if got := complete(t, "--root", root, "tasks", "add", "--status", ""); !has(got, "backlog") {
		t.Fatalf("status flag completion = %v", got)
	}
	if got := complete(t, "--root", root, "tasks", "list", "--parent", ""); !has(got, "al-001") {
		t.Fatalf("parent flag completion = %v", got)
	}
}

func TestCompletionStaysSilentOnABrokenRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/broken/broken.tasks.yaml", invalidProject)

	// A file that fails validation must not turn a keystroke into a diagnostic.
	code, _, stderr := run("__complete", "--root", root, "tasks", "show", "")
	if code != 0 {
		t.Fatalf("completion exit %d, want 0", code)
	}
	if strings.Contains(stderr, "nonsense") {
		t.Fatalf("completion leaked a validation error: %q", stderr)
	}
}
