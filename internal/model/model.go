// Package model defines the typed, in-memory representation of a schema
// version 1 *.tasks.yaml document. Decoding produces these types after
// syntax-level checks pass; validation and formatting operate on them.
package model

// SchemaVersion is the only supported schema version.
const SchemaVersion = 1

// ProjectStatus enumerates the allowed project lifecycle states.
type ProjectStatus string

// Project lifecycle states.
const (
	ProjectIdea       ProjectStatus = "idea"
	ProjectTodo       ProjectStatus = "todo"
	ProjectInProgress ProjectStatus = "in-progress"
	ProjectBlocked    ProjectStatus = "blocked"
	ProjectCancelled  ProjectStatus = "cancelled"
	ProjectDone       ProjectStatus = "done"
)

// Valid reports whether s is a recognized project status.
func (s ProjectStatus) Valid() bool {
	switch s {
	case ProjectIdea, ProjectTodo, ProjectInProgress, ProjectBlocked, ProjectCancelled, ProjectDone:
		return true
	default:
		return false
	}
}

// TaskStatus enumerates the allowed task lifecycle states.
type TaskStatus string

// Task lifecycle states.
const (
	TaskBacklog    TaskStatus = "backlog"
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in-progress"
	TaskInReview   TaskStatus = "in-review"
	TaskCancelled  TaskStatus = "cancelled"
	TaskDone       TaskStatus = "done"
)

// Valid reports whether s is a recognized task status.
func (s TaskStatus) Valid() bool {
	switch s {
	case TaskBacklog, TaskTodo, TaskInProgress, TaskInReview, TaskCancelled, TaskDone:
		return true
	default:
		return false
	}
}

// Terminal reports whether the task status is a terminal state.
func (s TaskStatus) Terminal() bool {
	return s == TaskCancelled || s == TaskDone
}

// Document is a complete task file: the schema version, the project record, and
// the flat, ordered task list.
type Document struct {
	SchemaVersion int
	Project       Project
	Tasks         []Task
}

// Project is the project record.
type Project struct {
	ID           string
	Title        string
	TaskIDPrefix string
	Status       ProjectStatus
	Priority     *int
	Areas        []string
	Created      string
	Started      string
	Due          string
	Blocked      *Blocked
	Cancellation *Cancellation
	Completed    string
}

// Task is one record in the flat task list. File order is meaningful and acts
// as priority within a shared status and parent group.
type Task struct {
	ID           string
	Title        string
	Description  string
	Status       TaskStatus
	Parent       string
	Created      string
	Started      string
	Due          string
	Tags         []string
	Blocked      *Blocked
	Cancellation *Cancellation
	Completed    string
}

// Blocked records a blocking condition on a nonterminal task or project.
type Blocked struct {
	Reason string
	Tasks  []string
	Since  string
}

// Cancellation records why and when a task or project was cancelled.
type Cancellation struct {
	Reason string
	Date   string
}
