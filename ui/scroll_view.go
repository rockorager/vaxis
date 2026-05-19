package ui

// ScrollView clips a single child to a vertical viewport and scrolls it.
//
// When used inside SelectionArea, selections that start outside the ScrollView
// include hidden rows, while selections that start inside it initially use the
// visible rows and expand as selection autoscroll moves the viewport.
type ScrollView struct {
	// Child is laid out at the viewport width with unbounded height.
	Child Widget
}

func (w ScrollView) CreateState() State {
	return &scrollViewState{}
}

type scrollViewState struct {
	StateBase
	node      FocusNode
	scrollRow int
}

func (s *scrollViewState) Build(BuildContext) Widget {
	return Focus(&s.node, scrollViewViewport{
		State: s,
		Child: s.Widget().(ScrollView).Child,
	})
}

func (s *scrollViewState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if keyIsRelease(ev) {
			return EventIgnored
		}
		switch ev.Keycode {
		case KeyPgUp:
			return s.scrollBy(-s.pageSize())
		case KeyPgDown:
			return s.scrollBy(s.pageSize())
		case KeyHome:
			return s.scrollTo(0)
		case KeyEnd:
			return s.scrollTo(s.maxScroll())
		}
	case Mouse:
		if ev.EventType != EventPress {
			return EventIgnored
		}
		switch ev.Button {
		case MouseWheelUp:
			return s.scrollBy(-1)
		case MouseWheelDown:
			return s.scrollBy(1)
		}
	}
	return EventIgnored
}

func (s *scrollViewState) scrollBy(delta int) EventResult {
	return s.scrollTo(s.scrollRow + delta)
}

func (s *scrollViewState) scrollTo(row int) EventResult {
	next := clampInt(row, 0, s.maxScroll())
	if next == s.scrollRow {
		return EventHandled
	}
	s.SetState(func() { s.scrollRow = next })
	return EventHandled
}

func (s *scrollViewState) pageSize() int {
	if r := s.renderObject(); r != nil {
		return max(1, r.Size().Height)
	}
	return 1
}

func (s *scrollViewState) maxScroll() int {
	if r := s.renderObject(); r != nil {
		return r.maxScroll()
	}
	return 0
}

func (s *scrollViewState) renderObject() *renderScrollView {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderScrollView); ok {
		return r
	}
	return nil
}

type scrollViewViewport struct {
	State *scrollViewState
	Child Widget
}

func (w scrollViewViewport) ChildWidget() Widget {
	return w.Child
}

func (w scrollViewViewport) CreateRenderObject(BuildContext) RenderObject {
	return &renderScrollView{State: w.State}
}

func (w scrollViewViewport) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderScrollView)
	if r.State != w.State {
		r.State = w.State
		r.MarkNeedsPaint()
	}
}

type renderScrollView struct {
	SingleChildRenderObject
	State     *scrollViewState
	childSize Size
}

func (r *renderScrollView) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.childSize = Size{}
		r.SetSize(c.Constrain(Size{}))
		return
	}
	childConstraints := Constraints{
		MinWidth:  c.MinWidth,
		MaxWidth:  c.MaxWidth,
		MaxHeight: Unbounded,
	}
	child.Layout(ctx, childConstraints)
	r.childSize = child.Base().Size()
	size := c.Constrain(r.childSize)
	if c.HasBoundedHeight() && size.Height > c.MaxHeight {
		size.Height = c.MaxHeight
	}
	r.SetSize(size)
	r.clampScroll()
}

func (r *renderScrollView) DryLayout(ctx LayoutContext, c Constraints) Size {
	childSize := DryLayout(ctx, r.Child(), Constraints{
		MinWidth:  c.MinWidth,
		MaxWidth:  c.MaxWidth,
		MaxHeight: Unbounded,
	})
	return c.Constrain(childSize)
}

func (r *renderScrollView) Paint(p *Painter, off Offset) {
	child := r.Child()
	if child == nil {
		return
	}
	p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
	defer p.PopClip()
	child.Paint(p, Offset{X: off.X, Y: off.Y - r.scrollRow()})
}

func (r *renderScrollView) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderScrollView) ChildOffset(RenderObject) Offset {
	return Offset{Y: -r.scrollRow()}
}

func (r *renderScrollView) SelectionClip() Rect {
	return Rect{Width: r.Size().Width, Height: r.Size().Height}
}

func (r *renderScrollView) SelectionChildOffset(RenderObject) Offset {
	return Offset{}
}

func (r *renderScrollView) SelectionSize() Size {
	child := r.Child()
	if child == nil {
		return Size{}
	}
	return selectionSize(child)
}

func (r *renderScrollView) ScrollByLines(lines int) bool {
	return r.ScrollToOffset(r.scrollRow() + lines)
}

func (r *renderScrollView) ScrollToOffset(row int) bool {
	if r.State == nil {
		return false
	}
	next := clampInt(row, 0, r.maxScroll())
	if next == r.scrollRow() {
		return false
	}
	r.State.SetState(func() { r.State.scrollRow = next })
	return true
}

func (r *renderScrollView) maxScroll() int {
	return max(0, r.childSize.Height-r.Size().Height)
}

func (r *renderScrollView) ScrollMetrics() ScrollMetrics {
	return ScrollMetrics{
		ScrollOffset:    r.scrollRow(),
		MaxScrollOffset: r.maxScroll(),
		ViewportHeight:  r.Size().Height,
		ViewportWidth:   r.Size().Width,
		ContentHeight:   r.childSize.Height,
	}
}

func (r *renderScrollView) scrollRow() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollRow
}

func (r *renderScrollView) clampScroll() {
	if r.State == nil {
		return
	}
	r.State.scrollRow = clampInt(r.State.scrollRow, 0, r.maxScroll())
}
