package ui

import (
	"strconv"
	"testing"
)

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

func TestSliverTableBuilderBuildsBoundedInitialRange(t *testing.T) {
	built := map[int]bool{}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverTableBuilder{
			RowCount:           1000,
			EstimatedRowExtent: 1,
			Builder: func(ctx BuildContext, row int) TableRow {
				built[row] = true
				return TableRow{Children: []Widget{Text{Value: "row"}}}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 4})

	if len(built) == 0 || len(built) > defaultSliverListBuilderInitialCount {
		t.Fatalf("built %d rows, want a bounded initial range", len(built))
	}
	if built[999] {
		t.Fatal("builder eagerly built the last row")
	}
}

func TestSliverTableBuilderAlignsColumnsAndScrollsRows(t *testing.T) {
	controller := &SliverTableController{}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverTableBuilder{
			Controller:         controller,
			Columns:            []TableColumn{IntrinsicColumn(), FixedColumn(1), FlexColumn(1)},
			RowCount:           20,
			EstimatedRowExtent: 1,
			Builder: func(ctx BuildContext, row int) TableRow {
				return TableRow{Children: []Widget{
					Text{Value: strconv.Itoa(row)},
					Text{Value: "|"},
					Text{Value: "code " + strconv.Itoa(row)},
				}}
			},
		},
	}})
	app.Pump(Size{Width: 12, Height: 3})
	app.Pump(Size{Width: 12, Height: 3})
	if !controller.ScrollToRow(10, ScrollAlignStart) {
		t.Fatal("ScrollToRow returned false")
	}
	app.Pump(Size{Width: 12, Height: 3})
	app.Pump(Size{Width: 12, Height: 3})

	p := NewPainter(Size{Width: 12, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme + p.Cell(1, 0).Grapheme; got != "10" {
		t.Fatalf("first visible row = %q, want 10", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "|" {
		t.Fatalf("separator column = %q, want |", got)
	}
	if got := p.Cell(3, 0).Grapheme; got != "c" {
		t.Fatalf("code column start = %q, want c", got)
	}
}

func TestSliverTableBuilderVariableHeightsUpdateMetrics(t *testing.T) {
	heights := []int{1, 3, 2, 1}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverTableBuilder{
			Columns:            []TableColumn{FlexColumn(1)},
			RowCount:           len(heights),
			EstimatedRowExtent: 1,
			Builder: func(ctx BuildContext, row int) TableRow {
				return TableRow{Children: []Widget{SizedBox{Width: 10, Height: heights[row], Child: Text{Value: strconv.Itoa(row)}}}}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	got := r.ScrollMetrics()
	want := ScrollMetrics{MaxScrollOffset: 4, ViewportHeight: 3, ViewportWidth: 10, ContentHeight: 7}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestSliverTableBuilderAnchorsVisibleRowOnResize(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverTableBuilder{
			Columns:            []TableColumn{FixedColumn(4), FlexColumn(1)},
			RowCount:           20,
			EstimatedRowExtent: 1,
			Overscan:           3,
			Builder: func(ctx BuildContext, row int) TableRow {
				return TableRow{Children: []Widget{
					Text{Value: "row "},
					Text{Value: padTestInt(row, 2) + " abcdefghij", SoftWrap: true},
				}}
			},
		},
	}})
	app.Pump(Size{Width: 20, Height: 3})
	app.Pump(Size{Width: 20, Height: 3})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	r.ScrollToOffset(6)
	app.Pump(Size{Width: 20, Height: 3})
	app.Pump(Size{Width: 20, Height: 3})

	app.Pump(Size{Width: 10, Height: 3})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme + p.Cell(5, 0).Grapheme; got != "06" {
		t.Fatalf("first visible row suffix after resize = %q, want row 06", got)
	}
}
