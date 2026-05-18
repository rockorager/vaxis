package ui

import "testing"

func TestLayoutTextPreservesBlankLines(t *testing.T) {
	layout := layoutText([]TextSpan{{Text: "a\n\nb"}}, Tight(Size{Width: 5, Height: 5}), textLayoutOptions{})
	if len(layout.Lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(layout.Lines))
	}
	if got := layout.Lines[0].Runs[0].Text; got != "a" {
		t.Fatalf("line 0 = %q, want a", got)
	}
	if layout.Lines[1].Width != 0 || len(layout.Lines[1].Runs) != 0 {
		t.Fatalf("line 1 = %#v, want blank", layout.Lines[1])
	}
	if got := layout.Lines[2].Runs[0].Text; got != "b" {
		t.Fatalf("line 2 = %q, want b", got)
	}
}

func TestLayoutTextEllipsisAtZeroWidthProducesEmptyLine(t *testing.T) {
	layout := layoutText([]TextSpan{{Text: "abc"}}, Constraints{MaxWidth: 0, MaxHeight: 1}, textLayoutOptions{Overflow: TextOverflowEllipsis, MaxLines: 1})
	if len(layout.Lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(layout.Lines))
	}
	if layout.Lines[0].Width != 0 || len(layout.Lines[0].Runs) != 0 {
		t.Fatalf("line = %#v, want empty", layout.Lines[0])
	}
}

func TestLayoutTextSoftWrapKeepsWideGraphemeWhole(t *testing.T) {
	layout := layoutText([]TextSpan{{Text: "a界b"}}, Constraints{MaxWidth: 2, MaxHeight: 10}, textLayoutOptions{SoftWrap: true})
	var got []string
	for _, line := range layout.Lines {
		if len(line.Runs) == 0 {
			got = append(got, "")
			continue
		}
		got = append(got, line.Runs[0].Text)
	}
	want := []string{"a", "界", "b"}
	if !sameStrings(got, want) {
		t.Fatalf("lines = %#v, want %#v", got, want)
	}
}

func TestLayoutTextEllipsisUsesLastVisibleStyle(t *testing.T) {
	red := Style{Foreground: RGB(255, 0, 0)}
	blue := Style{Foreground: RGB(0, 0, 255)}
	layout := layoutText([]TextSpan{{Text: "ab", Style: red}, {Text: "cd", Style: blue}}, Constraints{MaxWidth: 3, MaxHeight: 1}, textLayoutOptions{Overflow: TextOverflowEllipsis, MaxLines: 1})
	line := layout.Lines[0]
	if line.Width != 3 {
		t.Fatalf("width = %d, want 3", line.Width)
	}
	last := line.Runs[len(line.Runs)-1]
	if got := last.Text[len(last.Text)-len("…"):]; got != "…" {
		t.Fatalf("last run text = %q, want trailing ellipsis", last.Text)
	}
	if got := last.Style; got != red {
		t.Fatalf("ellipsis style = %#v, want red", got)
	}
}
