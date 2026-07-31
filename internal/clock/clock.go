// Package clock provides the time source used for lifecycle dates. Production
// code uses System; tests inject Fixed so results never depend on the machine
// date.
package clock

import "time"

// Clock is the seam through which the CLI reads the current date. Lifecycle
// events (created, started, completed, and the like) use Today.
type Clock interface {
	// Now returns the current instant in the process's local timezone.
	Now() time.Time
	// Today returns the current calendar date as a "YYYY-MM-DD" string in the
	// process's local timezone, matching the task-file date format.
	Today() string
}

// dateLayout is the canonical task-file date format.
const dateLayout = "2006-01-02"

// System is the real clock backed by time.Now.
type System struct{}

// Now returns the current local time.
func (System) Now() time.Time { return time.Now() }

// Today returns the current local calendar date as "YYYY-MM-DD".
func (s System) Today() string { return s.Now().Format(dateLayout) }

// Fixed is a deterministic clock for tests.
type Fixed struct {
	// Instant is the time reported by Now and formatted by Today.
	Instant time.Time
}

// Now returns the fixed instant.
func (f Fixed) Now() time.Time { return f.Instant }

// Today returns the fixed instant's calendar date as "YYYY-MM-DD".
func (f Fixed) Today() string { return f.Instant.Format(dateLayout) }
