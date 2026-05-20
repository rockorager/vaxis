package richtext

import (
	"slices"
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

var testDrawContext = vxfw.DrawContext{
	Characters: vaxis.Characters,
}

func testLayout(input string) richTextLayout {
	return richTextLayoutFromSegments([]vaxis.Segment{{Text: input}})
}

func cellsString(cells []vaxis.Cell) string {
	var str strings.Builder
	for _, cell := range cells {
		str.WriteString(cell.Grapheme)
	}
	return str.String()
}

func TestHardwrapScanner(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no breaks",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:  "single hard break",
			input: "each line\nfits",
			expected: []string{
				"each line",
				"fits",
			},
		},
		{
			name:  "sequential hardbreak",
			input: "each line\n\nfits",
			expected: []string{
				"each line",
				"",
				"fits",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scanner := newHardwrapScanner(testLayout(test.input))
			lines := []string{}
			for scanner.Scan() {
				lines = append(lines, cellsString(scanner.Line(testDrawContext)))
			}
			if !slices.Equal(lines, test.expected) {
				t.Fatalf("lines = %#v, want %#v", lines, test.expected)
			}
		})
	}
}

func TestLayoutStylesGraphemeFromStartOffset(t *testing.T) {
	bold := vaxis.Style{Attribute: vaxis.AttrBold}
	italic := vaxis.Style{Attribute: vaxis.AttrItalic}
	layout := richTextLayoutFromSegments([]vaxis.Segment{
		{Text: "e", Style: bold},
		{Text: "\u0301x", Style: italic},
	})

	cells := layout.cells(testDrawContext, 0, len(layout.text))
	if len(cells) != 2 {
		t.Fatalf("len(cells) = %d, want 2", len(cells))
	}
	if got := cells[0].Grapheme; got != "e\u0301" {
		t.Fatalf("cells[0].Grapheme = %q, want %q", got, "e\u0301")
	}
	if got := cells[0].Style; got != bold {
		t.Fatalf("cells[0].Style = %#v, want %#v", got, bold)
	}
	if got := cells[1].Grapheme; got != "x" {
		t.Fatalf("cells[1].Grapheme = %q, want %q", got, "x")
	}
	if got := cells[1].Style; got != italic {
		t.Fatalf("cells[1].Style = %#v, want %#v", got, italic)
	}
}

func TestSoftWrapScanner(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		width    uint16
	}{
		{
			name:     "no wrap, perfect width",
			input:    "foo",
			expected: []string{"foo"},
			width:    3,
		},
		{
			name:     "no wrap, large",
			input:    "foo",
			expected: []string{"foo"},
			width:    4,
		},
		{
			name:  "simple",
			input: "foo bar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "hard break",
			input: "foo\nbar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "lots of space",
			input: "foo         bar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "hard break and leading space",
			input: " foo\n bar",
			expected: []string{
				" foo",
				" bar",
			},
			width: 4,
		},
		{
			name:  "long word",
			input: "longwordwithnobreaks",
			expected: []string{
				"long",
				"word",
				"with",
				"nobr",
				"eaks",
			},
			width: 4,
		},
		{
			name:  "erock: 3 lines",
			input: "Line 1\nLine 2\nLine 3\n",
			expected: []string{
				"Line 1",
				"Line 2",
				"Line 3",
			},
			width: 6,
		},
		{
			name:  "no soft wrap needed",
			input: "each line\nfits",
			expected: []string{
				"each line",
				"fits",
			},
			width: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scanner := newSoftwrapScanner(testLayout(test.input), test.width)
			lines := []string{}
			for scanner.Scan(testDrawContext) {
				lines = append(lines, cellsString(scanner.Text(testDrawContext)))
			}

			if !slices.Equal(lines, test.expected) {
				t.Fatalf("lines = %#v, want %#v", lines, test.expected)
			}
		})
	}
}
