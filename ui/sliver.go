package ui

// SliverConstraints describes the portion of a sliver visible in a
// CustomScrollView.
type SliverConstraints struct {
	// ViewportWidth is the available width in cells.
	ViewportWidth int
	// ViewportHeight is the viewport height in cells.
	ViewportHeight int
	// RemainingPaintExtent is the number of viewport rows left after previous
	// slivers.
	RemainingPaintExtent int
	// ScrollOffset is the number of rows scrolled into this sliver.
	ScrollOffset int
}

// SliverGeometry reports a sliver's scrollable and visible extent.
type SliverGeometry struct {
	// ScrollExtent is the sliver's total logical height in rows.
	ScrollExtent int
	// PaintExtent is the number of rows this sliver can paint in the viewport.
	PaintExtent int
}

type renderSliver interface {
	RenderObject
	LayoutSliver(LayoutContext, SliverConstraints) SliverGeometry
	PaintSliver(*Painter, Offset)
}

// CustomScrollView composes row-based slivers in one vertical scroll viewport.
//
// Mouse wheel events scroll by one line. Page Up and Page Down scroll by one
// viewport. Home and End jump to the start and end. Scrollbar can wrap a
// CustomScrollView because it exposes the same scroll metrics and commands as
// ScrollView.
type CustomScrollView struct {
	// Slivers are laid out vertically in order.
	Slivers []Widget
}

func (w CustomScrollView) CreateState() State {
	return &customScrollViewState{}
}

type customScrollViewState struct {
	StateBase
	node      FocusNode
	scrollRow int
}

func (s *customScrollViewState) Build(BuildContext) Widget {
	return Focus(&s.node, customScrollViewport{
		State:   s,
		Slivers: s.Widget().(CustomScrollView).Slivers,
	})
}

func (s *customScrollViewState) HandleEvent(ctx EventContext, ev Event) EventResult {
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
			return s.scrollByPages(-1)
		case KeyPgDown:
			return s.scrollByPages(1)
		case KeyHome:
			return s.scrollToStart()
		case KeyEnd:
			return s.scrollToEnd()
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

func (s *customScrollViewState) scrollBy(delta int) EventResult {
	if r := s.renderObject(); r != nil {
		r.ScrollByLines(delta)
		return EventHandled
	}
	return EventIgnored
}

func (s *customScrollViewState) scrollByPages(pages int) EventResult {
	if r := s.renderObject(); r != nil {
		r.ScrollByPages(pages)
		return EventHandled
	}
	return EventIgnored
}

func (s *customScrollViewState) scrollToStart() EventResult {
	if r := s.renderObject(); r != nil {
		r.ScrollToStart()
		return EventHandled
	}
	return EventIgnored
}

func (s *customScrollViewState) scrollToEnd() EventResult {
	if r := s.renderObject(); r != nil {
		r.ScrollToEnd()
		return EventHandled
	}
	return EventIgnored
}

func (s *customScrollViewState) renderObject() *renderCustomScrollView {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderCustomScrollView); ok {
		return r
	}
	return nil
}

type customScrollViewport struct {
	State   *customScrollViewState
	Slivers []Widget
}

func (w customScrollViewport) Children() []Widget {
	return w.Slivers
}

func (w customScrollViewport) CreateRenderObject(BuildContext) RenderObject {
	return &renderCustomScrollView{State: w.State}
}

func (w customScrollViewport) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderCustomScrollView)
	if r.State != w.State {
		r.State = w.State
		r.MarkNeedsPaint()
	}
}

type renderCustomScrollView struct {
	MultiChildRenderObject
	State         *customScrollViewState
	contentHeight int
	childOffsets  []Offset
}

func (r *renderCustomScrollView) Layout(ctx LayoutContext, c Constraints) {
	width := 0
	if c.HasBoundedWidth() {
		width = c.MaxWidth
	}
	viewportHeight := c.MaxHeight
	if viewportHeight == Unbounded {
		viewportHeight = 0
	}
	slivers := r.Children()
	r.childOffsets = make([]Offset, len(slivers))
	scrollBefore := 0
	contentHeight := 0
	for i, child := range slivers {
		sliver, ok := child.(renderSliver)
		if !ok {
			continue
		}
		geometry := sliver.LayoutSliver(ctx, SliverConstraints{
			ViewportWidth:        width,
			ViewportHeight:       viewportHeight,
			RemainingPaintExtent: max(0, viewportHeight-contentHeight+r.scrollRow()),
			ScrollOffset:         max(0, r.scrollRow()-scrollBefore),
		})
		r.childOffsets[i] = Offset{Y: contentHeight}
		contentHeight += geometry.ScrollExtent
		width = max(width, sliver.Base().Size().Width)
		scrollBefore += geometry.ScrollExtent
	}
	r.contentHeight = contentHeight
	size := Size{Width: width, Height: contentHeight}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = min(contentHeight, c.MaxHeight)
	}
	r.SetSize(c.Constrain(size))
	r.clampScroll()
}

func (r *renderCustomScrollView) DryLayout(ctx LayoutContext, c Constraints) Size {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func (r *renderCustomScrollView) Paint(p *Painter, off Offset) {
	p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
	defer p.PopClip()
	for i, child := range r.Children() {
		sliver, ok := child.(renderSliver)
		if !ok || i >= len(r.childOffsets) {
			continue
		}
		childOff := r.childOffsets[i]
		sliver.PaintSliver(p, Offset{X: off.X + childOff.X, Y: off.Y + childOff.Y - r.scrollRow()})
	}
}

func (r *renderCustomScrollView) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderCustomScrollView) ChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return Offset{X: r.childOffsets[i].X, Y: r.childOffsets[i].Y - r.scrollRow()}
		}
	}
	return Offset{}
}

func (r *renderCustomScrollView) SelectionClip() Rect {
	return Rect{Width: r.Size().Width, Height: r.Size().Height}
}

func (r *renderCustomScrollView) SelectionChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return r.childOffsets[i]
		}
	}
	return Offset{}
}

func (r *renderCustomScrollView) SelectionSize() Size {
	return Size{Width: r.Size().Width, Height: r.contentHeight}
}

func (r *renderCustomScrollView) ScrollByLines(lines int) bool {
	return r.ScrollToOffset(r.scrollRow() + lines)
}

func (r *renderCustomScrollView) ScrollByPages(pages int) bool {
	return r.ScrollByLines(pages * r.pageSize())
}

func (r *renderCustomScrollView) ScrollToOffset(row int) bool {
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

func (r *renderCustomScrollView) ScrollToStart() bool {
	return r.ScrollToOffset(0)
}

func (r *renderCustomScrollView) ScrollToEnd() bool {
	return r.ScrollToOffset(r.maxScroll())
}

func (r *renderCustomScrollView) ScrollMetrics() ScrollMetrics {
	return ScrollMetrics{
		ScrollOffset:    r.scrollRow(),
		MaxScrollOffset: r.maxScroll(),
		ViewportHeight:  r.Size().Height,
		ViewportWidth:   r.Size().Width,
		ContentHeight:   r.contentHeight,
	}
}

func (r *renderCustomScrollView) pageSize() int {
	return max(1, r.Size().Height)
}

func (r *renderCustomScrollView) maxScroll() int {
	return max(0, r.contentHeight-r.Size().Height)
}

func (r *renderCustomScrollView) scrollRow() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollRow
}

func (r *renderCustomScrollView) clampScroll() {
	if r.State == nil {
		return
	}
	r.State.scrollRow = clampInt(r.State.scrollRow, 0, r.maxScroll())
}

// SliverToBox adapts a normal box widget into a CustomScrollView sliver.
type SliverToBox struct {
	// Child is laid out at the viewport width with unbounded height.
	Child Widget
}

func (w SliverToBox) ChildWidget() Widget {
	return w.Child
}

func (w SliverToBox) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverToBox{}
}

func (w SliverToBox) UpdateRenderObject(BuildContext, RenderObject) {
}

type renderSliverToBox struct {
	SingleChildRenderObject
	geometry SliverGeometry
}

func (r *renderSliverToBox) Layout(ctx LayoutContext, c Constraints) {
	geometry := r.LayoutSliver(ctx, SliverConstraints{
		ViewportWidth:        c.MaxWidth,
		ViewportHeight:       c.MaxHeight,
		RemainingPaintExtent: c.MaxHeight,
	})
	r.geometry = geometry
}

func (r *renderSliverToBox) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	child := r.Child()
	if child == nil {
		r.SetSize(Size{})
		return SliverGeometry{}
	}
	child.Layout(ctx, Constraints{MaxWidth: c.ViewportWidth, MaxHeight: Unbounded})
	size := child.Base().Size()
	r.SetSize(size)
	r.geometry = SliverGeometry{
		ScrollExtent: size.Height,
		PaintExtent:  visibleSliverExtent(c, size.Height),
	}
	return r.geometry
}

func (r *renderSliverToBox) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverToBox) PaintSliver(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderSliverToBox) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverToBox) SelectionSize() Size {
	if child := r.Child(); child != nil {
		return selectionSize(child)
	}
	return r.Size()
}

// SliverFillRemaining sizes its child to at least the remaining viewport height.
type SliverFillRemaining struct {
	// Child is laid out at the viewport width and fills any remaining rows.
	Child Widget
}

func (w SliverFillRemaining) ChildWidget() Widget {
	return w.Child
}

func (w SliverFillRemaining) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverFillRemaining{}
}

func (w SliverFillRemaining) UpdateRenderObject(BuildContext, RenderObject) {
}

type renderSliverFillRemaining struct {
	SingleChildRenderObject
	geometry SliverGeometry
}

func (r *renderSliverFillRemaining) Layout(ctx LayoutContext, c Constraints) {
	r.LayoutSliver(ctx, SliverConstraints{
		ViewportWidth:        c.MaxWidth,
		ViewportHeight:       c.MaxHeight,
		RemainingPaintExtent: c.MaxHeight,
	})
}

func (r *renderSliverFillRemaining) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	child := r.Child()
	if child == nil {
		r.SetSize(Size{Width: c.ViewportWidth, Height: max(0, c.RemainingPaintExtent)})
		r.geometry = SliverGeometry{ScrollExtent: r.Size().Height, PaintExtent: visibleSliverExtent(c, r.Size().Height)}
		return r.geometry
	}
	child.Layout(ctx, Constraints{
		MinWidth:  c.ViewportWidth,
		MaxWidth:  c.ViewportWidth,
		MinHeight: max(0, c.RemainingPaintExtent),
		MaxHeight: Unbounded,
	})
	size := child.Base().Size()
	r.SetSize(size)
	r.geometry = SliverGeometry{
		ScrollExtent: size.Height,
		PaintExtent:  visibleSliverExtent(c, size.Height),
	}
	return r.geometry
}

func (r *renderSliverFillRemaining) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverFillRemaining) PaintSliver(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderSliverFillRemaining) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverFillRemaining) SelectionSize() Size {
	if child := r.Child(); child != nil {
		return selectionSize(child)
	}
	return r.Size()
}

// SliverList lays out a fixed list of children as one scrollable sliver.
type SliverList struct {
	// Children are laid out vertically in order.
	ChildrenWidget []Widget
}

func (w SliverList) Children() []Widget {
	return w.ChildrenWidget
}

func (w SliverList) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverList{}
}

func (w SliverList) UpdateRenderObject(BuildContext, RenderObject) {
}

type renderSliverList struct {
	MultiChildRenderObject
	geometry     SliverGeometry
	childOffsets []Offset
}

func (r *renderSliverList) Layout(ctx LayoutContext, c Constraints) {
	r.LayoutSliver(ctx, SliverConstraints{
		ViewportWidth:        c.MaxWidth,
		ViewportHeight:       c.MaxHeight,
		RemainingPaintExtent: c.MaxHeight,
	})
}

func (r *renderSliverList) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	width := 0
	y := 0
	children := r.Children()
	r.childOffsets = make([]Offset, len(children))
	for i, child := range children {
		child.Layout(ctx, Constraints{MaxWidth: c.ViewportWidth, MaxHeight: Unbounded})
		r.childOffsets[i] = Offset{Y: y}
		size := child.Base().Size()
		y += size.Height
		width = max(width, size.Width)
	}
	r.SetSize(Size{Width: width, Height: y})
	r.geometry = SliverGeometry{
		ScrollExtent: y,
		PaintExtent:  visibleSliverExtent(c, y),
	}
	return r.geometry
}

func (r *renderSliverList) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverList) PaintSliver(p *Painter, off Offset) {
	for i, child := range r.Children() {
		if i >= len(r.childOffsets) {
			continue
		}
		child.Paint(p, off.Add(r.childOffsets[i]))
	}
}

func (r *renderSliverList) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverList) ChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return r.childOffsets[i]
		}
	}
	return Offset{}
}

func (r *renderSliverList) SelectionChildOffset(child RenderObject) Offset {
	return r.ChildOffset(child)
}

func (r *renderSliverList) SelectionSize() Size {
	return r.Size()
}

func visibleSliverExtent(c SliverConstraints, scrollExtent int) int {
	if c.ViewportHeight <= 0 || c.ScrollOffset >= scrollExtent {
		return 0
	}
	return min(c.ViewportHeight, scrollExtent-c.ScrollOffset)
}
