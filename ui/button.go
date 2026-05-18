package ui

type Button struct {
	Label     string
	OnPressed VoidCallback
}

func (w Button) CreateState() State { return &buttonState{} }

type buttonState struct {
	StateBase
	node    FocusNode
	hovered bool
}

func (s *buttonState) Build(ctx BuildContext) Widget {
	w := s.Widget().(Button)
	s.node.onChange = s.MarkNeedsBuild
	theme := MustDepend[Theme](ctx).Button
	style := theme.Normal
	left, right := " ", " "
	if s.node.HasFocus() {
		style = theme.Focused
		left = focusMarker(theme.FocusLeft, "[")
		right = focusMarker(theme.FocusRight, "]")
	}
	if s.hovered {
		style = theme.Hovered
	}
	return Focus(&s.node, SizedBox{Width: buttonWidth(w.Label), Height: 1, Child: DecoratedBox(
		Decoration{Style: style},
		Align{Alignment: CenterAlign, Child: RichText{Spans: []TextSpan{
			{Text: left, Style: style},
			{Text: " " + w.Label + " ", Style: style},
			{Text: right, Style: style},
		}}},
	)})
}

func buttonWidth(label string) int { return max(5, textWidth(label)+4) }

func focusMarker(ch Character, fallback string) string {
	if ch == (Character{}) {
		return fallback
	}
	return ch.Grapheme
}

func textWidth(s string) int {
	w := 0
	for _, ch := range vaxisCharacters(s) {
		w += ch.Width
	}
	return w
}

func (s *buttonState) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	shape := MustDepend[Theme](s.Context()).Button.Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *buttonState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
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
	if cb := s.Widget().(Button).OnPressed; cb != nil {
		cb(ctx)
	}
	return EventHandled
}
