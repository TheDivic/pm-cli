// Package mdrender turns a Markdown document into readable terminal text.
//
// It wraps charmbracelet/glamour, the library behind the `glow` terminal
// Markdown reader: full CommonMark parsing, tables, syntax-highlighted code
// blocks, and word wrap with hanging indents. An earlier version of this
// package was a small dependency-free regex renderer, but it could not wrap
// prose to the terminal width or lay out tables, which is most of what makes
// a long project document hard to read in a terminal.
package mdrender

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

// style is the glamour style used for both color and plain output. Plain
// output renders the same style and then strips the ANSI codes, rather than
// using glamour's own "notty"/"ascii" styles, which leave emphasis and code
// span markup unconsumed (`**bold**` prints literally) because they define no
// transformation for those elements.
const style = "dark"

// Render converts Markdown to terminal text at the given width. When color is
// false the escape codes produced by styling are stripped rather than never
// generated, so headings, lists, checkboxes, block quotes, and code blocks
// still lay out identically; only the color and boldness are gone.
func Render(md string, color bool, width int) string {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err != nil {
		return strings.TrimSpace(md) // styling is unavailable; the source is still readable
	}
	out, err := r.Render(skipFrontMatter(md))
	if err != nil {
		return strings.TrimSpace(md)
	}
	if !color {
		out = ansi.Strip(out)
	}
	return trimTrailingSpace(out)
}

// skipFrontMatter drops a leading YAML front-matter block, which is metadata
// for tools rather than something to read. Glamour renders it as a heading
// otherwise, since a bare "---" delimiter is indistinguishable from a
// thematic break without this.
func skipFrontMatter(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return md
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return md // an unterminated block is not front matter
}

// trimTrailingSpace removes the trailing spaces glamour pads lines with to
// fill a block's width, and the blank margin lines around the document.
func trimTrailingSpace(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}
