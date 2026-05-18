package ui

import "testing"

func TestFlexMainAxisAlignmentPositionsChildren(t *testing.T) {
	tests := []struct {
		name      string
		align     MainAxisAlignment
		wantFirst int
		wantLast  int
	}{
		{name: "start", align: MainAxisStart, wantFirst: 0, wantLast: 1},
		{name: "end", align: MainAxisEnd, wantFirst: 8, wantLast: 9},
		{name: "center", align: MainAxisCenter, wantFirst: 4, wantLast: 5},
		{name: "space between", align: MainAxisSpaceBetween, wantFirst: 0, wantLast: 9},
		{name: "space evenly", align: MainAxisSpaceEvenly, wantFirst: 2, wantLast: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(Flex{Axis: Horizontal, MainAxisAlignment: tt.align, ChildrenWidget: []Widget{Text{Value: "a"}, Text{Value: "b"}}})
			app.Pump(Size{Width: 10, Height: 1})
			p := NewPainter(Size{Width: 10, Height: 1})
			app.Paint(p)
			if got := p.Cell(tt.wantFirst, 0).Grapheme; got != "a" {
				t.Fatalf("first child at %d = %q, want a", tt.wantFirst, got)
			}
			if got := p.Cell(tt.wantLast, 0).Grapheme; got != "b" {
				t.Fatalf("last child at %d = %q, want b", tt.wantLast, got)
			}
		})
	}
}

func TestFlexCrossAxisAlignmentPositionsChildren(t *testing.T) {
	tests := []struct {
		name  string
		align CrossAxisAlignment
		wantY int
	}{
		{name: "start", align: CrossAxisStart, wantY: 0},
		{name: "center", align: CrossAxisCenter, wantY: 2},
		{name: "end", align: CrossAxisEnd, wantY: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(Flex{Axis: Horizontal, CrossAxisAlignment: tt.align, ChildrenWidget: []Widget{Text{Value: "x"}}})
			app.Pump(Size{Width: 1, Height: 5})
			p := NewPainter(Size{Width: 1, Height: 5})
			app.Paint(p)
			if got := p.Cell(0, tt.wantY).Grapheme; got != "x" {
				t.Fatalf("child at y=%d = %q, want x", tt.wantY, got)
			}
		})
	}
}

func TestFlexDefaultsMatchFlutter(t *testing.T) {
	app := NewApp(Row(Text{Value: "x"}))
	app.Pump(Size{Width: 1, Height: 5})
	p := NewPainter(Size{Width: 1, Height: 5})
	app.Paint(p)
	if got := p.Cell(0, 2).Grapheme; got != "x" {
		t.Fatalf("default row cross-axis position = %q at center, want x", got)
	}
	row := findRenderObject(app.build.Root()).(*RenderFlex)
	if row.Size().Width != 1 || row.Size().Height != 5 {
		t.Fatalf("default row size = %dx%d, want max constraints 1x5", row.Size().Width, row.Size().Height)
	}

	child := &recordingRenderObject{desired: Size{Width: 1, Height: 1}}
	app = NewApp(Row(Expanded(recordingWidget{RO: child})))
	app.Pump(Size{Width: 7, Height: 3})
	if child.Size().Width != 7 {
		t.Fatalf("Expanded default fit width = %d, want tight 7", child.Size().Width)
	}

	loose := &recordingRenderObject{desired: Size{Width: 2, Height: 1}}
	app = NewApp(Row(Flexible(recordingWidget{RO: loose})))
	app.Pump(Size{Width: 7, Height: 3})
	if loose.Size().Width != 2 {
		t.Fatalf("Flexible default fit width = %d, want loose 2", loose.Size().Width)
	}
}

func TestFlexExpandedDistributesAllRemainingSpace(t *testing.T) {
	left := &recordingRenderObject{}
	right := &recordingRenderObject{}
	app := NewApp(Row(
		ExpandedWidget{Flex: 1, ChildWidget: recordingWidget{RO: left}},
		ExpandedWidget{Flex: 2, ChildWidget: recordingWidget{RO: right}},
	))
	app.Pump(Size{Width: 10, Height: 1})
	if left.Size().Width != 3 || right.Size().Width != 7 {
		t.Fatalf("expanded widths = %d/%d, want 3/7", left.Size().Width, right.Size().Width)
	}
	if pd := right.ParentData().(FlexParentData); pd.Offset.X != 3 {
		t.Fatalf("right offset = %d, want 3", pd.Offset.X)
	}
}

func TestFlexibleLooseCanUseLessThanAllocatedSpace(t *testing.T) {
	loose := &recordingRenderObject{desired: Size{Width: 2, Height: 1}}
	app := NewApp(Row(Flexible(recordingWidget{RO: loose})))
	app.Pump(Size{Width: 10, Height: 1})
	if loose.Size().Width != 2 {
		t.Fatalf("loose flexible width = %d, want 2", loose.Size().Width)
	}
}

func TestFlexCrossAxisStretchTightensCrossAxis(t *testing.T) {
	child := &recordingRenderObject{desired: Size{Width: 1, Height: 1}}
	app := NewApp(Flex{Axis: Horizontal, CrossAxisAlignment: CrossAxisStretch, ChildrenWidget: []Widget{recordingWidget{RO: child}}})
	app.Pump(Size{Width: 5, Height: 4})
	if child.Size().Height != 4 {
		t.Fatalf("stretched child height = %d, want 4", child.Size().Height)
	}
}

func TestFlexMainAxisSizeMinShrinksToChildren(t *testing.T) {
	app := NewApp(Align{Alignment: TopLeft, Child: Flex{Axis: Horizontal, MainAxisSize: MainAxisSizeMin, ChildrenWidget: []Widget{Text{Value: "a"}, Text{Value: "bc"}}}})
	app.Pump(Size{Width: 10, Height: 1})
	row := findRenderObject(app.build.Root()).(*RenderAlign).Child().(*RenderFlex)
	if row.Size().Width != 3 {
		t.Fatalf("row width = %d, want 3", row.Size().Width)
	}
}

type recordingWidget struct{ RO *recordingRenderObject }

func (w recordingWidget) CreateRenderObject(BuildContext) RenderObject {
	return w.RO
}

func (w recordingWidget) UpdateRenderObject(BuildContext, RenderObject) {
}

type recordingRenderObject struct {
	LeafRenderObject
	desired Size
}

func (r *recordingRenderObject) Layout(_ LayoutContext, c Constraints) {
	desired := r.desired
	if desired == (Size{}) {
		desired = Size{Width: c.MaxWidth, Height: max(1, c.MinHeight)}
	}
	r.SetSize(c.Constrain(desired))
}

func (r *recordingRenderObject) Paint(*Painter, Offset) {
}
