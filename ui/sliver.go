package ui

import "strconv"

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
	// ObscuredLeadingExtent is the number of rows from this sliver's leading
	// edge that are hidden by viewport clipping or pinned content.
	ObscuredLeadingExtent int
}

// SliverGeometry reports a sliver's scrollable and visible extent.
type SliverGeometry struct {
	// ScrollExtent is the sliver's total logical height in rows.
	ScrollExtent int
	// PaintExtent is the number of rows this sliver can paint in the viewport.
	PaintExtent int
	// ScrollOffsetCorrection adjusts the viewport offset after newly measured
	// content changes the logical position of the current anchor row.
	ScrollOffsetCorrection int
}

type renderSliver interface {
	RenderObject
	LayoutSliver(LayoutContext, SliverConstraints) SliverGeometry
	PaintSliver(*Painter, Offset)
}

type pinnedSliver interface {
	PinnedOffset(logical Offset, scrollOffset int) Offset
}

const defaultSliverListBuilderInitialCount = 32

// CustomScrollView composes row-based slivers in one vertical scroll viewport.
//
// Mouse wheel events scroll by one line. Page Up and Page Down scroll by one
// viewport. Home and End jump to the start and end. Scrollbar can wrap a
// CustomScrollView because it exposes the same scroll metrics and commands as
// ScrollView. Slivers may report a scroll offset correction during layout when
// lazy measurement changes the logical position of visible content; the
// viewport applies the correction and lays out again so the current anchor row
// stays visually stable.
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
	for i := 0; i < 4; i++ {
		if r.layoutOnce(ctx, c) == 0 {
			return
		}
	}
	r.layoutOnce(ctx, c)
}

func (r *renderCustomScrollView) layoutOnce(ctx LayoutContext, c Constraints) int {
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
	pinnedLeadingExtent := 0
	for i, child := range slivers {
		sliver, ok := child.(renderSliver)
		if !ok {
			continue
		}
		childPaintOffset := contentHeight - r.scrollRow()
		geometry := sliver.LayoutSliver(ctx, SliverConstraints{
			ViewportWidth:        width,
			ViewportHeight:       viewportHeight,
			RemainingPaintExtent: max(0, viewportHeight-contentHeight+r.scrollRow()),
			ScrollOffset:         max(0, r.scrollRow()-scrollBefore),
			ObscuredLeadingExtent: max(0, pinnedLeadingExtent-
				childPaintOffset),
		})
		r.childOffsets[i] = Offset{Y: contentHeight}
		contentHeight += geometry.ScrollExtent
		width = max(width, sliver.Base().Size().Width)
		scrollBefore += geometry.ScrollExtent
		if geometry.ScrollOffsetCorrection != 0 && r.State != nil {
			r.State.scrollRow = max(0, r.State.scrollRow+geometry.ScrollOffsetCorrection)
			return geometry.ScrollOffsetCorrection
		}
		if _, ok := sliver.(pinnedSliver); ok && childPaintOffset <= pinnedLeadingExtent {
			pinnedLeadingExtent = max(pinnedLeadingExtent, sliver.Base().Size().Height)
		}
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
	return 0
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
		if _, ok := sliver.(pinnedSliver); ok {
			continue
		}
		childOff := r.paintChildOffset(i)
		sliver.PaintSliver(p, off.Add(childOff))
	}
	for i, child := range r.Children() {
		sliver, ok := child.(renderSliver)
		if !ok || i >= len(r.childOffsets) {
			continue
		}
		if _, ok := sliver.(pinnedSliver); !ok {
			continue
		}
		childOff := r.paintChildOffset(i)
		sliver.PaintSliver(p, off.Add(childOff))
	}
}

func (r *renderCustomScrollView) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderCustomScrollView) ChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return r.paintChildOffset(i)
		}
	}
	return Offset{}
}

func (r *renderCustomScrollView) paintChildOffset(i int) Offset {
	if i >= len(r.childOffsets) {
		return Offset{}
	}
	off := Offset{X: r.childOffsets[i].X, Y: r.childOffsets[i].Y - r.scrollRow()}
	if pinned, ok := r.Children()[i].(pinnedSliver); ok {
		return pinned.PinnedOffset(off, r.scrollRow())
	}
	return off
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
//
// The child is laid out at the viewport width with unbounded height. Use this
// for headers, footers, and other one-off content mixed into a sliver viewport.
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

// SliverPinnedHeader keeps its child visible at the top of a CustomScrollView
// after it would otherwise scroll offscreen.
//
// The header still contributes its normal height to scroll extent. While
// pinned, it paints after non-pinned slivers so it covers rows that scroll
// underneath it.
type SliverPinnedHeader struct {
	// Child is laid out at the viewport width with its natural height.
	Child Widget
}

func (w SliverPinnedHeader) ChildWidget() Widget {
	return w.Child
}

func (w SliverPinnedHeader) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverPinnedHeader{}
}

func (w SliverPinnedHeader) UpdateRenderObject(BuildContext, RenderObject) {
}

type renderSliverPinnedHeader struct {
	SingleChildRenderObject
	geometry SliverGeometry
}

func (r *renderSliverPinnedHeader) Layout(ctx LayoutContext, c Constraints) {
	r.LayoutSliver(ctx, SliverConstraints{
		ViewportWidth:        c.MaxWidth,
		ViewportHeight:       c.MaxHeight,
		RemainingPaintExtent: c.MaxHeight,
	})
}

func (r *renderSliverPinnedHeader) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	child := r.Child()
	if child == nil {
		r.SetSize(Size{})
		r.geometry = SliverGeometry{}
		return r.geometry
	}
	child.Layout(ctx, Constraints{MinWidth: c.ViewportWidth, MaxWidth: c.ViewportWidth, MaxHeight: Unbounded})
	size := child.Base().Size()
	r.SetSize(size)
	r.geometry = SliverGeometry{
		ScrollExtent: size.Height,
		PaintExtent:  min(size.Height, c.ViewportHeight),
	}
	return r.geometry
}

func (r *renderSliverPinnedHeader) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverPinnedHeader) PaintSliver(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderSliverPinnedHeader) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverPinnedHeader) PinnedOffset(off Offset, _ int) Offset {
	if off.Y < 0 {
		off.Y = 0
	}
	return off
}

func (r *renderSliverPinnedHeader) SelectionSize() Size {
	if child := r.Child(); child != nil {
		return selectionSize(child)
	}
	return r.Size()
}

// SliverFillRemaining sizes its child to at least the remaining viewport height.
//
// If previous slivers do not fill the viewport, the child is expanded to cover
// the remaining rows. If the child needs more height than remains, it scrolls as
// normal content.
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

// SliverList lays out an eager list of children as one scrollable sliver.
//
// All children are built and laid out every pass. Use SliverList for small or
// already-materialized lists; use SliverListBuilder for large or dynamic lists.
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

// SliverListBuilder lazily builds rows for a CustomScrollView.
//
// When ItemExtent is greater than zero, every row uses that fixed height and
// scroll offsets are exact. When ItemExtent is zero, rows are measured as they
// are laid out and EstimatedItemExtent is used for rows that have not been
// built yet. Overscan adds rows before and after the visible range so small
// scroll deltas can paint without waiting for another build.
//
// In measured mode, row heights are cached per viewport width. Resizing clears
// the measurements for the old width, anchors the currently visible row, and
// corrects the viewport scroll offset after rows are measured at the new width.
type SliverListBuilder struct {
	// Count is the number of logical rows available from Builder.
	Count int
	// ItemExtent is the fixed height of each item in cells when greater than
	// zero.
	ItemExtent int
	// EstimatedItemExtent is the height used for unmeasured rows when
	// ItemExtent is zero. A zero or negative value is treated as one row.
	EstimatedItemExtent int
	// Overscan builds this many extra items before and after the viewport.
	Overscan int
	// Builder returns the widget for index. It is only called for the active
	// visible range plus Overscan.
	Builder func(BuildContext, int) Widget
}

func (w SliverListBuilder) CreateState() State {
	return &sliverListBuilderState{last: defaultSliverListBuilderInitialCount}
}

type sliverListBuilderState struct {
	StateBase
	first   int
	last    int
	width   int
	extents map[int]int
}

func (s *sliverListBuilderState) Build(ctx BuildContext) Widget {
	w := s.Widget().(SliverListBuilder)
	count := max(0, w.Count)
	first := clampInt(s.first, 0, count)
	last := clampInt(s.last, first, count)
	children := make([]Widget, 0, max(0, last-first))
	if w.Builder != nil {
		for i := first; i < last; i++ {
			child := w.Builder(ctx, i)
			if child == nil {
				child = SizedBox{Height: normalizeSliverBuilderEstimate(w)}
			}
			children = append(children, sliverListBuilderChild{
				Key:         KeyValue(strconv.Itoa(i)),
				ChildWidget: child,
			})
		}
	}
	return sliverListBuilderView{
		State:       s,
		Count:       count,
		ItemExtent:  w.ItemExtent,
		Estimate:    normalizeSliverBuilderEstimate(w),
		Overscan:    max(0, w.Overscan),
		First:       first,
		Extents:     s.extentsForWidth(),
		ChildWidget: children,
	}
}

func (s *sliverListBuilderState) extentsForWidth() map[int]int {
	if s.extents == nil {
		return nil
	}
	return s.extents
}

func (s *sliverListBuilderState) updateLayout(width, first, last int, measured map[int]int) {
	reset := s.width != width
	changed := reset || first != s.first || last != s.last
	if !changed {
		for index, extent := range measured {
			if s.extents == nil || s.extents[index] != extent {
				changed = true
				break
			}
		}
	}
	if !changed {
		return
	}
	s.SetState(func() {
		if reset || s.extents == nil {
			s.width = width
			s.extents = make(map[int]int)
		}
		for index, extent := range measured {
			s.extents[index] = extent
		}
		s.first = first
		s.last = last
	})
}

type sliverListBuilderChild struct {
	Key         KeyValue
	ChildWidget Widget
}

func (w sliverListBuilderChild) WidgetKey() KeyValue {
	return w.Key
}

func (w sliverListBuilderChild) Build(BuildContext) Widget {
	return w.ChildWidget
}

type sliverListBuilderView struct {
	State       *sliverListBuilderState
	Count       int
	ItemExtent  int
	Estimate    int
	Overscan    int
	First       int
	Extents     map[int]int
	ChildWidget []Widget
}

func (w sliverListBuilderView) Children() []Widget {
	return w.ChildWidget
}

func (w sliverListBuilderView) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverListBuilder{
		State:      w.State,
		Count:      w.Count,
		ItemExtent: w.ItemExtent,
		Estimate:   w.Estimate,
		Overscan:   w.Overscan,
		First:      w.First,
		Extents:    w.Extents,
	}
}

func (w sliverListBuilderView) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderSliverListBuilder)
	r.State = w.State
	r.Count = w.Count
	r.ItemExtent = w.ItemExtent
	r.Estimate = w.Estimate
	r.Overscan = w.Overscan
	r.First = w.First
	r.Extents = w.Extents
	r.MarkNeedsLayout()
}

type renderSliverListBuilder struct {
	MultiChildRenderObject
	State        *sliverListBuilderState
	Count        int
	ItemExtent   int
	Estimate     int
	Overscan     int
	First        int
	Extents      map[int]int
	geometry     SliverGeometry
	childOffsets []Offset
}

func (r *renderSliverListBuilder) Layout(ctx LayoutContext, c Constraints) {
	r.LayoutSliver(ctx, SliverConstraints{
		ViewportWidth:        c.MaxWidth,
		ViewportHeight:       c.MaxHeight,
		RemainingPaintExtent: c.MaxHeight,
	})
}

func (r *renderSliverListBuilder) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	if r.ItemExtent > 0 {
		return r.layoutFixed(ctx, c)
	}
	return r.layoutVariable(ctx, c)
}

func (r *renderSliverListBuilder) layoutFixed(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	itemExtent := normalizeSliverItemExtent(r.ItemExtent)
	scrollExtent := max(0, r.Count) * itemExtent
	first, last := sliverListBuilderFixedVisibleRange(r.Count, itemExtent, r.Overscan, c)
	width := c.ViewportWidth
	children := r.Children()
	r.childOffsets = make([]Offset, len(children))
	for i, child := range children {
		index := r.First + i
		child.Layout(ctx, Constraints{
			MinWidth:  c.ViewportWidth,
			MaxWidth:  c.ViewportWidth,
			MinHeight: itemExtent,
			MaxHeight: itemExtent,
		})
		r.childOffsets[i] = Offset{Y: index * itemExtent}
		width = max(width, child.Base().Size().Width)
	}
	if r.State != nil {
		r.State.updateLayout(c.ViewportWidth, first, last, nil)
	}
	r.SetSize(Size{Width: width, Height: scrollExtent})
	r.geometry = SliverGeometry{
		ScrollExtent: scrollExtent,
		PaintExtent:  visibleSliverExtent(c, scrollExtent),
	}
	return r.geometry
}

func (r *renderSliverListBuilder) layoutVariable(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	estimate := normalizeSliverEstimatedItemExtent(r.Estimate)
	cachedExtents := cloneSliverExtentCache(r.Extents)
	resized := r.State != nil && r.State.width != 0 && r.State.width != c.ViewportWidth
	anchorExtents := cachedExtents
	if resized {
		anchorExtents = cloneSliverExtentCache(r.Extents)
		cachedExtents = nil
	}
	extents := cloneSliverExtentCache(cachedExtents)
	first, last := sliverListBuilderVariableVisibleRange(r.Count, estimate, r.Overscan, extents, c)
	anchorScrollOffset := max(c.ScrollOffset, c.ObscuredLeadingExtent)
	anchorIndex := sliverListBuilderIndexForOffset(anchorScrollOffset, r.Count, estimate, anchorExtents)
	anchorOffset := sliverListBuilderOffsetForIndex(anchorIndex, estimate, anchorExtents)
	anchorDelta := anchorScrollOffset - anchorOffset
	if resized {
		paintExtent := max(0, min(c.ViewportHeight, c.RemainingPaintExtent))
		first = clampInt(anchorIndex-r.Overscan, 0, max(0, r.Count))
		last = clampInt(anchorIndex+(paintExtent+estimate-1)/estimate+r.Overscan+1, first, max(0, r.Count))
	}
	width := c.ViewportWidth
	children := r.Children()
	measured := make(map[int]int, len(children))
	r.childOffsets = make([]Offset, len(children))
	for i, child := range children {
		index := r.First + i
		r.childOffsets[i] = Offset{Y: sliverListBuilderOffsetForIndex(index, estimate, extents)}
		child.Layout(ctx, Constraints{
			MinWidth:  c.ViewportWidth,
			MaxWidth:  c.ViewportWidth,
			MaxHeight: Unbounded,
		})
		size := child.Base().Size()
		extent := max(0, size.Height)
		measured[index] = extent
		extents[index] = extent
		width = max(width, size.Width)
	}
	scrollExtent := sliverListBuilderScrollExtent(r.Count, estimate, extents)
	correction := 0
	if anchorScrollOffset > 0 && anchorIndex < max(0, r.Count) {
		correction = sliverListBuilderOffsetForIndex(anchorIndex, estimate, extents) + anchorDelta - anchorScrollOffset
	}
	r.Extents = extents
	if r.State != nil {
		r.State.updateLayout(c.ViewportWidth, first, last, measured)
	}
	r.SetSize(Size{Width: width, Height: scrollExtent})
	r.geometry = SliverGeometry{
		ScrollExtent:           scrollExtent,
		PaintExtent:            visibleSliverExtent(c, scrollExtent),
		ScrollOffsetCorrection: correction,
	}
	return r.geometry
}

func (r *renderSliverListBuilder) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverListBuilder) PaintSliver(p *Painter, off Offset) {
	for i, child := range r.Children() {
		if i >= len(r.childOffsets) {
			continue
		}
		child.Paint(p, off.Add(r.childOffsets[i]))
	}
}

func (r *renderSliverListBuilder) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverListBuilder) ChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return r.childOffsets[i]
		}
	}
	return Offset{}
}

func (r *renderSliverListBuilder) SelectionChildOffset(child RenderObject) Offset {
	return r.ChildOffset(child)
}

func (r *renderSliverListBuilder) SelectionSize() Size {
	return r.Size()
}

func normalizeSliverItemExtent(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

func normalizeSliverEstimatedItemExtent(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

func normalizeSliverBuilderEstimate(w SliverListBuilder) int {
	if w.ItemExtent > 0 {
		return normalizeSliverItemExtent(w.ItemExtent)
	}
	return normalizeSliverEstimatedItemExtent(w.EstimatedItemExtent)
}

func sliverListBuilderFixedVisibleRange(count, itemExtent, overscan int, c SliverConstraints) (int, int) {
	count = max(0, count)
	if count == 0 {
		return 0, 0
	}
	itemExtent = normalizeSliverItemExtent(itemExtent)
	overscan = max(0, overscan)
	paintExtent := max(0, min(c.ViewportHeight, c.RemainingPaintExtent))
	first := clampInt(c.ScrollOffset/itemExtent-overscan, 0, count)
	last := clampInt((c.ScrollOffset+paintExtent+itemExtent-1)/itemExtent+overscan, first, count)
	return first, last
}

func sliverListBuilderVariableVisibleRange(count, estimate, overscan int, extents map[int]int, c SliverConstraints) (int, int) {
	count = max(0, count)
	if count == 0 {
		return 0, 0
	}
	estimate = normalizeSliverEstimatedItemExtent(estimate)
	overscan = max(0, overscan)
	paintExtent := max(0, min(c.ViewportHeight, c.RemainingPaintExtent))
	first := clampInt(sliverListBuilderIndexForOffset(c.ScrollOffset, count, estimate, extents)-overscan, 0, count)
	last := clampInt(sliverListBuilderIndexForOffset(c.ScrollOffset+paintExtent, count, estimate, extents)+overscan+1, first, count)
	return first, last
}

func sliverListBuilderIndexForOffset(offset, count, estimate int, extents map[int]int) int {
	offset = max(0, offset)
	y := 0
	for i := 0; i < count; i++ {
		y += sliverListBuilderExtentForIndex(i, estimate, extents)
		if y > offset {
			return i
		}
	}
	return count
}

func sliverListBuilderOffsetForIndex(index, estimate int, extents map[int]int) int {
	index = max(0, index)
	y := 0
	for i := 0; i < index; i++ {
		y += sliverListBuilderExtentForIndex(i, estimate, extents)
	}
	return y
}

func sliverListBuilderScrollExtent(count, estimate int, extents map[int]int) int {
	count = max(0, count)
	y := 0
	for i := 0; i < count; i++ {
		y += sliverListBuilderExtentForIndex(i, estimate, extents)
	}
	return y
}

func sliverListBuilderExtentForIndex(index, estimate int, extents map[int]int) int {
	if extent, ok := extents[index]; ok {
		return extent
	}
	return normalizeSliverEstimatedItemExtent(estimate)
}

func cloneSliverExtentCache(extents map[int]int) map[int]int {
	next := make(map[int]int, len(extents))
	for index, extent := range extents {
		next[index] = extent
	}
	return next
}

func visibleSliverExtent(c SliverConstraints, scrollExtent int) int {
	if c.ViewportHeight <= 0 || c.ScrollOffset >= scrollExtent {
		return 0
	}
	return min(c.ViewportHeight, scrollExtent-c.ScrollOffset)
}
