package ui

import "time"

// SelectionArea enables read-only text selection for descendant Text and RichText widgets.
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
	selecting    bool
	selected     []selectableTextRender
	now          func() time.Time
	lastClick    time.Time
	lastClickRow int
	lastClickCol int
	clickCount   int
}

func (s *selectionAreaState) Build(BuildContext) Widget {
	return Focus(&s.node, selectionAreaView{
		State: s,
		Child: s.Widget().(SelectionArea).Child,
	})
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
			if s.hasSelection {
				if area := s.areaRender(); area != nil {
					ctx.Copy(area.SelectedText(s.anchor, s.extent))
				}
			}
			return EventHandled
		}
		if ev.MatchString("Ctrl+a") {
			if area := s.areaRender(); area != nil {
				if anchor, extent, ok := area.selectAllEndpoints(); ok {
					s.setSelection(anchor, extent)
				}
				return EventHandled
			}
		}
	case Mouse:
		return s.handleMouse(ev)
	}
	return EventIgnored
}

func (s *selectionAreaState) handleMouse(mouse Mouse) EventResult {
	switch mouse.EventType {
	case EventPress:
		if mouse.Button != MouseLeftButton {
			return EventIgnored
		}
		area := s.areaRender()
		if area == nil {
			return EventIgnored
		}
		selectable, off, local, ok := area.selectableAt(Point{X: mouse.Col, Y: mouse.Row})
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
			s.setTextSelection(selectable, off, selectable.SelectLineAt(pos))
		case clickCount == 2:
			s.selecting = false
			s.setTextSelection(selectable, off, selectable.SelectWordAt(pos))
		default:
			s.selecting = true
			endpoint := selectionEndpoint{Selectable: selectable, Offset: off, Position: pos}
			s.setSelection(endpoint, endpoint)
		}
		return EventHandled
	case EventMotion:
		if !s.selecting || !s.hasSelection {
			return EventIgnored
		}
		area := s.areaRender()
		if area == nil {
			return EventIgnored
		}
		selectable, off, local, ok := area.selectableAt(Point{X: mouse.Col, Y: mouse.Row})
		if !ok {
			return EventIgnored
		}
		pos, ok := selectable.PositionForPoint(local)
		if !ok {
			return EventIgnored
		}
		s.setSelection(s.anchor, selectionEndpoint{Selectable: selectable, Offset: off, Position: pos})
		return EventHandled
	case EventRelease:
		if mouse.Button == MouseLeftButton && s.selecting {
			s.selecting = false
			return EventHandled
		}
	}
	return EventIgnored
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

func (s *selectionAreaState) setSelection(anchor, extent selectionEndpoint) {
	s.anchor = anchor
	s.extent = extent
	s.hasSelection = true
	area := s.areaRender()
	if area == nil {
		return
	}
	s.selected = area.ApplySelection(anchor, extent, MustDepend[Theme](s.Context()).TextField.Selection, s.selected)
}

func (s *selectionAreaState) setTextSelection(selectable selectableTextRender, off Offset, selection TextSelection) {
	s.setSelection(
		selectionEndpoint{Selectable: selectable, Offset: off, Position: selection.Base},
		selectionEndpoint{Selectable: selectable, Offset: off, Position: selection.Extent},
	)
}

func (s *selectionAreaState) clearSelection() {
	for _, selected := range s.selected {
		selected.SetSelection(TextSelection{}, Style{})
	}
	s.selected = nil
	s.hasSelection = false
	s.selecting = false
}

type selectionAreaView struct {
	State *selectionAreaState
	Child Widget
}

func (w selectionAreaView) ChildWidget() Widget {
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

func (r *renderSelectionArea) selectableAt(pt Point) (selectableTextRender, Offset, Point, bool) {
	child := r.Child()
	if child == nil {
		return nil, Offset{}, Point{}, false
	}
	return selectableInRender(child, pt, Offset{})
}

func (r *renderSelectionArea) ApplySelection(anchor, extent selectionEndpoint, style Style, previous []selectableTextRender) []selectableTextRender {
	for _, selected := range previous {
		selected.SetSelection(TextSelection{}, Style{})
	}
	items := r.selectables()
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

func (r *renderSelectionArea) SelectedText(anchor, extent selectionEndpoint) string {
	items := r.selectables()
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
		if i > start && items[i].Offset.Y > items[i-1].Offset.Y {
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
	child := r.Child()
	if child == nil {
		return nil
	}
	var out []selectionItem
	collectSelectables(child, Offset{}, &out)
	return out
}

type selectionEndpoint struct {
	Selectable selectableTextRender
	Offset     Offset
	Position   TextPosition
}

type selectionItem struct {
	Selectable selectableTextRender
	Offset     Offset
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

func collectSelectables(ro RenderObject, off Offset, out *[]selectionItem) {
	if selectionDisabled(ro) {
		return
	}
	if selectable, ok := ro.(selectableTextRender); ok {
		*out = append(*out, selectionItem{Selectable: selectable, Offset: off})
		return
	}
	ro.VisitChildren(func(child RenderObject) {
		childOff := Offset{}
		if provider, ok := ro.(ChildOffsetProvider); ok {
			childOff = provider.ChildOffset(child)
		}
		collectSelectables(child, off.Add(childOff), out)
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

func selectableInRender(ro RenderObject, pt Point, off Offset) (selectableTextRender, Offset, Point, bool) {
	if selectionDisabled(ro) {
		return nil, Offset{}, Point{}, false
	}
	if selectable, ok := ro.(selectableTextRender); ok {
		size := ro.Base().Size()
		if pt.X < 0 || pt.Y < 0 || pt.X > size.Width || pt.Y >= size.Height {
			return nil, Offset{}, Point{}, false
		}
		return selectable, off, pt, true
	}
	if pt.X < 0 || pt.Y < 0 || pt.X >= ro.Base().Size().Width || pt.Y >= ro.Base().Size().Height {
		return nil, Offset{}, Point{}, false
	}
	var found selectableTextRender
	var foundOff Offset
	var foundPt Point
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
		found, foundOff, foundPt, _ = selectableInRender(child, local, nextOff)
	})
	return found, foundOff, foundPt, found != nil
}

type selectionDisabler interface {
	SelectionDisabled() bool
}

func selectionDisabled(ro RenderObject) bool {
	d, ok := ro.(selectionDisabler)
	return ok && d.SelectionDisabled()
}
