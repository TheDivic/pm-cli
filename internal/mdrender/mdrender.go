// Package mdrender turns a Markdown document into readable terminal text.
//
// It is deliberately small rather than a CommonMark implementation: `pm` ships
// as a single self-contained binary, and a full renderer would pull in a large
// dependency tree to format prose that is already readable as plaintext. The
// goal is a document that scans well in a terminal — headings that stand out,
// bullets that align, code that is visually separate — not typographic fidelity.
//
// Constructs it does not understand pass through unchanged, which is the right
// failure mode for a format whose source is meant to be legible on its own.
package mdrender

import (
	"regexp"
	"strconv"
	"strings"
)

// ANSI escape sequences, used only when the caller asks for color.
const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	dim       = "\x1b[2m"
	italic    = "\x1b[3m"
	underline = "\x1b[4m"
	cyan      = "\x1b[36m"
)

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	bulletRe  = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	orderedRe = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+(.*)$`)
	quoteRe   = regexp.MustCompile(`^\s*>\s?(.*)$`)
	fenceRe   = regexp.MustCompile("^\\s*(```|~~~)(.*)$")

	codeSpanRe = regexp.MustCompile("`([^`]+)`")
	linkRe     = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	boldRe     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emphAsterR = regexp.MustCompile(`(^|[^*\w])\*([^*]+)\*`)
	emphUnderR = regexp.MustCompile(`(^|[^_\w])_([^_]+)_`)
)

// ruleWidth is the width of a rendered horizontal rule. Terminal width is not
// knowable without a dependency, so a fixed rule that fits an 80-column terminal
// is used rather than guessing.
const ruleWidth = 72

// Render converts Markdown to terminal text. When color is false the markup is
// still removed — emphasis markers, code fences, and link syntax — so the output
// reads as prose rather than as source.
func Render(md string, color bool) string {
	s := newStyler(color)
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	lines = skipFrontMatter(lines)

	var out []string
	inCode := false
	for _, line := range lines {
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			inCode = !inCode
			continue // the fence itself is never shown
		}
		if inCode {
			out = append(out, "    "+s.style(line, dim))
			continue
		}
		out = append(out, s.renderLine(line))
	}
	return strings.Join(trimBlankEdges(out), "\n")
}

// skipFrontMatter drops a leading YAML front-matter block, which is metadata for
// tools rather than something to read.
func skipFrontMatter(lines []string) []string {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return lines
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return lines[i+1:]
		}
	}
	return lines // an unterminated block is not front matter
}

func trimBlankEdges(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// styler applies ANSI styling, or drops it entirely when color is off.
type styler struct{ color bool }

func newStyler(color bool) styler { return styler{color: color} }

func (s styler) style(text string, codes ...string) string {
	if !s.color || text == "" {
		return text
	}
	return strings.Join(codes, "") + text + reset
}

// checkbox renders a task-list marker. Project documents lean on these for
// requirement checklists, where a glyph is far easier to scan than "[ ]".
func checkbox(text string) (box, rest string, ok bool) {
	switch {
	case strings.HasPrefix(text, "[ ] "):
		return "☐", text[4:], true
	case strings.HasPrefix(text, "[x] "), strings.HasPrefix(text, "[X] "):
		return "☑", text[4:], true
	}
	return "", text, false
}

// isRule reports whether a line is a thematic break: three or more of the same
// marker, optionally spaced. RE2 has no backreferences, so this is a predicate
// rather than one pattern.
func isRule(line string) bool {
	trimmed := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, line)
	if len(trimmed) < 3 {
		return false
	}
	switch trimmed[0] {
	case '-', '*', '_':
		return strings.Count(trimmed, string(trimmed[0])) == len(trimmed)
	}
	return false
}

func (s styler) renderLine(line string) string {
	if isRule(line) {
		return s.style(strings.Repeat("─", ruleWidth), dim)
	}
	if m := headingRe.FindStringSubmatch(line); m != nil {
		return s.heading(len(m[1]), s.inline(m[2]))
	}
	if m := quoteRe.FindStringSubmatch(line); m != nil {
		return s.style("│ ", dim) + s.style(s.inline(m[1]), italic)
	}
	if m := bulletRe.FindStringSubmatch(line); m != nil {
		if box, rest, ok := checkbox(m[2]); ok {
			return m[1] + box + " " + s.inline(rest)
		}
		return m[1] + s.style("• ", dim) + s.inline(m[2])
	}
	if m := orderedRe.FindStringSubmatch(line); m != nil {
		return m[1] + s.style(m[2]+". ", dim) + s.inline(m[3])
	}
	return s.inline(line)
}

// heading renders a heading by level: the title of the document reads as a
// banner, sections as bold, and anything deeper stays bold but unobtrusive.
func (s styler) heading(level int, text string) string {
	switch level {
	case 1:
		return s.style(strings.ToUpper(text), bold, underline)
	case 2:
		return s.style(text, bold)
	default:
		return s.style(text, bold, dim)
	}
}

// inline applies span-level markup. Code spans are lifted out first so their
// contents are never treated as emphasis — `a*b*c` is code, not italics.
func (s styler) inline(text string) string {
	var spans []string
	text = codeSpanRe.ReplaceAllStringFunc(text, func(m string) string {
		spans = append(spans, codeSpanRe.FindStringSubmatch(m)[1])
		return "\x00" + strconv.Itoa(len(spans)-1) + "\x00"
	})

	text = linkRe.ReplaceAllStringFunc(text, func(m string) string {
		p := linkRe.FindStringSubmatch(m)
		label, url := p[1], p[2]
		if label == "" {
			return s.style(url, underline, cyan)
		}
		return s.style(label, underline) + s.style(" ("+url+")", dim)
	})
	text = boldRe.ReplaceAllStringFunc(text, func(m string) string {
		return s.style(boldRe.FindStringSubmatch(m)[1], bold)
	})
	text = emphAsterR.ReplaceAllStringFunc(text, func(m string) string {
		p := emphAsterR.FindStringSubmatch(m)
		return p[1] + s.style(p[2], italic)
	})
	text = emphUnderR.ReplaceAllStringFunc(text, func(m string) string {
		p := emphUnderR.FindStringSubmatch(m)
		return p[1] + s.style(p[2], italic)
	})

	for i, span := range spans {
		text = strings.ReplaceAll(text, "\x00"+strconv.Itoa(i)+"\x00", s.style(span, cyan))
	}
	return text
}
