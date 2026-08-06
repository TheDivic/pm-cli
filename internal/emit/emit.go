// Package emit renders a typed document back to canonical *.tasks.yaml bytes.
// The output follows the format specification exactly — field order, two-space
// indentation, quoted dates, literal description blocks, blank lines between
// tasks, and preserved file order — so that emitting is deterministic and
// running it twice without an edit produces identical bytes.
package emit

import (
	"bytes"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheDivic/pm-cli/internal/model"
)

// Document renders doc as canonical UTF-8 bytes with LF line endings.
func Document(doc *model.Document) []byte {
	var b bytes.Buffer
	b.WriteString("schema-version: ")
	b.WriteString(strconv.Itoa(doc.SchemaVersion))
	b.WriteString("\n\n")

	writeProject(&b, &doc.Project)
	b.WriteByte('\n')

	if len(doc.Tasks) == 0 {
		b.WriteString("tasks: []\n")
		return b.Bytes()
	}
	b.WriteString("tasks:\n")
	for i := range doc.Tasks {
		if i > 0 {
			b.WriteByte('\n') // blank line between tasks
		}
		writeTask(&b, &doc.Tasks[i])
	}
	return b.Bytes()
}

func writeProject(b *bytes.Buffer, p *model.Project) {
	b.WriteString("project:\n")
	kv(b, 2, "id", scalar(p.ID))
	kv(b, 2, "title", scalar(p.Title))
	kv(b, 2, "task-id-prefix", scalar(p.TaskIDPrefix))
	kv(b, 2, "status", scalar(string(p.Status)))
	if p.Priority != nil {
		kv(b, 2, "priority", strconv.Itoa(*p.Priority))
	}
	if len(p.Areas) > 0 {
		writeList(b, 2, "areas", p.Areas)
	}
	kv(b, 2, "created", date(p.Created))
	if p.Started != "" {
		kv(b, 2, "started", date(p.Started))
	}
	if p.Due != "" {
		kv(b, 2, "due", date(p.Due))
	}
	if p.Blocked != nil {
		writeBlocked(b, 2, p.Blocked)
	}
	if p.Cancellation != nil {
		writeCancellation(b, 2, p.Cancellation)
	}
	if p.Completed != "" {
		kv(b, 2, "completed", date(p.Completed))
	}
}

func writeTask(b *bytes.Buffer, t *model.Task) {
	indent(b, 2)
	b.WriteString("- id: ")
	b.WriteString(scalar(t.ID))
	b.WriteByte('\n')

	kv(b, 4, "title", scalar(t.Title))
	if t.Description != "" {
		writeLiteral(b, 4, "description", t.Description)
	}
	kv(b, 4, "status", scalar(string(t.Status)))
	if t.Priority != nil {
		kv(b, 4, "priority", strconv.Itoa(*t.Priority))
	}
	if t.Parent != "" {
		kv(b, 4, "parent", scalar(t.Parent))
	}
	kv(b, 4, "created", date(t.Created))
	if t.Started != "" {
		kv(b, 4, "started", date(t.Started))
	}
	if t.Due != "" {
		kv(b, 4, "due", date(t.Due))
	}
	if len(t.Tags) > 0 {
		writeList(b, 4, "tags", t.Tags)
	}
	if t.Blocked != nil {
		writeBlocked(b, 4, t.Blocked)
	}
	if t.Cancellation != nil {
		writeCancellation(b, 4, t.Cancellation)
	}
	if t.Completed != "" {
		kv(b, 4, "completed", date(t.Completed))
	}
}

func writeBlocked(b *bytes.Buffer, ind int, bl *model.Blocked) {
	line(b, ind, "blocked:")
	kv(b, ind+2, "reason", scalar(bl.Reason))
	if len(bl.Tasks) > 0 {
		writeList(b, ind+2, "tasks", bl.Tasks)
	}
	kv(b, ind+2, "since", date(bl.Since))
}

func writeCancellation(b *bytes.Buffer, ind int, c *model.Cancellation) {
	line(b, ind, "cancellation:")
	kv(b, ind+2, "reason", scalar(c.Reason))
	kv(b, ind+2, "date", date(c.Date))
}

func writeList(b *bytes.Buffer, ind int, key string, items []string) {
	line(b, ind, key+":")
	for _, it := range items {
		indent(b, ind+2)
		b.WriteString("- ")
		b.WriteString(scalar(it))
		b.WriteByte('\n')
	}
}

func writeLiteral(b *bytes.Buffer, ind int, key, content string) {
	line(b, ind, key+": |")
	for _, ln := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		if ln == "" {
			b.WriteByte('\n') // no trailing whitespace on blank lines
			continue
		}
		indent(b, ind+2)
		b.WriteString(ln)
		b.WriteByte('\n')
	}
}

func kv(b *bytes.Buffer, ind int, key, val string) {
	indent(b, ind)
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(val)
	b.WriteByte('\n')
}

func line(b *bytes.Buffer, ind int, s string) {
	indent(b, ind)
	b.WriteString(s)
	b.WriteByte('\n')
}

func indent(b *bytes.Buffer, n int) {
	for range n {
		b.WriteByte(' ')
	}
}

// date returns a quoted date so parsers keep it as a string.
func date(s string) string { return `"` + s + `"` }

// scalar renders a string as a plain scalar when that round-trips safely, or as
// a double-quoted scalar otherwise. It never folds, so output stays one line.
func scalar(s string) string {
	if plainSafe(s) {
		return s
	}
	return doubleQuote(s)
}

const unsafeStart = "-?:,[]{}#&*!|>'\"%@` \t"

func plainSafe(s string) bool {
	if s == "" {
		return false
	}
	if strings.IndexByte(unsafeStart, s[0]) >= 0 {
		return false
	}
	if strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return false
	}
	if strings.ContainsAny(s, "\n\t") {
		return false
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") || strings.Contains(s, " #") {
		return false
	}
	return !resolvesToNonString(s)
}

// resolvesToNonString reports whether the plain form of s would be read back as
// a non-string scalar (number, bool, null, timestamp), which would require
// quoting to stay a string.
func resolvesToNonString(s string) bool {
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return true
	}
	_, ok := v.(string)
	return !ok
}

func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
