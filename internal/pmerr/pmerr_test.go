package pmerr

import (
	"errors"
	"testing"
)

func TestCodeExitCode(t *testing.T) {
	cases := map[Code]int{
		CodeValidation:  1,
		CodeUsage:       2,
		CodeIO:          3,
		CodeInternal:    3,
		Code("unknown"): 3,
	}
	for code, want := range cases {
		if got := code.ExitCode(); got != want {
			t.Errorf("Code(%q).ExitCode() = %d, want %d", code, got, want)
		}
	}
}

func TestErrorStringWithoutContext(t *testing.T) {
	e := Validation("schema-version 2 is unsupported")
	if got := e.Error(); got != "schema-version 2 is unsupported" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestErrorStringWithContext(t *testing.T) {
	e := Validation("status is invalid").
		WithFile("projects/demo/demo.tasks.yaml").
		WithTask("dm-001").
		WithField("status")
	want := "status is invalid (projects/demo/demo.tasks.yaml, task dm-001, field status)"
	if got := e.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("boom")
	e := IO("cannot read file").Wrapping(cause)
	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is could not reach the wrapped cause")
	}
}

func TestErrorsAsRecoversCode(t *testing.T) {
	var wrapped error = Validation("bad").WithProject("demo")
	var pe *Error
	if !errors.As(wrapped, &pe) {
		t.Fatalf("errors.As failed to recover *Error")
	}
	if pe.Code != CodeValidation || pe.Project != "demo" {
		t.Fatalf("recovered unexpected error: %+v", pe)
	}
}

func TestDetail(t *testing.T) {
	d := Validation("bad").WithFile("f.yaml").WithField("id").Detail()
	if d.Code != "validation" || d.Message != "bad" || d.File != "f.yaml" || d.Field != "id" {
		t.Fatalf("unexpected detail: %+v", d)
	}
	if d.Project != "" || d.Task != "" {
		t.Fatalf("empty context fields should stay empty: %+v", d)
	}
}
