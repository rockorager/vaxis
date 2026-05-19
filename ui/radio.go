package ui

// ValueChangedCallback receives a controlled value change.
type ValueChangedCallback[T comparable] func(EventContext, T)

// Radio is a controlled mutually exclusive selection input.
//
// Radio is selected when Value equals GroupValue. Activating the control by
// mouse, Enter, or Space calls OnChanged with Value. The caller owns updating
// GroupValue with the new value.
type Radio[T comparable] struct {
	// Value is the value represented by this radio.
	Value T
	// GroupValue is the currently selected value for the radio group.
	GroupValue T
	// Label is painted after the radio when non-empty.
	Label string
	// OnChanged is called with Value when the radio is activated.
	OnChanged ValueChangedCallback[T]
}

func (w Radio[T]) CreateState() State {
	return &radioState[T]{}
}

type radioState[T comparable] struct {
	StateBase
	node    FocusNode
	hovered bool
}

func (s *radioState[T]) Build(ctx BuildContext) Widget {
	w := s.Widget().(Radio[T])
	s.node.onChange = s.MarkNeedsBuild
	theme := MustDepend[Theme](ctx).Button
	labelStyle := theme.Normal
	boxStyle := theme.Normal
	if s.node.HasFocus() {
		boxStyle = theme.Focused
	}
	if s.hovered {
		boxStyle = theme.Hovered
	}
	return Focus(&s.node, RichText{Spans: radioSpans(w, boxStyle, labelStyle, s.node.HasFocus())})
}

func radioSpans[T comparable](w Radio[T], boxStyle, labelStyle Style, focused bool) []TextSpan {
	mark := " "
	if w.Value == w.GroupValue {
		mark = "•"
	}
	markStyle := boxStyle
	if focused {
		markStyle.UnderlineStyle = UnderlineSingle
	}
	spans := []TextSpan{
		{Text: "(", Style: boxStyle},
		{Text: mark, Style: markStyle},
		{Text: ")", Style: boxStyle},
	}
	if w.Label == "" {
		return spans
	}
	return append(spans, TextSpan{Text: " " + w.Label, Style: labelStyle})
}

func (s *radioState[T]) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	shape := MustDepend[Theme](s.Context()).Button.Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *radioState[T]) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if keyIsRelease(ev) {
			return EventIgnored
		}
		if !ev.MatchString("Enter") && !ev.MatchString("Space") {
			return EventIgnored
		}
	case hoverExit:
		if s.hovered {
			s.SetState(func() { s.hovered = false })
		}
		return EventIgnored
	case Mouse:
		if ev.EventType == EventMotion {
			if !s.hovered {
				s.SetState(func() { s.hovered = true })
			}
			return EventIgnored
		}
		if ev.EventType != EventPress || ev.Button != MouseLeftButton {
			return EventIgnored
		}
	default:
		return EventIgnored
	}
	w := s.Widget().(Radio[T])
	if w.OnChanged != nil {
		w.OnChanged(ctx, w.Value)
	}
	return EventHandled
}
