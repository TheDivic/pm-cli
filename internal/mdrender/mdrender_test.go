package mdrender

import (
	"strings"
	"testing"
)

// Rendering without color is the interesting case for most of these tests:
// the markup must be consumed rather than printed, and the prose must
// survive intact, even though styling ran and was stripped rather than never
// generated.
func TestRenderPlain(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"h1 becomes a banner", "# PM CLI", []string{"PM CLI"}},
		{"h2 keeps its hash prefix", "## Decisions", []string{"## Decisions"}},
		{"bullet gets a glyph", "- one", []string{"• one"}},
		{"unchecked task box", "- [ ] do it", []string{"[ ] do it"}},
		{"checked task box", "- [x] done it", []string{"[✓] done it"}},
		{"ordered list survives", "1. first", []string{"1. first"}},
		{"bold markers are consumed", "a **strong** point", []string{"strong"}},
		{"italic markers are consumed", "an *emphatic* point", []string{"emphatic"}},
		{"code spans are consumed", "run `pm tasks list` now", []string{"pm tasks list"}},
		{"links keep the label", "see [the spec](docs/spec.md)", []string{"the spec"}},
		{"blockquote is marked", "> quoted", []string{"│", "quoted"}},
		{"plain text passes through", "just prose", []string{"just prose"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in, false, 80)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("Render(%q) = %q, want it to contain %q", tt.in, got, want)
				}
			}
			for _, markup := range []string{"**", "`"} {
				if strings.Contains(got, markup) {
					t.Fatalf("Render(%q) = %q, markup %q was not consumed", tt.in, got, markup)
				}
			}
		})
	}
}

func TestRenderPlainHasNoEscapeCodes(t *testing.T) {
	in := "# Title\n\nSome **bold** and `code` and [a link](http://example.com).\n\n- item\n"
	if got := Render(in, false, 80); strings.Contains(got, "\x1b") {
		t.Fatalf("color=false produced escape codes: %q", got)
	}
}

func TestRenderColorProducesEscapeCodes(t *testing.T) {
	got := Render("## Decisions", true, 80)
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("color=true should style output: %q", got)
	}
	if !strings.Contains(got, "Decisions") {
		t.Fatalf("styled output lost the text: %q", got)
	}
}

func TestCodeFencesAreIndentedAndNotParsed(t *testing.T) {
	in := "text\n\n```sh\npm tasks list --all\n# not a heading\n```\n\nafter"
	got := Render(in, false, 80)
	if strings.Contains(got, "```") {
		t.Fatalf("fences should not be printed: %q", got)
	}
	if !strings.Contains(got, "pm tasks list --all") {
		t.Fatalf("code content missing: %q", got)
	}
	// Markup inside a fence is content, not markup: the heading marker should
	// still be there, just not turned into a heading.
	if !strings.Contains(got, "# not a heading") {
		t.Fatalf("code contents should not be rendered as Markdown: %q", got)
	}
	if !strings.Contains(got, "after") {
		t.Fatalf("content after the fence was dropped: %q", got)
	}
}

func TestCodeSpanContentsAreNotEmphasized(t *testing.T) {
	// The asterisks belong to the code, not to Markdown.
	got := Render("use `a*b*c` here", false, 80)
	if !strings.Contains(got, "a*b*c") {
		t.Fatalf("got %q", got)
	}
}

func TestFrontMatterIsSkipped(t *testing.T) {
	in := "---\ntitle: Something\n---\n\n# Real Title\n"
	got := Render(in, false, 80)
	if strings.Contains(got, "title: Something") {
		t.Fatalf("front matter should not be shown: %q", got)
	}
	if !strings.Contains(got, "Real Title") {
		t.Fatalf("got %q", got)
	}
}

func TestLongProseWrapsWithAHangingIndent(t *testing.T) {
	in := "- " + strings.Repeat("word ", 40)
	got := Render(in, false, 40)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapping to produce multiple lines, got %d: %q", len(lines), got)
	}
	for _, l := range lines {
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, "  ") {
			t.Fatalf("wrapped line missing hanging indent: %q", l)
		}
	}
}

func TestEmptyAndWhitespaceDocuments(t *testing.T) {
	for _, in := range []string{"", "\n\n\n", "   \n  \n"} {
		if got := Render(in, false, 80); strings.TrimSpace(got) != "" {
			t.Fatalf("Render(%q) = %q, want empty", in, got)
		}
	}
}

func TestTablesAreFormatted(t *testing.T) {
	in := "| a | b |\n|---|---|\n| 1 | 2 |"
	got := Render(in, false, 80)
	for _, want := range []string{"a", "b", "1", "2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table cell %q missing from %q", want, got)
		}
	}
}

func TestNonPositiveWidthFallsBackToADefault(t *testing.T) {
	if got := Render("hello", false, 0); !strings.Contains(got, "hello") {
		t.Fatalf("Render with width=0 should still render: %q", got)
	}
}
