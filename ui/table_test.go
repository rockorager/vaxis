package ui

import "testing"

func TestTableIntrinsicColumnsAlignRows(t *testing.T) {
	app := NewApp(Table{
		ColumnGap: 1,
		Rows: []TableRow{
			{Children: []Widget{Text{Value: "name"}, Text{Value: "state"}}},
			{Children: []Widget{Text{Value: "build"}, Text{Value: "running"}}},
		},
	})
	app.Pump(Size{Width: 20, Height: 2})
	p := NewPainter(Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "n" {
		t.Fatalf("first row first column = %q, want n", got)
	}
	if got := p.Cell(6, 0).Grapheme; got != "s" {
		t.Fatalf("first row second column at x=6 = %q, want s", got)
	}
	if got := p.Cell(6, 1).Grapheme; got != "r" {
		t.Fatalf("second row second column at x=6 = %q, want r", got)
	}
}

func TestTableFlexColumnsShareRemainingWidth(t *testing.T) {
	left := &recordingRenderObject{}
	right := &recordingRenderObject{}
	app := NewApp(Table{
		Columns: []TableColumn{FixedColumn(2), FlexColumn(1), FlexColumn(2)},
		Rows: []TableRow{{Children: []Widget{
			Text{Value: "id"},
			recordingWidget{RO: left},
			recordingWidget{RO: right},
		}}},
	})
	app.Pump(Size{Width: 11, Height: 1})
	if left.Size().Width != 3 || right.Size().Width != 6 {
		t.Fatalf("flex widths = %d/%d, want 3/6", left.Size().Width, right.Size().Width)
	}
	if pd := right.ParentData().(TableParentData); pd.Offset.X != 5 {
		t.Fatalf("right offset x = %d, want 5", pd.Offset.X)
	}
}

func TestTableRowGapAndHeight(t *testing.T) {
	tall := &recordingRenderObject{desired: Size{Width: 1, Height: 2}}
	app := NewApp(Table{
		Columns: []TableColumn{FixedColumn(1)},
		RowGap:  1,
		Rows: []TableRow{
			{Children: []Widget{recordingWidget{RO: tall}}},
			{Children: []Widget{Text{Value: "x"}}},
		},
	})
	app.Pump(Size{Width: 5, Height: 5})
	p := NewPainter(Size{Width: 5, Height: 5})
	app.Paint(p)
	if got := p.Cell(0, 3).Grapheme; got != "x" {
		t.Fatalf("second row y = %q, want x at y=3", got)
	}
}
