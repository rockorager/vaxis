package ui

import "testing"

type lifecycleWidget struct {
	Name string
	Key  KeyValue
	Log  *[]string
}

func (w lifecycleWidget) WidgetKey() KeyValue {
	return w.Key
}

func (w lifecycleWidget) CreateState() State {
	return &lifecycleState{log: w.Log}
}

type lifecycleState struct {
	StateBase
	log *[]string
}

func (s *lifecycleState) InitState() {
	*s.log = append(*s.log, "init:"+s.Widget().(lifecycleWidget).Name)
}

func (s *lifecycleState) DidUpdateWidget(old Widget) {
	*wlog(old) = append(*s.log, "update-old:"+old.(lifecycleWidget).Name+":new:"+s.Widget().(lifecycleWidget).Name)
}

func (s *lifecycleState) Dispose() {
	*s.log = append(*s.log, "dispose:"+s.Widget().(lifecycleWidget).Name)
}

func (s *lifecycleState) Build(BuildContext) Widget {
	return Text{Value: s.Widget().(lifecycleWidget).Name}
}

func wlog(w Widget) *[]string {
	return w.(lifecycleWidget).Log
}

func TestStateLifecycleUpdateReceivesOldWidget(t *testing.T) {
	var log []string
	app := NewApp(lifecycleWidget{Name: "one", Log: &log})
	app.Pump(Size{Width: 10, Height: 1})
	app.UpdateRoot(lifecycleWidget{Name: "two", Log: &log})
	app.Pump(Size{Width: 10, Height: 1})
	want := []string{"init:one", "update-old:one:new:two"}
	if !sameStrings(log, want) {
		t.Fatalf("log = %#v, want %#v", log, want)
	}
}

func TestStateLifecycleKeyChangeDisposesAndRecreates(t *testing.T) {
	var log []string
	app := NewApp(lifecycleWidget{Name: "one", Key: "a", Log: &log})
	app.Pump(Size{Width: 10, Height: 1})
	app.UpdateRoot(lifecycleWidget{Name: "two", Key: "b", Log: &log})
	app.Pump(Size{Width: 10, Height: 1})
	want := []string{"init:one", "dispose:one", "init:two"}
	if !sameStrings(log, want) {
		t.Fatalf("log = %#v, want %#v", log, want)
	}
}

type buildCounterWidget struct{ Builds *int }

func (w buildCounterWidget) CreateState() State {
	return &buildCounterState{builds: w.Builds}
}

type buildCounterState struct {
	StateBase
	builds *int
}

func (s *buildCounterState) Build(BuildContext) Widget {
	(*s.builds)++
	return Text{Value: "x"}
}

func TestMultipleMarkNeedsBuildCallsCoalesce(t *testing.T) {
	builds := 0
	app := NewApp(buildCounterWidget{Builds: &builds})
	app.Pump(Size{Width: 1, Height: 1})
	if builds != 1 {
		t.Fatalf("initial builds = %d, want 1", builds)
	}
	state := findState[*buildCounterState](app.build.Root())
	state.MarkNeedsBuild()
	state.MarkNeedsBuild()
	app.Pump(Size{Width: 1, Height: 1})
	if builds != 2 {
		t.Fatalf("builds after two dirty marks = %d, want 2", builds)
	}
}

func TestMarkNeedsBuildAfterDisposePanics(t *testing.T) {
	builds := 0
	app := NewApp(buildCounterWidget{Builds: &builds})
	app.Pump(Size{Width: 1, Height: 1})
	state := findState[*buildCounterState](app.build.Root())
	app.UpdateRoot(Text{Value: "gone"})
	app.Pump(Size{Width: 1, Height: 1})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MarkNeedsBuild after dispose did not panic")
		}
	}()
	state.MarkNeedsBuild()
}

type dirtyParentWidget struct {
	childName string
	childLog  *[]string
}

func (w dirtyParentWidget) CreateState() State {
	return &dirtyParentState{}
}

type dirtyParentState struct {
	StateBase
	childName string
}

func (s *dirtyParentState) InitState() {
	s.childName = s.Widget().(dirtyParentWidget).childName
}

func (s *dirtyParentState) SetChildName(name string) {
	s.SetState(func() { s.childName = name })
}

func (s *dirtyParentState) Build(BuildContext) Widget {
	w := s.Widget().(dirtyParentWidget)
	if s.childName == "" {
		return Text{Value: "empty"}
	}
	return lifecycleWidget{Name: s.childName, Log: w.childLog}
}

func TestDirtyParentUpdatesDirtyChildOnce(t *testing.T) {
	var log []string
	app := NewApp(dirtyParentWidget{childName: "one", childLog: &log})
	app.Pump(Size{Width: 10, Height: 1})
	parent := findState[*dirtyParentState](app.build.Root())
	child := findState[*lifecycleState](app.build.Root())

	child.MarkNeedsBuild()
	parent.SetChildName("two")
	app.Pump(Size{Width: 10, Height: 1})

	want := []string{"init:one", "update-old:one:new:two"}
	if !sameStrings(log, want) {
		t.Fatalf("log = %#v, want %#v", log, want)
	}
}

func TestDirtyParentRemovesDirtyChild(t *testing.T) {
	var log []string
	app := NewApp(dirtyParentWidget{childName: "one", childLog: &log})
	app.Pump(Size{Width: 10, Height: 1})
	parent := findState[*dirtyParentState](app.build.Root())
	child := findState[*lifecycleState](app.build.Root())

	child.MarkNeedsBuild()
	parent.SetChildName("")
	app.Pump(Size{Width: 10, Height: 1})

	want := []string{"init:one", "dispose:one"}
	if !sameStrings(log, want) {
		t.Fatalf("log = %#v, want %#v", log, want)
	}
}

type dirtyChildStats struct {
	builds  int
	updates int
	dispose int
}

type dirtyCountingParentWidget struct {
	childName string
	stats     *dirtyChildStats
}

func (w dirtyCountingParentWidget) CreateState() State {
	return &dirtyCountingParentState{}
}

type dirtyCountingParentState struct {
	StateBase
	childName string
}

func (s *dirtyCountingParentState) InitState() {
	s.childName = s.Widget().(dirtyCountingParentWidget).childName
}

func (s *dirtyCountingParentState) SetChildName(name string) {
	s.SetState(func() { s.childName = name })
}

func (s *dirtyCountingParentState) Build(BuildContext) Widget {
	w := s.Widget().(dirtyCountingParentWidget)
	if s.childName == "" {
		return Text{Value: "empty"}
	}
	return dirtyCountingChildWidget{Name: s.childName, Stats: w.stats}
}

type dirtyCountingChildWidget struct {
	Name  string
	Stats *dirtyChildStats
}

func (w dirtyCountingChildWidget) CreateState() State {
	return &dirtyCountingChildState{}
}

type dirtyCountingChildState struct {
	StateBase
}

func (s *dirtyCountingChildState) DidUpdateWidget(Widget) {
	s.Widget().(dirtyCountingChildWidget).Stats.updates++
}

func (s *dirtyCountingChildState) Dispose() {
	s.Widget().(dirtyCountingChildWidget).Stats.dispose++
}

func (s *dirtyCountingChildState) Build(BuildContext) Widget {
	w := s.Widget().(dirtyCountingChildWidget)
	w.Stats.builds++
	return Text{Value: w.Name}
}

func TestDirtyParentRebuildsDirtyChildOnce(t *testing.T) {
	stats := &dirtyChildStats{}
	app := NewApp(dirtyCountingParentWidget{childName: "one", stats: stats})
	app.Pump(Size{Width: 10, Height: 1})
	parent := findState[*dirtyCountingParentState](app.build.Root())
	child := findState[*dirtyCountingChildState](app.build.Root())

	child.MarkNeedsBuild()
	parent.SetChildName("two")
	app.Pump(Size{Width: 10, Height: 1})

	if stats.builds != 2 || stats.updates != 1 || stats.dispose != 0 {
		t.Fatalf("stats = %#v, want 2 builds, 1 update, 0 dispose", stats)
	}
}

func TestDirtyParentDoesNotRebuildRemovedDirtyChild(t *testing.T) {
	stats := &dirtyChildStats{}
	app := NewApp(dirtyCountingParentWidget{childName: "one", stats: stats})
	app.Pump(Size{Width: 10, Height: 1})
	parent := findState[*dirtyCountingParentState](app.build.Root())
	child := findState[*dirtyCountingChildState](app.build.Root())

	child.MarkNeedsBuild()
	parent.SetChildName("")
	app.Pump(Size{Width: 10, Height: 1})

	if stats.builds != 1 || stats.updates != 0 || stats.dispose != 1 {
		t.Fatalf("stats = %#v, want 1 build, 0 updates, 1 dispose", stats)
	}
}

func findState[T any](root element) T {
	var zero T
	var found T
	var walk func(element) bool
	walk = func(e element) bool {
		if se, ok := e.(*statefulElement); ok {
			if state, ok := se.state.(T); ok {
				found = state
				return true
			}
		}
		done := false
		e.VisitChildren(func(child element) {
			if !done && walk(child) {
				done = true
			}
		})
		return done
	}
	if walk(root) {
		return found
	}
	return zero
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

type testRenderWidget struct {
	RO   *testRenderObject
	Kids []Widget
}

func (w testRenderWidget) CreateRenderObject(BuildContext) RenderObject {
	return w.RO
}

func (w testRenderWidget) UpdateRenderObject(BuildContext, RenderObject) {
}

func (w testRenderWidget) Children() []Widget {
	return w.Kids
}

type testRenderObject struct{ MultiChildRenderObject }

func (r *testRenderObject) Layout(_ LayoutContext, c Constraints) {
	r.SetSize(c.Constrain(Size{Width: 1, Height: 1}))
	for _, child := range r.Children() {
		child.Layout(LayoutContext{}, c)
	}
}

func (r *testRenderObject) Paint(*Painter, Offset) {
}

func (r *testRenderObject) HitTest(*HitTestResult, Point) bool {
	return true
}

type testLeafWidget struct{ RO *testLeafRenderObject }

func (w testLeafWidget) CreateRenderObject(BuildContext) RenderObject {
	return w.RO
}

func (w testLeafWidget) UpdateRenderObject(BuildContext, RenderObject) {
}

type testLeafRenderObject struct{ LeafRenderObject }

func (r *testLeafRenderObject) Layout(_ LayoutContext, c Constraints) {
	r.SetSize(c.Constrain(Size{Width: 1, Height: 1}))
}

func (r *testLeafRenderObject) Paint(*Painter, Offset) {
}

func TestRenderChildrenDetachWhenRemoved(t *testing.T) {
	parent := &testRenderObject{}
	child := &testLeafRenderObject{}
	app := NewApp(testRenderWidget{RO: parent, Kids: []Widget{testLeafWidget{RO: child}}})
	app.Pump(Size{Width: 5, Height: 1})
	if child.Base().owner != app || child.Base().parent != parent {
		t.Fatal("child render object was not attached")
	}
	app.UpdateRoot(testRenderWidget{RO: parent})
	app.Pump(Size{Width: 5, Height: 1})
	if child.Base().owner != nil || child.Base().parent != nil {
		t.Fatalf("removed child owner/parent = %v/%v, want nil/nil", child.Base().owner, child.Base().parent)
	}
}

func TestRenderDirtyFlagsClearAfterPumpAndPaint(t *testing.T) {
	ro := &testLeafRenderObject{}
	app := NewApp(testLeafWidget{RO: ro})
	app.Pump(Size{Width: 5, Height: 1})
	ro.MarkNeedsLayout()
	if !ro.NeedsLayout() || !ro.NeedsPaint() || !app.FrameRequested() {
		t.Fatal("MarkNeedsLayout should dirty layout, paint, and request frame")
	}
	app.Pump(Size{Width: 5, Height: 1})
	if ro.NeedsLayout() {
		t.Fatal("layout dirty flag should clear after pump")
	}
	app.Paint(NewPainter(Size{Width: 5, Height: 1}))
	if ro.NeedsPaint() {
		t.Fatal("paint dirty flag should clear after paint")
	}
}

func TestRenderInvalidationBubblesUnlessRelayoutBoundary(t *testing.T) {
	parent := &testRenderObject{}
	child := &testLeafRenderObject{}
	app := NewApp(testRenderWidget{RO: parent, Kids: []Widget{testLeafWidget{RO: child}}})
	app.Pump(Size{Width: 5, Height: 1})
	child.MarkNeedsLayout()
	if !child.NeedsLayout() || !parent.NeedsLayout() {
		t.Fatal("child layout invalidation should bubble to parent")
	}
	app.Pump(Size{Width: 5, Height: 1})
	parent.ClearNeedsLayout()
	parent.ClearNeedsPaint()
	child.ClearNeedsLayout()
	child.ClearNeedsPaint()
	child.SetRelayoutBoundary(true)
	child.MarkNeedsLayout()
	if !child.NeedsLayout() {
		t.Fatal("child should be dirty")
	}
	if parent.NeedsLayout() {
		t.Fatal("relayout boundary should stop layout invalidation")
	}
}

func TestMarkNeedsPaintDoesNotDirtyLayout(t *testing.T) {
	ro := &testLeafRenderObject{}
	app := NewApp(testLeafWidget{RO: ro})
	app.Pump(Size{Width: 5, Height: 1})
	ro.MarkNeedsPaint()
	if ro.NeedsLayout() {
		t.Fatal("MarkNeedsPaint should not dirty layout")
	}
	if !ro.NeedsPaint() || !app.FrameRequested() {
		t.Fatal("MarkNeedsPaint should dirty paint and request frame")
	}
}

func TestDetachedRenderObjectDoesNotRequestFrame(t *testing.T) {
	parent := &testRenderObject{}
	child := &testLeafRenderObject{}
	app := NewApp(testRenderWidget{RO: parent, Kids: []Widget{testLeafWidget{RO: child}}})
	app.Pump(Size{Width: 5, Height: 1})
	app.UpdateRoot(testRenderWidget{RO: parent})
	app.Pump(Size{Width: 5, Height: 1})
	child.MarkNeedsPaint()
	if app.FrameRequested() {
		t.Fatal("detached render object should not request frames")
	}
}
