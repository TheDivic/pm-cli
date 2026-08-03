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
	ver, _, _ := buildMetadata()
	if !strings.Contains(stdout, ver) || !strings.Contains(stdout, "pm ") {
		t.Fatalf("version output %q missing version %q", stdout, ver)
	}
}

func TestMergeBuildInfo(t *testing.T) {
	tests := []struct {
		name                        string
		ver, rev, built             string
		mainVer, vcsRev, vcsTime    string
		wantVer, wantRev, wantBuilt string
	}{
		{
			name: "ldflags win over build info",
			ver:  "0.1.0", rev: "abc1234", built: "2026-08-03T00:00:00Z",
			mainVer: "v9.9.9", vcsRev: "ffffffffffff", vcsTime: "2020-01-01T00:00:00Z",
			wantVer: "0.1.0", wantRev: "abc1234", wantBuilt: "2026-08-03T00:00:00Z",
		},
		{
			name: "go install fills placeholders and shortens the revision",
			ver:  "dev", rev: "none", built: "unknown",
			mainVer: "v0.1.0", vcsRev: "0123456789abcdef", vcsTime: "2026-08-03T15:46:41Z",
			wantVer: "v0.1.0", wantRev: "0123456", wantBuilt: "2026-08-03T15:46:41Z",
		},
		{
			name: "local go build leaves placeholders when nothing is embedded",
			ver:  "dev", rev: "none", built: "unknown",
			mainVer: "(devel)", vcsRev: "", vcsTime: "",
			wantVer: "dev", wantRev: "none", wantBuilt: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, rev, built := mergeBuildInfo(tt.ver, tt.rev, tt.built, tt.mainVer, tt.vcsRev, tt.vcsTime)
			if ver != tt.wantVer || rev != tt.wantRev || built != tt.wantBuilt {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)",
					ver, rev, built, tt.wantVer, tt.wantRev, tt.wantBuilt)
			}
		})
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
