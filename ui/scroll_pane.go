package ui

// ScrollPane clips a single child to a viewport and scrolls it vertically and
// horizontally.
//
// Mouse wheel up and down scroll rows. Mouse wheel left and right scroll
// columns. Arrow keys and h/j/k/l scroll by one row or column. Page Up, Page
// Down, Space, Shift+Space, Home, and End operate on the vertical axis.
type ScrollPane struct {
	// Controller can be used to inspect and change the pane's row and column
	// offsets after it is mounted.
	Controller *ScrollPaneController
	// Child is laid out with unbounded width and height.
	Child Widget
}

func (w ScrollPane) CreateState() State {
	return &scrollPaneState{}
}

type scrollPaneState struct {
	StateBase
	node      FocusNode
	scrollRow int
	scrollCol int
}

func (s *scrollPaneState) Build(BuildContext) Widget {
	w := s.Widget().(ScrollPane)
	s.attachController(w.Controller)
	s.node.onChange = s.markNeedsFocusPaint
	child := Focus(&s.node, scrollPaneViewport{
		State: s,
		Child: w.Child,
	})
	return scrollDefaultActions(s, child)
}

func (s *scrollPaneState) DidUpdateWidget(old Widget) {
	next := s.Widget().(ScrollPane).Controller
	prev := old.(ScrollPane).Controller
	if next == prev {
		return
	}
	if prev != nil {
		prev.detach(s)
	}
	s.attachController(next)
}

func (s *scrollPaneState) Dispose() {
	if c := s.Widget().(ScrollPane).Controller; c != nil {
		c.detach(s)
	}
}

func (s *scrollPaneState) attachController(c *ScrollPaneController) {
	if c != nil {
		c.attach(s)
	}
}

func (s *scrollPaneState) markNeedsFocusPaint() {
	if r := s.renderObject(); r != nil {
		r.MarkNeedsPaint()
	}
}

func (s *scrollPaneState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		return handleScrollPaneKeyWithInvoke(ctx, ev)
	case Mouse:
		if ev.EventType != EventPress {
			return EventIgnored
		}
		if r := s.renderObject(); r != nil {
			switch ev.Button {
			case MouseWheelUp:
				r.ScrollByLinesAxis(ScrollVertical, -1)
				return EventHandled
			case MouseWheelDown:
				r.ScrollByLinesAxis(ScrollVertical, 1)
				return EventHandled
			case MouseWheelLeft:
				r.ScrollByLinesAxis(ScrollHorizontal, -1)
				return EventHandled
			case MouseWheelRight:
				r.ScrollByLinesAxis(ScrollHorizontal, 1)
				return EventHandled
			}
		}
	}
	return EventIgnored
}

func (s *scrollPaneState) renderObject() *renderScrollPane {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderScrollPane); ok {
		return r
	}
	return nil
}

func (s *scrollPaneState) ScrollByLinesAxis(axis ScrollAxis, lines int) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollByLinesAxis(axis, lines)
	}
	return false
}

func (s *scrollPaneState) ScrollByPagesAxis(axis ScrollAxis, pages int) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollByPagesAxis(axis, pages)
	}
	return false
}

func (s *scrollPaneState) ScrollToOffsetAxis(axis ScrollAxis, offset int) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollToOffsetAxis(axis, offset)
	}
	return false
}

func (s *scrollPaneState) ScrollToStartAxis(axis ScrollAxis) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollToStartAxis(axis)
	}
	return false
}

func (s *scrollPaneState) ScrollToEndAxis(axis ScrollAxis) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollToEndAxis(axis)
	}
	return false
}

func (s *scrollPaneState) ScrollMetricsForAxis(axis ScrollAxis) ScrollMetrics {
	if r := s.renderObject(); r != nil {
		return r.ScrollMetricsForAxis(axis)
	}
	return ScrollMetrics{}
}

type scrollPaneViewport struct {
	State *scrollPaneState
	Child Widget
}

func (w scrollPaneViewport) WidgetChild() Widget {
	return w.Child
}

func (w scrollPaneViewport) CreateRenderObject(BuildContext) RenderObject {
	return &renderScrollPane{State: w.State}
}

func (w scrollPaneViewport) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderScrollPane)
	if r.State != w.State {
		r.State = w.State
		r.MarkNeedsPaint()
	}
}

type renderScrollPane struct {
	SingleChildRenderObject
	State     *scrollPaneState
	childSize Size
}

func (r *renderScrollPane) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.childSize = Size{}
		r.SetSize(c.Constrain(Size{}))
		return
	}
	child.Layout(ctx, Constraints{MaxWidth: Unbounded, MaxHeight: Unbounded})
	r.childSize = child.Base().Size()
	r.SetSize(c.Constrain(r.childSize))
	r.clampScroll()
}

func (r *renderScrollPane) DryLayout(ctx LayoutContext, c Constraints) Size {
	childSize := DryLayout(ctx, r.Child(), Constraints{MaxWidth: Unbounded, MaxHeight: Unbounded})
	return c.Constrain(childSize)
}

func (r *renderScrollPane) Paint(p *Painter, off Offset) {
	child := r.Child()
	if child == nil {
		return
	}
	p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
	defer p.PopClip()
	child.Paint(p, off.Add(r.scrollOffset().Negate()))
}

func (r *renderScrollPane) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderScrollPane) ChildOffset(RenderObject) Offset {
	return r.scrollOffset().Negate()
}

func (r *renderScrollPane) SelectionClip() Rect {
	return Rect{Width: r.Size().Width, Height: r.Size().Height}
}

func (r *renderScrollPane) SelectionChildOffset(RenderObject) Offset {
	return Offset{}
}

func (r *renderScrollPane) SelectionSize() Size {
	child := r.Child()
	if child == nil {
		return Size{}
	}
	return selectionSize(child)
}

func (r *renderScrollPane) ScrollMetrics() ScrollMetrics {
	return r.ScrollMetricsForAxis(ScrollVertical)
}

func (r *renderScrollPane) ScrollByLines(lines int) bool {
	return r.ScrollByLinesAxis(ScrollVertical, lines)
}

func (r *renderScrollPane) ScrollByPages(pages int) bool {
	return r.ScrollByPagesAxis(ScrollVertical, pages)
}

func (r *renderScrollPane) ScrollToOffset(offset int) bool {
	return r.ScrollToOffsetAxis(ScrollVertical, offset)
}

func (r *renderScrollPane) ScrollToStart() bool {
	return r.ScrollToStartAxis(ScrollVertical)
}

func (r *renderScrollPane) ScrollToEnd() bool {
	return r.ScrollToEndAxis(ScrollVertical)
}

func (r *renderScrollPane) ScrollMetricsForAxis(axis ScrollAxis) ScrollMetrics {
	if axis == ScrollHorizontal {
		return ScrollMetrics{
			ScrollOffset:    r.scrollCol(),
			MaxScrollOffset: r.maxScrollAxis(ScrollHorizontal),
			ViewportHeight:  r.Size().Height,
			ViewportWidth:   r.Size().Width,
			ContentHeight:   r.childSize.Height,
			ContentWidth:    r.childSize.Width,
		}
	}
	return ScrollMetrics{
		ScrollOffset:    r.scrollRow(),
		MaxScrollOffset: r.maxScrollAxis(ScrollVertical),
		ViewportHeight:  r.Size().Height,
		ViewportWidth:   r.Size().Width,
		ContentHeight:   r.childSize.Height,
		ContentWidth:    r.childSize.Width,
	}
}

func (r *renderScrollPane) ScrollByLinesAxis(axis ScrollAxis, lines int) bool {
	return r.ScrollToOffsetAxis(axis, r.scrollOffsetAxis(axis)+lines)
}

func (r *renderScrollPane) ScrollByPagesAxis(axis ScrollAxis, pages int) bool {
	return r.ScrollByLinesAxis(axis, pages*r.pageSizeAxis(axis))
}

func (r *renderScrollPane) ScrollToOffsetAxis(axis ScrollAxis, offset int) bool {
	if r.State == nil {
		return false
	}
	next := clampInt(offset, 0, r.maxScrollAxis(axis))
	if next == r.scrollOffsetAxis(axis) {
		return false
	}
	r.State.SetState(func() {
		if axis == ScrollHorizontal {
			r.State.scrollCol = next
		} else {
			r.State.scrollRow = next
		}
	})
	return true
}

func (r *renderScrollPane) ScrollToStartAxis(axis ScrollAxis) bool {
	return r.ScrollToOffsetAxis(axis, 0)
}

func (r *renderScrollPane) ScrollToEndAxis(axis ScrollAxis) bool {
	return r.ScrollToOffsetAxis(axis, r.maxScrollAxis(axis))
}

func (r *renderScrollPane) HandleKey(key Key) EventResult {
	return handleScrollPaneKey(key, r)
}

func (r *renderScrollPane) scrollOffset() Offset {
	return Offset{X: r.scrollCol(), Y: r.scrollRow()}
}

func (r *renderScrollPane) scrollOffsetAxis(axis ScrollAxis) int {
	if axis == ScrollHorizontal {
		return r.scrollCol()
	}
	return r.scrollRow()
}

func (r *renderScrollPane) scrollRow() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollRow
}

func (r *renderScrollPane) scrollCol() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollCol
}

func (r *renderScrollPane) pageSizeAxis(axis ScrollAxis) int {
	if axis == ScrollHorizontal {
		return max(1, r.Size().Width)
	}
	return max(1, r.Size().Height)
}

func (r *renderScrollPane) maxScrollAxis(axis ScrollAxis) int {
	if axis == ScrollHorizontal {
		return max(0, r.childSize.Width-r.Size().Width)
	}
	return max(0, r.childSize.Height-r.Size().Height)
}

func (r *renderScrollPane) clampScroll() {
	if r.State == nil {
		return
	}
	r.State.scrollRow = clampInt(r.State.scrollRow, 0, r.maxScrollAxis(ScrollVertical))
	r.State.scrollCol = clampInt(r.State.scrollCol, 0, r.maxScrollAxis(ScrollHorizontal))
}
