package ui

// ScrollAxis identifies the direction a ScrollView scrolls.
type ScrollAxis int

const (
	// ScrollVertical scrolls a child vertically. This is the ScrollView default.
	ScrollVertical ScrollAxis = iota
	// ScrollHorizontal scrolls a child horizontally.
	ScrollHorizontal
)

// ScrollView clips a single child to a viewport and scrolls it on one axis.
// Mouse wheel events scroll by one row or column. Arrow keys and h/j/k/l scroll
// by one row or column. Page Up, Page Down, Space, and Shift+Space scroll by
// one viewport. Home and End jump to the start and end.
//
// When used inside SelectionArea, selections that start outside the ScrollView
// include hidden rows, while selections that start inside it initially use the
// visible rows and expand as selection autoscroll moves the viewport.
type ScrollView struct {
	// Axis controls which direction is scrollable. The zero value is vertical.
	Axis ScrollAxis
	// Child is laid out at the viewport cross-axis size with unbounded space
	// along Axis.
	Child Widget
}

func (w ScrollView) CreateState() State {
	return &scrollViewState{}
}

type scrollViewState struct {
	StateBase
	node      FocusNode
	scrollRow int
	scrollCol int
}

func (s *scrollViewState) Build(BuildContext) Widget {
	s.node.onChange = s.markNeedsFocusPaint
	return Focus(&s.node, scrollViewViewport{
		State: s,
		Axis:  s.Widget().(ScrollView).Axis,
		Child: s.Widget().(ScrollView).Child,
	})
}

func (s *scrollViewState) markNeedsFocusPaint() {
	if r := s.renderObject(); r != nil {
		r.MarkNeedsPaint()
	}
}

func (s *scrollViewState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if r := s.renderObject(); r != nil {
			return r.HandleKey(ev)
		}
	case Mouse:
		if ev.EventType != EventPress {
			return EventIgnored
		}
		switch ev.Button {
		case MouseWheelUp:
			if r := s.renderObject(); r != nil && r.Axis == ScrollHorizontal {
				return EventIgnored
			}
			return s.scrollBy(-1)
		case MouseWheelDown:
			if r := s.renderObject(); r != nil && r.Axis == ScrollHorizontal {
				return EventIgnored
			}
			return s.scrollBy(1)
		case MouseWheelLeft:
			if r := s.renderObject(); r == nil || r.Axis != ScrollHorizontal {
				return EventIgnored
			}
			return s.scrollBy(-1)
		case MouseWheelRight:
			if r := s.renderObject(); r == nil || r.Axis != ScrollHorizontal {
				return EventIgnored
			}
			return s.scrollBy(1)
		}
	}
	return EventIgnored
}

func (s *scrollViewState) scrollBy(delta int) EventResult {
	if r := s.renderObject(); r != nil {
		r.ScrollByLines(delta)
		return EventHandled
	}
	return EventIgnored
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
	Axis  ScrollAxis
	Child Widget
}

func (w scrollViewViewport) WidgetChild() Widget {
	return w.Child
}

func (w scrollViewViewport) CreateRenderObject(BuildContext) RenderObject {
	return &renderScrollView{State: w.State, Axis: w.Axis}
}

func (w scrollViewViewport) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderScrollView)
	if r.State != w.State {
		r.State = w.State
		r.MarkNeedsPaint()
	}
	if r.Axis != w.Axis {
		r.Axis = w.Axis
		r.MarkNeedsLayout()
	}
}

type renderScrollView struct {
	SingleChildRenderObject
	State     *scrollViewState
	Axis      ScrollAxis
	childSize Size
}

func (r *renderScrollView) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.childSize = Size{}
		r.SetSize(c.Constrain(Size{}))
		return
	}
	childConstraints := r.childConstraints(c)
	child.Layout(ctx, childConstraints)
	r.childSize = child.Base().Size()
	size := c.Constrain(r.childSize)
	if r.Axis == ScrollVertical && c.HasBoundedHeight() && size.Height > c.MaxHeight {
		size.Height = c.MaxHeight
	}
	if r.Axis == ScrollHorizontal && c.HasBoundedWidth() && size.Width > c.MaxWidth {
		size.Width = c.MaxWidth
	}
	r.SetSize(size)
	r.clampScroll()
}

func (r *renderScrollView) DryLayout(ctx LayoutContext, c Constraints) Size {
	childSize := DryLayout(ctx, r.Child(), r.childConstraints(c))
	return c.Constrain(childSize)
}

func (r *renderScrollView) childConstraints(c Constraints) Constraints {
	if r.Axis == ScrollHorizontal {
		return Constraints{
			MinHeight: c.MinHeight,
			MaxWidth:  Unbounded,
			MaxHeight: c.MaxHeight,
		}
	}
	return Constraints{
		MinWidth:  c.MinWidth,
		MaxWidth:  c.MaxWidth,
		MaxHeight: Unbounded,
	}
}

func (r *renderScrollView) Paint(p *Painter, off Offset) {
	child := r.Child()
	if child == nil {
		return
	}
	p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
	defer p.PopClip()
	child.Paint(p, off.Add(r.scrollOffset().Negate()))
}

func (r *renderScrollView) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderScrollView) ChildOffset(RenderObject) Offset {
	return r.scrollOffset().Negate()
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
	return r.ScrollToOffset(r.scrollMainOffset() + lines)
}

func (r *renderScrollView) ScrollByPages(pages int) bool {
	return r.ScrollByLines(pages * r.pageSize())
}

func (r *renderScrollView) ScrollToOffset(offset int) bool {
	if r.State == nil {
		return false
	}
	next := clampInt(offset, 0, r.maxScroll())
	if next == r.scrollMainOffset() {
		return false
	}
	r.State.SetState(func() {
		if r.Axis == ScrollHorizontal {
			r.State.scrollCol = next
		} else {
			r.State.scrollRow = next
		}
	})
	return true
}

func (r *renderScrollView) ScrollToStart() bool {
	return r.ScrollToOffset(0)
}

func (r *renderScrollView) ScrollToEnd() bool {
	return r.ScrollToOffset(r.maxScroll())
}

func (r *renderScrollView) pageSize() int {
	if r.Axis == ScrollHorizontal {
		return max(1, r.Size().Width)
	}
	return max(1, r.Size().Height)
}

func (r *renderScrollView) maxScroll() int {
	if r.Axis == ScrollHorizontal {
		return max(0, r.childSize.Width-r.Size().Width)
	}
	return max(0, r.childSize.Height-r.Size().Height)
}

func (r *renderScrollView) ScrollMetrics() ScrollMetrics {
	if r.Axis == ScrollHorizontal {
		return ScrollMetrics{
			ScrollOffset:    r.scrollCol(),
			MaxScrollOffset: r.maxScroll(),
			ViewportHeight:  r.Size().Height,
			ViewportWidth:   r.Size().Width,
			ContentHeight:   r.Size().Height,
			ContentWidth:    r.childSize.Width,
		}
	}
	return ScrollMetrics{
		ScrollOffset:    r.scrollRow(),
		MaxScrollOffset: r.maxScroll(),
		ViewportHeight:  r.Size().Height,
		ViewportWidth:   r.Size().Width,
		ContentHeight:   r.childSize.Height,
		ContentWidth:    r.Size().Width,
	}
}

func (r *renderScrollView) HandleKey(key Key) EventResult {
	if r.Axis == ScrollHorizontal {
		return handleHorizontalScrollKey(key, r)
	}
	return handleScrollKey(key, r)
}

func (r *renderScrollView) scrollMainOffset() int {
	if r.Axis == ScrollHorizontal {
		return r.scrollCol()
	}
	return r.scrollRow()
}

func (r *renderScrollView) scrollOffset() Offset {
	if r.Axis == ScrollHorizontal {
		return Offset{X: r.scrollCol()}
	}
	return Offset{Y: r.scrollRow()}
}

func (r *renderScrollView) scrollRow() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollRow
}

func (r *renderScrollView) scrollCol() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollCol
}

func (r *renderScrollView) clampScroll() {
	if r.State == nil {
		return
	}
	if r.Axis == ScrollHorizontal {
		r.State.scrollCol = clampInt(r.State.scrollCol, 0, r.maxScroll())
	} else {
		r.State.scrollRow = clampInt(r.State.scrollRow, 0, r.maxScroll())
	}
}
