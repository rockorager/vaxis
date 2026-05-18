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
