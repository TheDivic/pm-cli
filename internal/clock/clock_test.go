package clock

import (
	"testing"
	"time"
)

func TestFixedTodayFormatsCalendarDate(t *testing.T) {
	f := Fixed{Instant: time.Date(2026, 7, 31, 23, 59, 0, 0, time.UTC)}
	if got := f.Today(); got != "2026-07-31" {
		t.Fatalf("Today() = %q, want %q", got, "2026-07-31")
	}
}

func TestFixedNowReturnsInstant(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f := Fixed{Instant: want}
	if got := f.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestSystemTodayMatchesNow(t *testing.T) {
	s := System{}
	if got, want := s.Today(), s.Now().Format("2006-01-02"); got != want {
		t.Fatalf("Today() = %q, want %q", got, want)
	}
}

// Ensure the concrete clocks satisfy the interface.
var (
	_ Clock = System{}
	_ Clock = Fixed{}
)
