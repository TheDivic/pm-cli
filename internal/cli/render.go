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
