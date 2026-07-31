package cli

import (
	"fmt"
	"io"
	"strings"
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
// returns a dash when there are no tasks.
func miniProgress(done, total, width int) string {
	if total == 0 {
		return "-"
	}
	filled := filledCells(done, total, width)
	return fmt.Sprintf("%s%s %3d%%", strings.Repeat("█", filled), strings.Repeat("░", width-filled), pct(done, total))
}

// progressBar renders a fixed-width completion bar for done out of total. The
// caller guarantees total > 0.
func progressBar(done, total, width int) string {
	filled := filledCells(done, total, width)
	return fmt.Sprintf("[%s%s] %d%% (%d/%d done)",
		strings.Repeat("█", filled), strings.Repeat("░", width-filled), pct(done, total), done, total)
}
