package ui

// BoolChangedCallback receives a boolean control value change.
type BoolChangedCallback func(EventContext, bool)

// Checkbox is a controlled boolean input.
//
// Checkbox calls OnChanged with the next checked value when activated by mouse,
// Enter, or Space. The caller owns updating Checked with the new value.
type Checkbox struct {
	// Checked controls whether the checkbox is painted as selected.
	Checked bool
	// Label is painted after the checkbox when non-empty.
	Label string
	// OnChanged is called with the next checked value when the checkbox is activated.
	OnChanged BoolChangedCallback
}

func (w Checkbox) CreateState() State {
	return &checkboxState{}
}

type checkboxState struct {
	StateBase
	node    FocusNode
	hovered bool
}

func (s *checkboxState) Build(ctx BuildContext) Widget {
	w := s.Widget().(Checkbox)
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
	return Focus(&s.node, RichText{Spans: checkboxSpans(w, boxStyle, labelStyle, s.node.HasFocus())})
}

func checkboxSpans(w Checkbox, boxStyle, labelStyle Style, focused bool) []TextSpan {
	mark := " "
	if w.Checked {
		mark = "✓"
	}
	markStyle := boxStyle
	if focused {
		markStyle.UnderlineStyle = UnderlineSingle
	}
	spans := []TextSpan{
		{Text: "[", Style: boxStyle},
		{Text: mark, Style: markStyle},
		{Text: "]", Style: boxStyle},
	}
	if w.Label == "" {
		return spans
	}
	return append(spans, TextSpan{Text: " " + w.Label, Style: labelStyle})
}

func (s *checkboxState) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	shape := MustDepend[Theme](s.Context()).Button.Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *checkboxState) HandleEvent(ctx EventContext, ev Event) EventResult {
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
	w := s.Widget().(Checkbox)
	if w.OnChanged != nil {
		w.OnChanged(ctx, !w.Checked)
	}
	return EventHandled
}
