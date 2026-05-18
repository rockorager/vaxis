package ui

type Button struct {
	Label     string
	OnPressed VoidCallback
	Padding   Insets
	MinWidth  int
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
	padding := buttonPadding(w, theme)
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
	return Focus(&s.node, SizedBox{Width: buttonWidthFor(w.Label, padding, buttonMinWidth(w, theme)), Height: buttonHeightFor(padding), Child: DecoratedBox(
		Decoration{Style: style},
		Padding(padding, Align{Alignment: CenterAlign, Child: RichText{Spans: []TextSpan{
			{Text: left, Style: style},
			{Text: w.Label, Style: style},
			{Text: right, Style: style},
		}}}),
	)})
}

func buttonWidth(label string) int {
	w := buttonWidthFor(label, DefaultTheme().Button.Padding, DefaultTheme().Button.MinWidth)
	return w
}

func buttonWidthFor(label string, padding Insets, minWidth int) int {
	if minWidth <= 0 {
		minWidth = DefaultTheme().Button.MinWidth
	}
	return max(minWidth, textWidth(label)+2+padding.Left+padding.Right)
}

func buttonHeightFor(padding Insets) int { return 1 + padding.Top + padding.Bottom }

func buttonPadding(w Button, theme ButtonTheme) Insets {
	if w.Padding != (Insets{}) {
		return w.Padding
	}
	if theme.Padding == (Insets{}) {
		return DefaultTheme().Button.Padding
	}
	return theme.Padding
}

func buttonMinWidth(w Button, theme ButtonTheme) int {
	if w.MinWidth > 0 {
		return w.MinWidth
	}
	if theme.MinWidth <= 0 {
		return DefaultTheme().Button.MinWidth
	}
	return theme.MinWidth
}

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
