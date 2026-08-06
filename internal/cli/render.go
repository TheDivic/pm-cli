package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/TheDivic/pm-cli/internal/pmerr"
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

// validColorModes are the values accepted by --color.
var validColorModes = map[string]bool{"auto": true, "always": true, "never": true}

// colorEnvVar names the environment variable that supplies a default --color
// mode when the flag is not given, mirroring $PM_ROOT for --root.
const colorEnvVar = "PM_COLOR"

// resolveColor returns the effective --color mode: the flag takes precedence
// when given, then $PM_COLOR, then "auto". opts.Color is empty when --color
// was not passed, since the flag has no default of its own — that way a
// user can set $PM_COLOR once instead of passing --color on every call.
func resolveColor(opts *GlobalOptions) string {
	if opts.Color != "" {
		return opts.Color
	}
	if env := os.Getenv(colorEnvVar); env != "" {
		return env
	}
	return "auto"
}

// validateColor rejects a --color value that isn't auto, always, or never,
// before it reaches useColor.
func validateColor(mode string) error {
	if !validColorModes[mode] {
		return pmerr.Usage("invalid --color value %q: must be auto, always, or never", mode)
	}
	return nil
}

// useColor reports whether ANSI styling is appropriate for w. mode is the
// validated --color flag: "always" and "never" bypass the terminal check
// entirely, which is the point of the override — piping into a pager like
// `less -R` otherwise loses styling because the pipe isn't a terminal. The
// default, "auto", keeps the original behavior: color only when NO_COLOR is
// unset and w is a real terminal, so piped output and test buffers stay
// plain unless the caller asks for color explicitly.
func useColor(w io.Writer, mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	}
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
