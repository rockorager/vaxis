package ui

// SegmentedItem describes one option in a SegmentedControl.
type SegmentedItem[T comparable] struct {
	// Value is reported through OnChanged when the segment is selected.
	Value T
	// Label is the text shown for the segment.
	Label string
	// Disabled prevents mouse and keyboard selection for this segment.
	Disabled bool
}

// SegmentedControl is a controlled single-selection input rendered as a compact
// horizontal group.
//
// Arrow keys and h/l move to the previous or next enabled segment. Mouse clicks,
// Enter, and Space select the active segment. The caller owns updating Value in
// response to OnChanged.
type SegmentedControl[T comparable] struct {
	// Value is the currently selected value.
	Value T
	// Segments is the ordered list of selectable segments.
	Segments []SegmentedItem[T]
	// Disabled prevents focus, hover, and activation for the whole control.
	Disabled bool
	// OnChanged is called when an enabled segment is selected.
	OnChanged ValueChangedCallback[T]
}

func (w SegmentedControl[T]) CreateState() State {
	return &segmentedControlState[T]{active: -1, hovered: -1}
}

type segmentedControlState[T comparable] struct {
	StateBase
	node    FocusNode
	active  int
	hovered int
}

func (s *segmentedControlState[T]) Build(ctx BuildContext) Widget {
	w := s.Widget().(SegmentedControl[T])
	s.node.onChange = s.MarkNeedsBuild
	active := s.activeIndex(w)
	render := segmentedControlRenderWidget[T]{
		Value:    w.Value,
		Segments: w.Segments,
		Disabled: w.Disabled,
		Theme:    MustDepend[Theme](ctx).SegmentedControl,
		Focused:  !w.Disabled && s.node.HasFocus(),
		Active:   active,
		Hovered:  s.hovered,
	}
	if w.Disabled || len(w.Segments) == 0 {
		return render
	}
	return Focus(&s.node, render)
}

func (s *segmentedControlState[T]) activeIndex(w SegmentedControl[T]) int {
	if s.active >= 0 && s.active < len(w.Segments) && !w.Segments[s.active].Disabled {
		return s.active
	}
	if selected := segmentedControlSelectedIndex(w.Value, w.Segments); selected >= 0 && !w.Segments[selected].Disabled {
		return selected
	}
	return nextEnabledSegment(w.Segments, -1, 1)
}

func (s *segmentedControlState[T]) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	w := s.Widget().(SegmentedControl[T])
	if w.Disabled || len(w.Segments) == 0 || s.segmentAt(mouse.Col) < 0 {
		return MouseShapeDefault
	}
	shape := MustDepend[Theme](s.Context()).SegmentedControl.Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *segmentedControlState[T]) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(SegmentedControl[T])
	if w.Disabled || len(w.Segments) == 0 {
		return EventIgnored
	}
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if keyIsRelease(ev) {
			return EventIgnored
		}
		switch {
		case ev.Keycode == KeyLeft || ev.MatchString("h"):
			return s.moveActive(w, -1)
		case ev.Keycode == KeyRight || ev.MatchString("l"):
			return s.moveActive(w, 1)
		case ev.MatchString("Enter") || ev.MatchString("Space"):
			return s.selectActive(ctx, w)
		default:
			return EventIgnored
		}
	case hoverExit:
		if s.hovered >= 0 {
			s.SetState(func() { s.hovered = -1 })
		}
		return EventIgnored
	case Mouse:
		index := s.segmentAt(ev.Col)
		if ev.EventType == EventMotion {
			if s.hovered != index {
				s.SetState(func() { s.hovered = index })
			}
			return EventIgnored
		}
		if ev.EventType != EventPress || ev.Button != MouseLeftButton || index < 0 {
			return EventIgnored
		}
		return s.selectIndex(ctx, w, index)
	default:
		return EventIgnored
	}
}

func (s *segmentedControlState[T]) moveActive(w SegmentedControl[T], delta int) EventResult {
	current := s.activeIndex(w)
	next := nextEnabledSegment(w.Segments, current, delta)
	if next < 0 || next == current {
		return EventHandled
	}
	s.SetState(func() { s.active = next })
	return EventHandled
}

func (s *segmentedControlState[T]) selectActive(ctx EventContext, w SegmentedControl[T]) EventResult {
	return s.selectIndex(ctx, w, s.activeIndex(w))
}

func (s *segmentedControlState[T]) selectIndex(ctx EventContext, w SegmentedControl[T], index int) EventResult {
	if index < 0 || index >= len(w.Segments) || w.Segments[index].Disabled {
		return EventIgnored
	}
	s.SetState(func() { s.active = index })
	if w.OnChanged != nil {
		w.OnChanged(ctx, w.Segments[index].Value)
	}
	return EventHandled
}

func (s *segmentedControlState[T]) segmentAt(col int) int {
	if r := s.renderObject(); r != nil {
		return r.segmentAt(col)
	}
	return -1
}

func (s *segmentedControlState[T]) renderObject() *renderSegmentedControl[T] {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderSegmentedControl[T]); ok {
		return r
	}
	return nil
}

func segmentedControlSelectedIndex[T comparable](value T, segments []SegmentedItem[T]) int {
	for i, segment := range segments {
		if segment.Value == value {
			return i
		}
	}
	return -1
}

func nextEnabledSegment[T comparable](segments []SegmentedItem[T], current, delta int) int {
	if delta == 0 || len(segments) == 0 {
		return -1
	}
	for step := 1; step <= len(segments); step++ {
		index := (current + step*delta) % len(segments)
		if index < 0 {
			index += len(segments)
		}
		if !segments[index].Disabled {
			return index
		}
	}
	return -1
}

type segmentedControlRenderWidget[T comparable] struct {
	Value    T
	Segments []SegmentedItem[T]
	Disabled bool
	Theme    SegmentedControlTheme
	Focused  bool
	Active   int
	Hovered  int
}

func (w segmentedControlRenderWidget[T]) CreateRenderObject(BuildContext) RenderObject {
	return &renderSegmentedControl[T]{
		Value:    w.Value,
		Segments: w.Segments,
		Disabled: w.Disabled,
		Theme:    w.Theme,
		Focused:  w.Focused,
		Active:   w.Active,
		Hovered:  w.Hovered,
	}
}

func (w segmentedControlRenderWidget[T]) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderSegmentedControl[T])
	r.Value = w.Value
	r.Segments = w.Segments
	r.Disabled = w.Disabled
	r.Theme = w.Theme
	r.Focused = w.Focused
	r.Active = w.Active
	r.Hovered = w.Hovered
	r.MarkNeedsLayout()
}

type renderSegmentedControl[T comparable] struct {
	LeafRenderObject
	Value    T
	Segments []SegmentedItem[T]
	Disabled bool
	Theme    SegmentedControlTheme
	Focused  bool
	Active   int
	Hovered  int
	ranges   []segmentRange
}

type segmentRange struct {
	Start int
	End   int
}

func (r *renderSegmentedControl[T]) Layout(LayoutContext, Constraints) {
	r.computeRanges()
	width := 0
	if len(r.ranges) > 0 {
		width = r.ranges[len(r.ranges)-1].End
	}
	r.SetSize(Size{Width: width, Height: 1})
}

func (r *renderSegmentedControl[T]) DryLayout(LayoutContext, Constraints) Size {
	width := 0
	for i, segment := range r.Segments {
		if i > 0 {
			width++
		}
		width += textWidth(segment.Label) + 2
	}
	return Size{Width: width, Height: 1}
}

func (r *renderSegmentedControl[T]) Paint(p *Painter, off Offset) {
	if len(r.ranges) != len(r.Segments) {
		r.computeRanges()
	}
	for i, segment := range r.Segments {
		if i > 0 {
			p.DrawText(Offset{X: off.X + r.ranges[i].Start - 1, Y: off.Y}, "│", r.Theme.Separator)
		}
		style := r.segmentStyle(i, segment)
		x := off.X + r.ranges[i].Start
		p.DrawText(Offset{X: x, Y: off.Y}, " "+segment.Label+" ", style)
	}
}

func (r *renderSegmentedControl[T]) HitTest(*HitTestResult, Point) bool {
	return true
}

func (r *renderSegmentedControl[T]) segmentAt(col int) int {
	for i, segmentRange := range r.ranges {
		if col >= segmentRange.Start && col < segmentRange.End {
			if i < len(r.Segments) && !r.Segments[i].Disabled {
				return i
			}
			return -1
		}
	}
	return -1
}

func (r *renderSegmentedControl[T]) computeRanges() {
	r.ranges = make([]segmentRange, len(r.Segments))
	x := 0
	for i, segment := range r.Segments {
		if i > 0 {
			x++
		}
		width := textWidth(segment.Label) + 2
		r.ranges[i] = segmentRange{Start: x, End: x + width}
		x += width
	}
}

func (r *renderSegmentedControl[T]) segmentStyle(index int, segment SegmentedItem[T]) Style {
	style := r.Theme.Normal
	if segment.Value == r.Value {
		style = mergeStyle(style, r.Theme.Selected)
	}
	if index == r.Hovered && !r.Disabled && !segment.Disabled {
		style = mergeStyle(style, r.Theme.Hovered)
	}
	if index == r.Active && r.Focused && !r.Disabled && !segment.Disabled {
		style = mergeStyle(style, r.Theme.Focused)
	}
	if r.Disabled || segment.Disabled {
		style = mergeStyle(style, r.Theme.Disabled)
	}
	return style
}
