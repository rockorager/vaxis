package ui_test

import (
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

func TestFrameSchedulerSchedulesImmediatelyWhenIdle(t *testing.T) {
	now := time.Unix(10, 0)
	s := ui.NewFrameScheduler(time.Second / 60)
	if due := s.Request(now); !due.Equal(now) {
		t.Fatalf("due = %v, want immediate %v", due, now)
	}
}

func TestFrameSchedulerDelaysUntilNextFrameInterval(t *testing.T) {
	interval := 16 * time.Millisecond
	now := time.Unix(10, 0)
	s := ui.NewFrameScheduler(interval)
	s.Request(now)
	s.DidFrame(now)
	if due := s.Request(now.Add(5 * time.Millisecond)); !due.Equal(now.Add(interval)) {
		t.Fatalf("due = %v, want %v", due, now.Add(interval))
	}
}

func TestFrameSchedulerCoalescesRequests(t *testing.T) {
	now := time.Unix(10, 0)
	s := ui.NewFrameScheduler(time.Second / 60)
	first := s.Request(now)
	second := s.Request(now.Add(time.Second))
	if !second.Equal(first) {
		t.Fatalf("second due = %v, want coalesced %v", second, first)
	}
}

func TestIgnoredEventDoesNotRequestFrame(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "x"})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.Send(vaxis.Key{Keycode: 'x'})
	if app.FrameRequested() {
		t.Fatal("ignored event should not request frame")
	}
}

func TestSetStateRequestsFrame(t *testing.T) {
	state := &frameState{}
	app := ui.NewApp(frameWidget{state: state})
	app.Pump(ui.Size{Width: 1, Height: 1})
	state.SetState(func() { state.value = "b" })
	if !app.FrameRequested() {
		t.Fatal("SetState should request frame")
	}
}

func TestMarkNeedsPaintRequestsFrame(t *testing.T) {
	ro := &frameRender{}
	app := ui.NewApp(frameRenderWidget{render: ro})
	app.Pump(ui.Size{Width: 1, Height: 1})
	ro.MarkNeedsPaint()
	if !app.FrameRequested() {
		t.Fatal("MarkNeedsPaint should request frame")
	}
}

func TestMarkNeedsLayoutBubblesToParent(t *testing.T) {
	child := &frameRender{}
	parent := &frameParentRender{}
	app := ui.NewApp(frameParentWidget{parent: parent, child: child})
	app.Pump(ui.Size{Width: 1, Height: 1})
	child.MarkNeedsLayout()
	if !app.FrameRequested() {
		t.Fatal("MarkNeedsLayout should request frame")
	}
	if !parent.NeedsLayout() {
		t.Fatal("child layout dirtiness should bubble to parent")
	}
}

func TestRelayoutBoundaryStopsLayoutBubbling(t *testing.T) {
	child := &frameRender{}
	parent := &frameParentRender{}
	app := ui.NewApp(frameParentWidget{parent: parent, child: child})
	app.Pump(ui.Size{Width: 1, Height: 1})
	child.SetRelayoutBoundary(true)
	child.MarkNeedsLayout()
	if !app.FrameRequested() {
		t.Fatal("MarkNeedsLayout should request frame even at relayout boundary")
	}
	if parent.NeedsLayout() {
		t.Fatal("relayout boundary should stop layout dirtiness from bubbling")
	}
}

func TestPumpClearsNeedsLayout(t *testing.T) {
	ro := &frameRender{}
	app := ui.NewApp(frameRenderWidget{render: ro})
	app.Pump(ui.Size{Width: 1, Height: 1})
	ro.MarkNeedsLayout()
	if !ro.NeedsLayout() {
		t.Fatal("expected render object to need layout")
	}
	app.Pump(ui.Size{Width: 1, Height: 1})
	if ro.NeedsLayout() {
		t.Fatal("Pump should clear layout dirtiness after layout")
	}
}

func TestPaintClearsNeedsPaint(t *testing.T) {
	ro := &frameRender{}
	app := ui.NewApp(frameRenderWidget{render: ro})
	app.Pump(ui.Size{Width: 1, Height: 1})
	ro.MarkNeedsPaint()
	if !ro.NeedsPaint() {
		t.Fatal("expected render object to need paint")
	}
	app.Paint(ui.NewPainter(ui.Size{Width: 1, Height: 1}))
	if ro.NeedsPaint() {
		t.Fatal("Paint should clear paint dirtiness after painting")
	}
}

type frameWidget struct{ state *frameState }

func (w frameWidget) CreateState() ui.State { return w.state }

type frameState struct {
	ui.StateBase
	value string
}

func (s *frameState) Build(ctx ui.BuildContext) ui.Widget {
	if s.value == "" {
		s.value = "a"
	}
	return (ui.Text{Value: s.value})
}

type frameRenderWidget struct{ render *frameRender }

func (w frameRenderWidget) CreateRenderObject(ctx ui.BuildContext) ui.RenderObject     { return w.render }
func (w frameRenderWidget) UpdateRenderObject(ctx ui.BuildContext, ro ui.RenderObject) {}

type frameRender struct{ ui.LeafRenderObject }

func (r *frameRender) Layout(ctx ui.LayoutContext, c ui.Constraints) {
	r.SetSize(c.Constrain(ui.Size{Width: 1, Height: 1}))
}
func (r *frameRender) Paint(p *ui.Painter, off ui.Offset) {}

type frameParentWidget struct {
	parent *frameParentRender
	child  *frameRender
}

func (w frameParentWidget) Child() ui.Widget                                           { return frameRenderWidget{render: w.child} }
func (w frameParentWidget) CreateRenderObject(ctx ui.BuildContext) ui.RenderObject     { return w.parent }
func (w frameParentWidget) UpdateRenderObject(ctx ui.BuildContext, ro ui.RenderObject) {}

type frameParentRender struct{ ui.SingleChildRenderObject }

func (r *frameParentRender) Layout(ctx ui.LayoutContext, c ui.Constraints) {
	if child := r.Child(); child != nil {
		child.Layout(ctx, c)
	}
	r.SetSize(c.Constrain(ui.Size{Width: 1, Height: 1}))
}
func (r *frameParentRender) Paint(p *ui.Painter, off ui.Offset)       {}
func (r *frameParentRender) HitTest(*ui.HitTestResult, ui.Point) bool { return false }
