package ui

import (
	"math"
	"time"
)

// SelectionArea enables read-only text selection for descendant Text and
// RichText widgets.
//
// Users can drag to select text, double-click to select a word, triple-click
// to select a line, press Ctrl+A to select all selectable descendants, and
// press Ctrl+C to copy the current selection. TextField and TextArea manage
// their own editable selections and are skipped by SelectionArea traversal.
//
// Mouse selections copy the visible text when they start inside clipped
// content. Selections that start outside a ScrollView include its hidden rows,
// and selections that start inside a ScrollView expand to hidden rows only when
// autoscrolling moves the viewport.
type SelectionArea struct {
	// Child is the subtree that can contain selectable text.
	Child Widget
}

func (w SelectionArea) CreateState() State {
	return &selectionAreaState{}
}

type selectionAreaState struct {
	StateBase
	node         FocusNode
	anchor       selectionEndpoint
	extent       selectionEndpoint
	hasSelection bool
	visibleOnly  bool
	selecting    bool
	selected     []selectableTextRender
	autoScroll   selectionAutoScroll
	now          func() time.Time
	lastClick    time.Time
	lastClickRow int
	lastClickCol int
	clickCount   int
}

func (s *selectionAreaState) Build(BuildContext) Widget {
	child := FocusWithOptions(&s.node, FocusOptions{SkipTraversal: true}, selectionAreaView{
		State: s,
		Child: s.Widget().(SelectionArea).Child,
	})
	return DefaultActions{
		Bindings: map[IntentType]ActionFunc{
			SelectAllTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return s.selectAllText()
			},
			CopySelectionTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return s.copySelection(ctx, intent)
			},
		},
		Child: child,
	}
}

func (s *selectionAreaState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if keyIsRelease(ev) {
			return EventIgnored
		}
		if ev.MatchString("Ctrl+c") {
			return ctx.Invoke(CopySelectionTextIntent{})
		}
		if ev.MatchString("Ctrl+a") {
			return ctx.Invoke(SelectAllTextIntent{})
		}
	case Mouse:
		return s.handleMouse(ctx, ev)
	}
	return EventIgnored
}

func (s *selectionAreaState) copySelection(ctx EventContext, intent Intent) EventResult {
	copyIntent, _ := intent.(CopySelectionTextIntent)
	if s.hasSelection {
		if area := s.areaRender(); area != nil {
			text := area.SelectedText(s.anchor, s.extent, s.visibleOnly)
			ctx.Copy(text)
			if text != "" && copyIntent.OnCopied != nil {
				copyIntent.OnCopied(text)
			}
		}
	}
	return EventHandled
}

func (s *selectionAreaState) selectAllText() EventResult {
	area := s.areaRender()
	if area == nil {
		return EventIgnored
	}
	if anchor, extent, ok := area.selectAllEndpoints(); ok {
		s.setSelection(anchor, extent, false)
	}
	return EventHandled
}

func (s *selectionAreaState) handleMouse(ctx EventContext, mouse Mouse) EventResult {
	switch mouse.EventType {
	default:
		return EventIgnored
	case EventPress:
		if mouse.Button != MouseLeftButton {
			return EventIgnored
		}
		area := s.areaRender()
		if area == nil {
			return EventIgnored
		}
		selectable, off, local, clipped, scrollTarget, scrollOff, ok := area.selectableAt(Point{X: mouse.Col, Y: mouse.Row})
		if !ok {
			s.clearSelection()
			return EventIgnored
		}
		pos, ok := selectable.PositionForPoint(local)
		if !ok {
			return EventIgnored
		}
		s.node.RequestFocus()
		clickCount := s.mouseClickCount(mouse)
		switch {
		case clickCount >= 3:
			s.selecting = false
			s.setTextSelection(selectable, off, selectable.SelectLineAt(pos), true)
		case clickCount == 2:
			s.selecting = false
			s.setTextSelection(selectable, off, selectable.SelectWordAt(pos), true)
		default:
			s.selecting = true
			endpoint := selectionEndpoint{Selectable: selectable, Offset: off, Position: pos, Clipped: clipped}
			s.setSelection(endpoint, endpoint, true)
			if ctx.app != nil {
				ctx.app.captureMouse(s.element)
			}
			s.startAutoScroll(ctx, mouse, scrollTarget, scrollOff)
		}
		return EventHandled
	case EventMotion:
		if !s.selecting || !s.hasSelection {
			return EventIgnored
		}
		if mouse.Button == MouseNoButton {
			s.stopSelecting(ctx)
			return EventHandled
		}
		area := s.areaRender()
		if area == nil {
			return EventIgnored
		}
		selectable, off, local, clipped, scrollTarget, scrollOff, ok := area.selectableAt(Point{X: mouse.Col, Y: mouse.Row})
		if ok {
			pos, ok := selectable.PositionForPoint(local)
			if !ok {
				return EventIgnored
			}
			extent := selectionEndpoint{Selectable: selectable, Offset: off, Position: pos, Clipped: clipped}
			s.setSelection(s.anchor, extent, s.visibleOnlyForExtent(extent))
			if s.autoScroll.Target == nil && s.anchor.Clipped {
				s.startAutoScroll(ctx, mouse, scrollTarget, scrollOff)
			}
		} else if extent, ok := area.selectionBoundaryAt(Point{X: mouse.Col, Y: mouse.Row}); ok {
			s.setSelection(s.anchor, extent, s.visibleOnlyForExtent(extent))
		}
		s.updateAutoScroll(ctx, mouse)
		return EventHandled
	case EventRelease:
		if s.selecting {
			s.stopSelecting(ctx)
			return EventHandled
		}
	}
	return EventIgnored
}

func (s *selectionAreaState) TickFrame(now time.Time) bool {
	if !s.selecting || s.autoScroll.Target == nil || s.autoScroll.Velocity == 0 {
		s.autoScroll.LastTick = time.Time{}
		return false
	}
	if s.autoScroll.LastTick.IsZero() {
		s.autoScroll.LastTick = now
		return true
	}
	elapsed := now.Sub(s.autoScroll.LastTick).Seconds()
	s.autoScroll.LastTick = now
	s.autoScroll.Pending += math.Abs(s.autoScroll.Velocity) * elapsed
	lines := int(s.autoScroll.Pending)
	if lines <= 0 {
		return true
	}
	s.autoScroll.Pending -= float64(lines)
	if s.autoScroll.Velocity < 0 {
		lines = -lines
	}
	if !s.autoScroll.Target.ScrollByLines(lines) {
		return true
	}
	s.extendSelectionToAutoScrollEdge()
	return true
}

func (s *selectionAreaState) mouseClickCount(mouse Mouse) int {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if s.clickCount == 0 || mouse.Row != s.lastClickRow || mouse.Col != s.lastClickCol || now.Sub(s.lastClick) > textEditorMultiClickInterval {
		s.clickCount = 1
	} else {
		s.clickCount++
	}
	s.lastClick = now
	s.lastClickRow = mouse.Row
	s.lastClickCol = mouse.Col
	return s.clickCount
}

func (s *selectionAreaState) areaRender() *renderSelectionArea {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderSelectionArea); ok {
		return r
	}
	return nil
}

func (s *selectionAreaState) setSelection(anchor, extent selectionEndpoint, visibleOnly bool) {
	s.anchor = anchor
	s.extent = extent
	s.hasSelection = true
	s.visibleOnly = visibleOnly
	area := s.areaRender()
	if area == nil {
		return
	}
	s.selected = area.ApplySelection(anchor, extent, textFieldTheme(MustDepend[Theme](s.Context())).Selection, s.selected, visibleOnly)
}

func (s *selectionAreaState) startAutoScroll(ctx EventContext, mouse Mouse, target selectionAutoScroller, off Offset) {
	if target == nil {
		s.stopAutoScroll()
		return
	}
	s.autoScroll = selectionAutoScroll{Target: target, Offset: off, StartedOffset: target.ScrollMetrics().ScrollOffset}
	s.updateAutoScroll(ctx, mouse)
}

func (s *selectionAreaState) updateAutoScroll(ctx EventContext, mouse Mouse) {
	if s.autoScroll.Target == nil {
		return
	}
	fractional := ctx.FractionalMousePoint(mouse)
	rowFraction := fractional.Row - math.Floor(fractional.Row)
	colFraction := fractional.Col - math.Floor(fractional.Col)
	s.autoScroll.Mouse = FractionalMousePoint{
		Col: float64(mouse.Col) + colFraction,
		Row: float64(mouse.Row) + rowFraction,
	}
	s.autoScroll.Velocity = selectionAutoScrollVelocity(s.autoScroll.Mouse.Row, s.autoScroll.Offset, s.autoScroll.Target.ScrollMetrics())
	if s.autoScroll.Velocity != 0 && ctx.app != nil {
		ctx.app.RequestFrame()
	}
}

func (s *selectionAreaState) stopAutoScroll() {
	s.autoScroll = selectionAutoScroll{}
}

func (s *selectionAreaState) stopSelecting(ctx EventContext) {
	s.selecting = false
	s.stopAutoScroll()
	if ctx.app != nil {
		ctx.app.releaseMouseCapture(s.element)
	}
}

func (s *selectionAreaState) visibleOnlyForExtent(extent selectionEndpoint) bool {
	visibleOnly := s.anchor.Clipped || extent.Clipped
	if s.autoScroll.Target != nil && s.autoScroll.Target.ScrollMetrics().ScrollOffset != s.autoScroll.StartedOffset {
		visibleOnly = false
	}
	return visibleOnly
}

func (s *selectionAreaState) extendSelectionToAutoScrollEdge() {
	area := s.areaRender()
	if area == nil || s.autoScroll.Target == nil {
		return
	}
	metrics := s.autoScroll.Target.ScrollMetrics()
	if metrics.ViewportHeight <= 0 {
		return
	}
	row := s.autoScroll.Offset.Y
	aboveViewport := s.autoScroll.Mouse.Row < float64(s.autoScroll.Offset.Y)
	belowViewport := s.autoScroll.Mouse.Row >= float64(s.autoScroll.Offset.Y+metrics.ViewportHeight)
	if s.autoScroll.Velocity > 0 {
		row += metrics.ViewportHeight - 1
	}
	col := int(math.Floor(s.autoScroll.Mouse.Col))
	if col < s.autoScroll.Offset.X {
		col = s.autoScroll.Offset.X
	}
	if col >= s.autoScroll.Offset.X+metrics.ViewportWidth {
		col = s.autoScroll.Offset.X + metrics.ViewportWidth - 1
	}
	selectable, off, local, clipped, _, _, ok := area.selectableAt(Point{X: col, Y: row})
	if !ok {
		return
	}
	var pos TextPosition
	if belowViewport {
		pos = selectable.EndPosition()
	} else if aboveViewport {
		pos = selectable.StartPosition()
	} else {
		var ok bool
		pos, ok = selectable.PositionForPoint(local)
		if !ok {
			return
		}
	}
	extent := selectionEndpoint{Selectable: selectable, Offset: off, Position: pos, Clipped: clipped}
	s.setSelection(s.anchor, extent, false)
}

func (s *selectionAreaState) setTextSelection(selectable selectableTextRender, off Offset, selection TextSelection, visibleOnly bool) {
	s.setSelection(
		selectionEndpoint{Selectable: selectable, Offset: off, Position: selection.Base, Clipped: visibleOnly},
		selectionEndpoint{Selectable: selectable, Offset: off, Position: selection.Extent, Clipped: visibleOnly},
		visibleOnly,
	)
}

func (s *selectionAreaState) clearSelection() {
	for _, selected := range s.selected {
		selected.SetSelection(TextSelection{}, Style{})
	}
	s.selected = nil
	s.hasSelection = false
	s.visibleOnly = false
	s.selecting = false
	s.stopAutoScroll()
}

type selectionAreaView struct {
	State *selectionAreaState
	Child Widget
}

func (w selectionAreaView) WidgetChild() Widget {
	return w.Child
}

func (w selectionAreaView) CreateRenderObject(BuildContext) RenderObject {
	return &renderSelectionArea{State: w.State}
}

func (w selectionAreaView) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderSelectionArea)
	r.State = w.State
}

type renderSelectionArea struct {
	SingleChildRenderObject
	State *selectionAreaState
}

func (r *renderSelectionArea) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.SetSize(c.Constrain(Size{}))
		return
	}
	child.Layout(ctx, c)
	r.SetSize(c.Constrain(child.Base().Size()))
}

func (r *renderSelectionArea) DryLayout(ctx LayoutContext, c Constraints) Size {
	return DryLayout(ctx, r.Child(), c)
}

func (r *renderSelectionArea) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderSelectionArea) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSelectionArea) selectableAt(pt Point) (selectableTextRender, Offset, Point, bool, selectionAutoScroller, Offset, bool) {
	child := r.Child()
	if child == nil {
		return nil, Offset{}, Point{}, false, nil, Offset{}, false
	}
	return selectableInRender(child, pt, Offset{}, false, nil, Offset{})
}

func (r *renderSelectionArea) selectionBoundaryAt(pt Point) (selectionEndpoint, bool) {
	items := r.selectablesForSelection(true)
	if len(items) == 0 {
		return selectionEndpoint{}, false
	}
	for _, item := range items {
		rect := selectableRect(item.Selectable, item.Offset)
		if pt.Y < rect.Y {
			return selectionEndpoint{
				Selectable: item.Selectable,
				Offset:     item.Offset,
				Position:   item.Selectable.StartPosition(),
				Clipped:    item.Clipped,
			}, true
		}
		if pt.Y < rect.Y+rect.Height {
			position := item.Selectable.EndPosition()
			if pt.X < rect.X {
				position = item.Selectable.StartPosition()
			}
			return selectionEndpoint{
				Selectable: item.Selectable,
				Offset:     item.Offset,
				Position:   position,
				Clipped:    item.Clipped,
			}, true
		}
	}
	item := items[len(items)-1]
	return selectionEndpoint{
		Selectable: item.Selectable,
		Offset:     item.Offset,
		Position:   item.Selectable.EndPosition(),
		Clipped:    item.Clipped,
	}, true
}

func (r *renderSelectionArea) ApplySelection(anchor, extent selectionEndpoint, style Style, previous []selectableTextRender, visibleOnly bool) []selectableTextRender {
	for _, selected := range previous {
		selected.SetSelection(TextSelection{}, Style{})
	}
	items := r.selectablesForSelection(visibleOnly)
	start, end, forward, ok := selectionEndpointRange(items, anchor, extent)
	if !ok {
		return nil
	}
	selected := make([]selectableTextRender, 0, end-start+1)
	for i := start; i <= end; i++ {
		var selection TextSelection
		switch {
		case start == end:
			if forward {
				selection = TextSelection{Base: anchor.Position, Extent: extent.Position}
			} else {
				selection = TextSelection{Base: extent.Position, Extent: anchor.Position}
			}
		case i == start:
			if forward {
				selection = TextSelection{Base: anchor.Position, Extent: items[i].Selectable.EndPosition()}
			} else {
				selection = TextSelection{Base: extent.Position, Extent: items[i].Selectable.EndPosition()}
			}
		case i == end:
			if forward {
				selection = TextSelection{Base: items[i].Selectable.StartPosition(), Extent: extent.Position}
			} else {
				selection = TextSelection{Base: items[i].Selectable.StartPosition(), Extent: anchor.Position}
			}
		default:
			selection = items[i].Selectable.SelectAll()
		}
		items[i].Selectable.SetSelection(selection, style)
		selected = append(selected, items[i].Selectable)
	}
	return selected
}

func (r *renderSelectionArea) SelectedText(anchor, extent selectionEndpoint, visibleOnly bool) string {
	items := r.selectablesForSelection(visibleOnly)
	start, end, forward, ok := selectionEndpointRange(items, anchor, extent)
	if !ok {
		return ""
	}
	out := ""
	for i := start; i <= end; i++ {
		var selection TextSelection
		switch {
		case start == end:
			if forward {
				selection = TextSelection{Base: anchor.Position, Extent: extent.Position}
			} else {
				selection = TextSelection{Base: extent.Position, Extent: anchor.Position}
			}
		case i == start:
			if forward {
				selection = TextSelection{Base: anchor.Position, Extent: items[i].Selectable.EndPosition()}
			} else {
				selection = TextSelection{Base: extent.Position, Extent: items[i].Selectable.EndPosition()}
			}
		case i == end:
			if forward {
				selection = TextSelection{Base: items[i].Selectable.StartPosition(), Extent: extent.Position}
			} else {
				selection = TextSelection{Base: items[i].Selectable.StartPosition(), Extent: anchor.Position}
			}
		default:
			selection = items[i].Selectable.SelectAll()
		}
		if i > start && items[i].Offset.Y != items[i-1].Offset.Y {
			out += "\n"
		}
		out += items[i].Selectable.SelectedText(selection)
	}
	return out
}

func (r *renderSelectionArea) selectAllEndpoints() (selectionEndpoint, selectionEndpoint, bool) {
	items := r.selectables()
	if len(items) == 0 {
		return selectionEndpoint{}, selectionEndpoint{}, false
	}
	return selectionEndpoint{
			Selectable: items[0].Selectable,
			Offset:     items[0].Offset,
			Position:   items[0].Selectable.StartPosition(),
		}, selectionEndpoint{
			Selectable: items[len(items)-1].Selectable,
			Offset:     items[len(items)-1].Offset,
			Position:   items[len(items)-1].Selectable.EndPosition(),
		}, true
}

func (r *renderSelectionArea) selectables() []selectionItem {
	return r.selectablesForSelection(false)
}

func (r *renderSelectionArea) selectablesForSelection(visibleOnly bool) []selectionItem {
	child := r.Child()
	if child == nil {
		return nil
	}
	var out []selectionItem
	collectSelectables(child, Offset{}, Rect{}, false, visibleOnly, &out)
	return out
}

type selectionEndpoint struct {
	Selectable selectableTextRender
	Offset     Offset
	Position   TextPosition
	Clipped    bool
}

type selectionItem struct {
	Selectable selectableTextRender
	Offset     Offset
	Clipped    bool
}

type selectionAutoScroll struct {
	Target        selectionAutoScroller
	Offset        Offset
	StartedOffset int
	Mouse         FractionalMousePoint
	Velocity      float64
	Pending       float64
	LastTick      time.Time
}

type selectableTextRender interface {
	RenderObject
	PositionForPoint(Point) (TextPosition, bool)
	StartPosition() TextPosition
	EndPosition() TextPosition
	SelectAll() TextSelection
	SelectWordAt(TextPosition) TextSelection
	SelectLineAt(TextPosition) TextSelection
	SelectedText(TextSelection) string
	SetSelection(TextSelection, Style)
}

type selectionAutoScroller interface {
	ScrollMetrics() ScrollMetrics
	ScrollByLines(int) bool
}

func collectSelectables(ro RenderObject, off Offset, clip Rect, hasClip bool, visibleOnly bool, out *[]selectionItem) {
	if selectionDisabled(ro) {
		return
	}
	if visibleOnly {
		if provider, ok := ro.(selectionClipProvider); ok {
			nextClip := rectAtOffset(provider.SelectionClip(), off)
			if hasClip {
				nextClip = intersectRect(clip, nextClip)
			}
			clip = nextClip
			hasClip = true
		}
	}
	if selectable, ok := ro.(selectableTextRender); ok {
		if visibleOnly && hasClip && !rectsIntersect(selectableRect(selectable, off), clip) {
			return
		}
		*out = append(*out, selectionItem{Selectable: selectable, Offset: off, Clipped: hasClip})
		return
	}
	ro.VisitChildren(func(child RenderObject) {
		childOff := Offset{}
		if !visibleOnly {
			if provider, ok := ro.(selectionChildOffsetProvider); ok {
				childOff = provider.SelectionChildOffset(child)
			} else if provider, ok := ro.(ChildOffsetProvider); ok {
				childOff = provider.ChildOffset(child)
			}
		} else if provider, ok := ro.(ChildOffsetProvider); ok {
			childOff = provider.ChildOffset(child)
		}
		collectSelectables(child, off.Add(childOff), clip, hasClip, visibleOnly, out)
	})
}

func selectionEndpointRange(items []selectionItem, anchor, extent selectionEndpoint) (start, end int, forward bool, ok bool) {
	anchorIndex, extentIndex := -1, -1
	for i, item := range items {
		if item.Selectable == anchor.Selectable {
			anchorIndex = i
		}
		if item.Selectable == extent.Selectable {
			extentIndex = i
		}
	}
	if anchorIndex < 0 || extentIndex < 0 {
		return 0, 0, true, false
	}
	forward = anchorIndex < extentIndex || anchorIndex == extentIndex && compareTextPosition(anchor.Position, extent.Position) <= 0
	if forward {
		return anchorIndex, extentIndex, true, true
	}
	return extentIndex, anchorIndex, false, true
}

func selectableInRender(ro RenderObject, pt Point, off Offset, clipped bool, scroller selectionAutoScroller, scrollerOff Offset) (selectableTextRender, Offset, Point, bool, selectionAutoScroller, Offset, bool) {
	if selectionDisabled(ro) {
		return nil, Offset{}, Point{}, false, nil, Offset{}, false
	}
	if _, ok := ro.(selectionClipProvider); ok {
		clipped = true
	}
	if target, ok := ro.(selectionAutoScroller); ok {
		scroller = target
		scrollerOff = off
	}
	if selectable, ok := ro.(selectableTextRender); ok {
		size := ro.Base().Size()
		if pt.X < 0 || pt.Y < 0 || pt.X > size.Width || pt.Y >= size.Height {
			return nil, Offset{}, Point{}, false, nil, Offset{}, false
		}
		return selectable, off, pt, clipped, scroller, scrollerOff, true
	}
	if pt.X < 0 || pt.Y < 0 || pt.X >= ro.Base().Size().Width || pt.Y >= ro.Base().Size().Height {
		return nil, Offset{}, Point{}, false, nil, Offset{}, false
	}
	var found selectableTextRender
	var foundOff Offset
	var foundPt Point
	var foundClipped bool
	var foundScroller selectionAutoScroller
	var foundScrollerOff Offset
	ro.VisitChildren(func(child RenderObject) {
		if found != nil {
			return
		}
		childOff := Offset{}
		if provider, ok := ro.(ChildOffsetProvider); ok {
			childOff = provider.ChildOffset(child)
		}
		local := Point{X: pt.X - childOff.X, Y: pt.Y - childOff.Y}
		nextOff := off.Add(childOff)
		found, foundOff, foundPt, foundClipped, foundScroller, foundScrollerOff, _ = selectableInRender(child, local, nextOff, clipped, scroller, scrollerOff)
	})
	return found, foundOff, foundPt, foundClipped, foundScroller, foundScrollerOff, found != nil
}

type selectionDisabler interface {
	SelectionDisabled() bool
}

type selectionClipProvider interface {
	SelectionClip() Rect
}

type selectionChildOffsetProvider interface {
	SelectionChildOffset(RenderObject) Offset
}

type selectionSizeProvider interface {
	SelectionSize() Size
}

func selectionDisabled(ro RenderObject) bool {
	d, ok := ro.(selectionDisabler)
	return ok && d.SelectionDisabled()
}

func selectionSize(ro RenderObject) Size {
	if provider, ok := ro.(selectionSizeProvider); ok {
		return provider.SelectionSize()
	}
	return ro.Base().Size()
}

func rectAtOffset(r Rect, off Offset) Rect {
	return Rect{X: off.X + r.X, Y: off.Y + r.Y, Width: r.Width, Height: r.Height}
}

func selectableRect(ro selectableTextRender, off Offset) Rect {
	size := ro.Base().Size()
	return Rect{X: off.X, Y: off.Y, Width: max(1, size.Width), Height: max(1, size.Height)}
}

func rectsIntersect(a, b Rect) bool {
	intersection := intersectRect(a, b)
	return intersection.Width > 0 && intersection.Height > 0
}

func selectionAutoScrollVelocity(row float64, off Offset, metrics ScrollMetrics) float64 {
	if metrics.ViewportHeight <= 0 {
		return 0
	}
	relative := row - float64(off.Y)
	var depth float64
	var sign float64
	switch {
	case relative < 0.5:
		depth = 0.5 - relative
		sign = -1
	case relative >= float64(metrics.ViewportHeight)-0.5:
		depth = relative - (float64(metrics.ViewportHeight) - 0.5)
		sign = 1
	default:
		return 0
	}
	if depth <= 0 {
		return 0
	}
	const maxLinesPerSecond = 80.0
	scaled := math.Pow(minFloat(depth/2.5, 1), 1.5)
	return sign * maxLinesPerSecond * scaled
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
