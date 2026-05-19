package ui

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
	active       selectableTextRender
	activeOffset Offset
	selection    TextSelection
	selecting    bool
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
			if s.active != nil && !s.selection.IsCollapsed() {
				ctx.Copy(s.active.SelectedText(s.selection))
			}
			return EventHandled
		}
		if ev.MatchString("Ctrl+a") {
			if s.active != nil {
				s.setSelection(s.active, s.activeOffset, s.active.SelectAll())
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
		s.selecting = true
		s.setSelection(selectable, off, TextSelection{Base: pos, Extent: pos})
		return EventHandled
	case EventMotion:
		if !s.selecting || s.active == nil {
			return EventIgnored
		}
		pos, ok := s.active.PositionForPoint(Point{X: mouse.Col - s.activeOffset.X, Y: mouse.Row - s.activeOffset.Y})
		if !ok {
			return EventIgnored
		}
		s.setSelection(s.active, s.activeOffset, TextSelection{Base: s.selection.Base, Extent: pos})
		return EventHandled
	case EventRelease:
		if mouse.Button == MouseLeftButton && s.selecting {
			s.selecting = false
			return EventHandled
		}
	}
	return EventIgnored
}

func (s *selectionAreaState) areaRender() *renderSelectionArea {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderSelectionArea); ok {
		return r
	}
	return nil
}

func (s *selectionAreaState) setSelection(active selectableTextRender, off Offset, selection TextSelection) {
	if s.active != nil && s.active != active {
		s.active.SetSelection(TextSelection{}, Style{})
	}
	s.active = active
	s.activeOffset = off
	s.selection = selection
	active.SetSelection(selection, MustDepend[Theme](s.Context()).TextField.Selection)
}

func (s *selectionAreaState) clearSelection() {
	if s.active != nil {
		s.active.SetSelection(TextSelection{}, Style{})
	}
	s.active = nil
	s.selection = TextSelection{}
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

type selectableTextRender interface {
	RenderObject
	PositionForPoint(Point) (TextPosition, bool)
	SelectAll() TextSelection
	SelectedText(TextSelection) string
	SetSelection(TextSelection, Style)
}

func selectableInRender(ro RenderObject, pt Point, off Offset) (selectableTextRender, Offset, Point, bool) {
	if pt.X < 0 || pt.Y < 0 || pt.X >= ro.Base().Size().Width || pt.Y >= ro.Base().Size().Height {
		return nil, Offset{}, Point{}, false
	}
	if selectable, ok := ro.(selectableTextRender); ok {
		return selectable, off, pt, true
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
