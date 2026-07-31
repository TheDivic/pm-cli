// Package pmerr defines the CLI's structured error type. Every error carries a
// category Code that maps to a process exit code and optional location context
// (file, project, task, field) so both humans and agents can act on it.
package pmerr

import (
	"fmt"
	"strings"
)

// Code categorizes an error and determines the process exit code.
type Code string

const (
	// CodeValidation means task data, references, or a requested transition is
	// invalid. Exit code 1.
	CodeValidation Code = "validation"
	// CodeUsage means the command syntax or arguments are invalid. Exit code 2.
	CodeUsage Code = "usage"
	// CodeIO means an I/O or locking problem occurred. Exit code 3.
	CodeIO Code = "io"
	// CodeInternal means an unexpected internal error occurred. Exit code 3.
	CodeInternal Code = "internal"
)

// ExitCode returns the process exit code for the category, per the CLI
// specification: 1 invalid data, 2 invalid usage, 3 I/O or internal.
func (c Code) ExitCode() int {
	switch c {
	case CodeValidation:
		return 1
	case CodeUsage:
		return 2
	case CodeIO, CodeInternal:
		return 3
	default:
		return 3
	}
}

// Error is a categorized CLI error with optional location context.
type Error struct {
	Code    Code
	Message string
	File    string
	Project string
	Task    string
	Field   string
	err     error
}

// New builds an Error with the given category and message.
func New(code Code, message string) *Error { return &Error{Code: code, Message: message} }

// Usage builds a CodeUsage error from a format string.
func Usage(format string, a ...any) *Error { return New(CodeUsage, fmt.Sprintf(format, a...)) }

// Validation builds a CodeValidation error from a format string.
func Validation(format string, a ...any) *Error {
	return New(CodeValidation, fmt.Sprintf(format, a...))
}

// IO builds a CodeIO error from a format string.
func IO(format string, a ...any) *Error { return New(CodeIO, fmt.Sprintf(format, a...)) }

// Internal builds a CodeInternal error from a format string.
func Internal(format string, a ...any) *Error { return New(CodeInternal, fmt.Sprintf(format, a...)) }

// WithFile sets the file path context and returns the error for chaining.
func (e *Error) WithFile(path string) *Error { e.File = path; return e }

// WithProject sets the project ID context and returns the error for chaining.
func (e *Error) WithProject(id string) *Error { e.Project = id; return e }

// WithTask sets the task ID context and returns the error for chaining.
func (e *Error) WithTask(id string) *Error { e.Task = id; return e }

// WithField sets the field-path context and returns the error for chaining.
func (e *Error) WithField(path string) *Error { e.Field = path; return e }

// Wrapping records an underlying cause reachable through errors.Unwrap.
func (e *Error) Wrapping(cause error) *Error { e.err = cause; return e }

// Unwrap exposes the wrapped cause for errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.err }

// Error renders the message followed by any available location context.
func (e *Error) Error() string {
	var loc []string
	if e.File != "" {
		loc = append(loc, e.File)
	}
	if e.Project != "" {
		loc = append(loc, "project "+e.Project)
	}
	if e.Task != "" {
		loc = append(loc, "task "+e.Task)
	}
	if e.Field != "" {
		loc = append(loc, "field "+e.Field)
	}
	if len(loc) == 0 {
		return e.Message
	}
	return e.Message + " (" + strings.Join(loc, ", ") + ")"
}

// Detail is the machine-readable form emitted in --json mode.
type Detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
	Project string `json:"project,omitempty"`
	Task    string `json:"task,omitempty"`
	Field   string `json:"field,omitempty"`
}

// Detail returns the error's machine-readable representation.
func (e *Error) Detail() Detail {
	return Detail{
		Code:    string(e.Code),
		Message: e.Message,
		File:    e.File,
		Project: e.Project,
		Task:    e.Task,
		Field:   e.Field,
	}
}
