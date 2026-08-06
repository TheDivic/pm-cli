package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

// labelWidth aligns the value column in key/value detail output.
const labelWidth = 12

// field writes a "Label: value" line with the value column aligned.
func field(w io.Writer, label, value string) {
	fmt.Fprintf(w, "%-*s%s\n", labelWidth, label+":", value)
}

// optField writes a field only when the value is non-empty.
func optField(w io.Writer, label, value string) {
	if value != "" {
		field(w, label, value)
	}
}

func joinComma(v []string) string { return strings.Join(v, ", ") }

// useColor reports whether ANSI styling is appropriate for w. Per the CLI
// specification, color is used only when the destination is a terminal and
// NO_COLOR is unset — so piped output and test buffers stay plain.
func useColor(w io.Writer) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// docWidth is the fallback and bound for the width Markdown documents are
// wrapped and ruled to: wide enough to read comfortably, capped so a maximized
// terminal does not stretch prose across the whole screen.
const (
	docWidthDefault = 72
	docWidthMin     = 40
	docWidthMax     = 100
)

// docWidth reports the column width to wrap a project document to. Piped or
// non-terminal output uses the same width the CLI has always ruled its
// document separator at.
func docWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return docWidthDefault
	}
	width, _, err := term.GetSize(f.Fd())
	if err != nil || width <= 0 {
		return docWidthDefault
	}
	switch {
	case width < docWidthMin:
		return docWidthMin
	case width > docWidthMax:
		return docWidthMax
	default:
		return width
	}
}

// filledCells returns how many of width cells to fill for done out of total,
// reserving a full bar for true completion and showing at least one cell for
// any progress. The caller guarantees total > 0.
func filledCells(done, total, width int) int {
	filled := (done*width + total/2) / total // rounded
	if filled >= width && done < total {
		filled = width - 1 // a full bar means 100%
	}
	if filled == 0 && done > 0 {
		filled = 1 // any progress shows at least one cell
	}
	return filled
}

func pct(done, total int) int { return (done*100 + total/2) / total }

// miniProgress renders a compact bar with a percentage for table cells. It
// returns a dash when there is nothing countable.
func miniProgress(done, countable, width int) string {
	if countable == 0 {
		return "-"
	}
	filled := filledCells(done, countable, width)
	return fmt.Sprintf("%s%s %3d%%", strings.Repeat("█", filled), strings.Repeat("░", width-filled), pct(done, countable))
}

// progressBar renders a fixed-width completion bar for done out of countable.
// Cancelled tasks are outside the ratio; they are reported alongside it so the
// denominator is self-explanatory. The caller guarantees countable > 0.
func progressBar(done, countable, cancelled, width int) string {
	filled := filledCells(done, countable, width)
	suffix := ""
	if cancelled > 0 {
		suffix = fmt.Sprintf(", %d cancelled", cancelled)
	}
	return fmt.Sprintf("[%s%s] %d%% (%d/%d done%s)",
		strings.Repeat("█", filled), strings.Repeat("░", width-filled), pct(done, countable), done, countable, suffix)
}
