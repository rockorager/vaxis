package ui

import "testing"

func TestTextSelectionIntersectsCells(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "abcd"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
		Extent: TextPosition{Span: 0, ByteOffset: 3, RuneOffset: 3, GraphemeOffset: 3},
	}
	line := layout.Lines[0]
	if selection.IntersectsCell(line.Cells[0]) {
		t.Fatal("first cell should not intersect selection")
	}
	if !selection.IntersectsCell(line.Cells[1]) {
		t.Fatal("second cell should intersect selection")
	}
	if !selection.IntersectsCell(line.Cells[2]) {
		t.Fatal("third cell should intersect selection")
	}
	if selection.IntersectsCell(line.Cells[3]) {
		t.Fatal("fourth cell should not intersect selection")
	}
}

func TestTextSelectionIntersectsCellsAcrossSpans(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "ab"}, {Text: "cd"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
		Extent: TextPosition{Span: 1, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
	}
	line := layout.Lines[0]
	if selection.IntersectsCell(line.Cells[0]) {
		t.Fatal("first cell should not intersect selection")
	}
	if !selection.IntersectsCell(line.Cells[1]) {
		t.Fatal("second cell should intersect selection")
	}
	if !selection.IntersectsCell(line.Cells[2]) {
		t.Fatal("third cell should intersect selection")
	}
	if selection.IntersectsCell(line.Cells[3]) {
		t.Fatal("fourth cell should not intersect selection")
	}
}

func TestTextSelectionIntersectsWideCell(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "a界b"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
		Extent: TextPosition{Span: 0, ByteOffset: len("a界"), RuneOffset: 2, GraphemeOffset: 2},
	}
	line := layout.Lines[0]
	if !selection.IntersectsCell(line.Cells[1]) {
		t.Fatal("wide cell should intersect selection")
	}
}

func TestTextSelectionContainsLineBreak(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "a\n\nb"}}, Constraints{MaxWidth: 10, MaxHeight: 4}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: len("a\n"), RuneOffset: 2, GraphemeOffset: 2},
		Extent: TextPosition{Span: 0, ByteOffset: len("a\n\n"), RuneOffset: 3, GraphemeOffset: 3},
	}
	if !selection.ContainsLineBreak(layout.Lines[1]) {
		t.Fatal("empty line break should be selected")
	}
	if selection.ContainsLineBreak(layout.Lines[0]) {
		t.Fatal("first line break should not be selected")
	}
}

func TestTextSelectionCollapsedDoesNotSelectCellOrLineBreak(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "a\n"}}, Constraints{MaxWidth: 10, MaxHeight: 2}, TextLayoutOptions{})
	selection := TextSelection{Base: layout.Lines[0].Start, Extent: layout.Lines[0].Start}
	if selection.IntersectsCell(layout.Lines[0].Cells[0]) {
		t.Fatal("collapsed selection should not select cells")
	}
	if selection.ContainsLineBreak(layout.Lines[0]) {
		t.Fatal("collapsed selection should not select line breaks")
	}
}

func TestTextLayoutSelectionRangesUseLayoutCoordinates(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "ab\ncd"}}, Tight(Size{Width: 6, Height: 2}), TextLayoutOptions{Align: TextAlignCenter})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
		Extent: TextPosition{Span: 0, ByteOffset: len("ab\nc"), RuneOffset: 4, GraphemeOffset: 4},
	}
	got := layout.SelectionRanges(selection)
	want := []TextSelectionRange{
		{Row: 0, Col: 3, Width: 1},
		{Row: 1, Col: 2, Width: 1},
	}
	if !sameSelectionRanges(got, want) {
		t.Fatalf("ranges = %#v, want %#v", got, want)
	}
}

func TestTextLayoutSelectionRangesIncludeEmptyLines(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "a\n\nb"}}, Constraints{MaxWidth: 10, MaxHeight: 4}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: len("a\n"), RuneOffset: 2, GraphemeOffset: 2},
		Extent: TextPosition{Span: 0, ByteOffset: len("a\n\n"), RuneOffset: 3, GraphemeOffset: 3},
	}
	got := layout.SelectionRanges(selection)
	want := []TextSelectionRange{{Row: 1, Col: 0, Width: 1}}
	if !sameSelectionRanges(got, want) {
		t.Fatalf("ranges = %#v, want %#v", got, want)
	}
}

func TestTextLayoutSelectionRangesIncludeWideCells(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "a界b"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	selection := TextSelection{
		Base:   TextPosition{Span: 0, ByteOffset: 1, RuneOffset: 1, GraphemeOffset: 1},
		Extent: TextPosition{Span: 0, ByteOffset: len("a界"), RuneOffset: 2, GraphemeOffset: 2},
	}
	got := layout.SelectionRanges(selection)
	want := []TextSelectionRange{{Row: 0, Col: 1, Width: 2}}
	if !sameSelectionRanges(got, want) {
		t.Fatalf("ranges = %#v, want %#v", got, want)
	}
}

func TestTextLayoutSelectionRangesIgnoreCollapsedSelections(t *testing.T) {
	layout := LayoutText([]TextSpan{{Text: "abc"}}, Constraints{MaxWidth: 10, MaxHeight: 1}, TextLayoutOptions{})
	selection := TextSelection{Base: layout.Lines[0].Start, Extent: layout.Lines[0].Start}
	if got := layout.SelectionRanges(selection); len(got) != 0 {
		t.Fatalf("ranges = %#v, want none", got)
	}
}

func sameSelectionRanges(a, b []TextSelectionRange) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
