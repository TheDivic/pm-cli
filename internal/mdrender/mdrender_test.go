package mdrender

import (
	"strings"
	"testing"
)

// Rendering without color is the interesting case for tests: the markup must be
// consumed rather than printed, and the prose must survive intact.
func TestRenderPlain(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"h1 becomes a banner", "# PM CLI", "PM CLI"},
		{"h2 keeps its case", "## Decisions", "Decisions"},
		{"h3 keeps its case", "### Format", "Format"},
		{"bullet gets a glyph", "- one", "• one"},
		{"nested bullet keeps indent", "  - deep", "  • deep"},
		{"unchecked task box", "- [ ] do it", "☐ do it"},
		{"checked task box", "- [x] done it", "☑ done it"},
		{"ordered list survives", "1. first", "1. first"},
		{"bold markers are consumed", "a **strong** point", "a strong point"},
		{"italic markers are consumed", "an *emphatic* point", "an emphatic point"},
		{"underscore italics too", "an _emphatic_ point", "an emphatic point"},
		{"code spans are consumed", "run `pm tasks list` now", "run pm tasks list now"},
		{"links keep label and url", "see [the spec](docs/spec.md)", "see the spec (docs/spec.md)"},
		{"blockquote is marked", "> quoted", "│ quoted"},
		{"plain text passes through", "just prose", "just prose"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Render(tt.in, false); got != tt.want {
				t.Fatalf("Render(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRenderPlainHasNoEscapeCodes(t *testing.T) {
	in := "# Title\n\nSome **bold** and `code` and [a link](http://example.com).\n\n- item\n"
	if got := Render(in, false); strings.Contains(got, "\x1b") {
		t.Fatalf("color=false produced escape codes: %q", got)
	}
}

func TestRenderColorStyles(t *testing.T) {
	got := Render("## Decisions", true)
	if !strings.Contains(got, "\x1b[1m") || !strings.Contains(got, "Decisions") {
		t.Fatalf("h2 should be bold: %q", got)
	}
	if !strings.HasSuffix(got, reset) {
		t.Fatalf("styling should be reset: %q", got)
	}
}

func TestCodeFencesAreIndentedAndNotParsed(t *testing.T) {
	in := "text\n\n```sh\npm tasks list --all\n# not a heading\n```\n\nafter"
	got := Render(in, false)
	if strings.Contains(got, "```") {
		t.Fatalf("fences should not be printed: %q", got)
	}
	if !strings.Contains(got, "    pm tasks list --all") {
		t.Fatalf("code should be indented: %q", got)
	}
	// Markup inside a fence is content, not markup.
	if !strings.Contains(got, "    # not a heading") {
		t.Fatalf("code contents should not be rendered: %q", got)
	}
	if !strings.Contains(got, "after") {
		t.Fatalf("content after the fence was dropped: %q", got)
	}
}

func TestCodeSpanContentsAreNotEmphasized(t *testing.T) {
	// The asterisks belong to the code, not to Markdown.
	got := Render("use `a*b*c` here", false)
	if got != "use a*b*c here" {
		t.Fatalf("got %q", got)
	}
}

func TestFrontMatterIsSkipped(t *testing.T) {
	in := "---\ntitle: Something\n---\n\n# Real Title\n"
	got := Render(in, false)
	if strings.Contains(got, "title: Something") {
		t.Fatalf("front matter should not be shown: %q", got)
	}
	if got != "REAL TITLE" {
		t.Fatalf("got %q", got)
	}
}

func TestUnterminatedDashesAreARule(t *testing.T) {
	// Without a closing delimiter it is a thematic break, not front matter.
	got := Render("above\n\n---\n\nbelow", false)
	if !strings.Contains(got, "─") {
		t.Fatalf("expected a rule: %q", got)
	}
	if !strings.Contains(got, "above") || !strings.Contains(got, "below") {
		t.Fatalf("content lost: %q", got)
	}
}

func TestIsRule(t *testing.T) {
	for _, in := range []string{"---", "***", "___", "- - -", "----------"} {
		if !isRule(in) {
			t.Fatalf("%q should be a rule", in)
		}
	}
	for _, in := range []string{"--", "- item", "-- text", "a---b", ""} {
		if isRule(in) {
			t.Fatalf("%q should not be a rule", in)
		}
	}
}

func TestEmptyAndWhitespaceDocuments(t *testing.T) {
	for _, in := range []string{"", "\n\n\n", "   \n  \n"} {
		if got := Render(in, false); strings.TrimSpace(got) != "" {
			t.Fatalf("Render(%q) = %q, want empty", in, got)
		}
	}
}

func TestTablesPassThrough(t *testing.T) {
	in := "| a | b |\n|---|---|\n| 1 | 2 |"
	got := Render(in, false)
	for _, want := range []string{"| a | b |", "| 1 | 2 |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table row %q missing from %q", want, got)
		}
	}
}
