package ui

import "testing"

func TestDryLayoutTextComputesHeightFromWidth(t *testing.T) {
	app := NewApp(Text{Value: "hello world", SoftWrap: true})
	ro := findRenderObject(app.build.Root()).(*renderText)
	size := DryLayout(LayoutContext{}, ro, Constraints{MaxWidth: 5, MaxHeight: Unbounded})
	if size != (Size{Width: 5, Height: 2}) {
		t.Fatalf("dry text size = %#v, want 5x2", size)
	}
	if ro.Size() != (Size{}) {
		t.Fatalf("dry layout mutated render size to %#v", ro.Size())
	}
}

func TestDryLayoutMatchesLayoutThroughWrappers(t *testing.T) {
	app := NewApp(Padding(All(1), ConstrainedBox{
		Constraints: Constraints{MaxWidth: 5},
		Child:       Text{Value: "hello world", SoftWrap: true},
	}))
	ro := findRenderObject(app.build.Root())
	constraints := Constraints{MaxWidth: 7, MaxHeight: Unbounded}
	dry := DryLayout(LayoutContext{}, ro, constraints)
	ro.Layout(LayoutContext{}, constraints)
	if got := ro.Base().Size(); got != dry {
		t.Fatalf("layout size = %#v, dry = %#v", got, dry)
	}
}

func TestDryLayoutDoesNotLayoutChildren(t *testing.T) {
	child := &dryRecordingRenderObject{drySize: Size{Width: 3, Height: 2}}
	app := NewApp(Padding(All(1), dryRecordingWidget{RO: child}))
	ro := findRenderObject(app.build.Root())
	size := DryLayout(LayoutContext{}, ro, Constraints{MaxWidth: 10, MaxHeight: 10})
	if size != (Size{Width: 5, Height: 4}) {
		t.Fatalf("dry size = %#v, want 5x4", size)
	}
	if child.layouts != 0 {
		t.Fatalf("child layouts = %d, want 0", child.layouts)
	}
	if child.dryLayouts != 1 {
		t.Fatalf("child dry layouts = %d, want 1", child.dryLayouts)
	}
}

func TestDryLayoutFlexMatchesLayout(t *testing.T) {
	app := NewApp(Row(
		Text{Value: "x"},
		Expanded(Text{Value: "wrapped text", SoftWrap: true}),
	))
	ro := findRenderObject(app.build.Root())
	constraints := Tight(Size{Width: 8, Height: 4})
	dry := DryLayout(LayoutContext{}, ro, constraints)
	ro.Layout(LayoutContext{}, constraints)
	if got := ro.Base().Size(); got != dry {
		t.Fatalf("layout size = %#v, dry = %#v", got, dry)
	}
}

type dryRecordingRenderObject struct {
	LeafRenderObject
	drySize    Size
	layouts    int
	dryLayouts int
}

type dryRecordingWidget struct {
	RO *dryRecordingRenderObject
}

func (w dryRecordingWidget) CreateRenderObject(BuildContext) RenderObject {
	return w.RO
}

func (w dryRecordingWidget) UpdateRenderObject(BuildContext, RenderObject) {
}

func (r *dryRecordingRenderObject) Layout(_ LayoutContext, c Constraints) {
	r.layouts++
	r.SetSize(c.Constrain(r.drySize))
}

func (r *dryRecordingRenderObject) DryLayout(_ LayoutContext, c Constraints) Size {
	r.dryLayouts++
	return c.Constrain(r.drySize)
}

func (r *dryRecordingRenderObject) Paint(*Painter, Offset) {
}
