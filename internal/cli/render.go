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

// progressBar renders a fixed-width completion bar for done out of total. The
// caller guarantees total > 0.
func progressBar(done, total, width int) string {
	filled := (done*width + total/2) / total // rounded
	if filled > width {
		filled = width
	}
	pct := (done*100 + total/2) / total
	return fmt.Sprintf("[%s%s] %d%% (%d/%d done)",
		strings.Repeat("█", filled), strings.Repeat("░", width-filled), pct, done, total)
}
