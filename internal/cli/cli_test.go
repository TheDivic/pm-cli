package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func run(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = Main(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionSucceeds(t *testing.T) {
	code, stdout, stderr := run("version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, version) || !strings.Contains(stdout, "pm ") {
		t.Fatalf("version output %q missing version %q", stdout, version)
	}
}

func TestNoArgsShowsHelpAndSucceeds(t *testing.T) {
	code, stdout, _ := run()
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("expected help output, got: %q", stdout)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, stderr := run("frobnicate")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if stderr == "" {
		t.Fatalf("expected an error message on stderr")
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	code, _, _ := run("version", "--nope")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestJSONErrorOutput(t *testing.T) {
	code, _, stderr := run("--json", "frobnicate")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	var d struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(stderr), &d); err != nil {
		t.Fatalf("stderr is not valid JSON: %v\n%s", err, stderr)
	}
	if d.Code != "usage" {
		t.Fatalf("json code = %q, want %q", d.Code, "usage")
	}
	if d.Message == "" {
		t.Fatalf("json message should not be empty")
	}
}

func TestProjectsAndTasksParentsSucceed(t *testing.T) {
	for _, group := range []string{"projects", "tasks"} {
		code, stdout, stderr := run(group)
		if code != 0 {
			t.Fatalf("%s exit = %d, want 0 (stderr: %s)", group, code, stderr)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Fatalf("%s should print help, got: %q", group, stdout)
		}
	}
}
