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

func TestLayoutTextMapsCellsToPositions(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "ab\n界c"}}, Constraints{MaxWidth: 10, MaxHeight: 10}, TextLayoutOptions{})
	pos, ok := layout.PositionForCell(0, 1)
	if !ok {
		t.Fatal("PositionForCell failed")
	}
	if pos.Span != 0 || pos.ByteOffset != 1 || pos.RuneOffset != 1 || pos.GraphemeOffset != 1 {
		t.Fatalf("position = %#v, want second grapheme", pos)
	}
	row, col, ok := layout.CellForPosition(TextPosition{Span: 0, ByteOffset: 3, RuneOffset: 3, GraphemeOffset: 3})
	if !ok || row != 1 || col != 0 {
		t.Fatalf("cell for wide grapheme = %d,%d ok=%v, want 1,0 true", row, col, ok)
	}
	pos, ok = layout.PositionForCell(1, 1)
	if !ok || pos.ByteOffset != 3 {
		t.Fatalf("position in second cell of wide grapheme = %#v ok=%v, want byte offset 3", pos, ok)
	}
	pos, ok = layout.PositionForCell(1, 3)
	if !ok || pos.ByteOffset != len("ab\n界c") {
		t.Fatalf("end position = %#v ok=%v, want end of source", pos, ok)
	}
}

func TestLayoutTextMappingIncludesAlignmentOffset(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "ab"}}, Tight(Size{Width: 6, Height: 1}), TextLayoutOptions{Align: TextAlignCenter})
	if got := layout.Lines[0].Offset; got != 2 {
		t.Fatalf("offset = %d, want 2", got)
	}
	pos, ok := layout.PositionForCell(0, 2)
	if !ok || pos.ByteOffset != 0 {
		t.Fatalf("position at aligned text start = %#v ok=%v, want start", pos, ok)
	}
	row, col, ok := layout.CellForPosition(TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1})
	if !ok || row != 0 || col != 3 {
		t.Fatalf("cell for second grapheme = %d,%d ok=%v, want 0,3 true", row, col, ok)
	}
}

func TestLayoutTextMappingTracksSpans(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "ab"}, {Text: "cd"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	row, col, ok := layout.CellForPosition(TextPosition{Span: 1, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1})
	if !ok || row != 0 || col != 3 {
		t.Fatalf("cell for second span = %d,%d ok=%v, want 0,3 true", row, col, ok)
	}
	pos, ok := layout.PositionForCell(0, 2)
	if !ok || pos.Span != 1 || pos.ByteOffset != 0 {
		t.Fatalf("position at first cell of second span = %#v ok=%v, want span 1 offset 0", pos, ok)
	}
}

func TestLayoutTextEllipsisHidesClippedPositions(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "abcdef"}}, Constraints{MaxWidth: 3, MaxHeight: 1}, TextLayoutOptions{Overflow: TextOverflowEllipsis, MaxLines: 1})
	if _, _, ok := layout.CellForPosition(TextPosition{Span: 0, ByteOffset: 4, RuneOffset: 4, GraphemeOffset: 4}); ok {
		t.Fatal("hidden clipped position unexpectedly mapped to a cell")
	}
	pos, ok := layout.PositionForCell(0, 2)
	if !ok || pos.ByteOffset != 2 {
		t.Fatalf("ellipsis position = %#v ok=%v, want clipped boundary", pos, ok)
	}
	if !layout.Lines[0].Cells[2].Synthetic {
		t.Fatal("ellipsis cell should be synthetic")
	}
}
