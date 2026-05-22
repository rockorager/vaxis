package ui

type selectControlState struct {
	StateBase
	node    FocusNode
	hovered bool
}

type selectControlStyles struct {
	box     Style
	label   Style
	focused bool
}

func (s *selectControlState) styles(ctx BuildContext, disabled, active bool) selectControlStyles {
	s.node.onChange = s.MarkNeedsBuild
	appTheme := MustDepend[Theme](ctx)
	theme := buttonTheme(appTheme)
	styles := selectControlStyles{
		box:     theme.Normal,
		label:   theme.Normal,
		focused: !disabled && s.node.HasFocus(),
	}
	if styles.focused {
		styles.box = theme.Focused
	}
	if !disabled && s.hovered {
		styles.box = theme.Hovered
		if active {
			styles.box = Style{Foreground: appTheme.Foreground, Background: appTheme.PrimaryHovered}
		} else if styles.focused {
			styles.box = theme.FocusedHovered
		}
	}
	if disabled {
		styles.box.Foreground = appTheme.DisabledForeground
		styles.label.Foreground = appTheme.DisabledForeground
	}
	return styles
}

func selectControlSpans(left, mark, right, label string, styles selectControlStyles) []TextSpan {
	markStyle := styles.box
	if styles.focused {
		markStyle.UnderlineStyle = UnderlineSingle
	}
	spans := []TextSpan{
		{Text: left, Style: styles.box},
		{Text: mark, Style: markStyle},
		{Text: right, Style: styles.box},
	}
	if label == "" {
		return spans
	}
	return append(spans, TextSpan{Text: " " + label, Style: styles.label})
}

func (s *selectControlState) build(ctx BuildContext, disabled bool, spans []TextSpan) Widget {
	child := RichText{Spans: spans}
	if disabled {
		return child
	}
	return Focus(&s.node, child)
}

func (s *selectControlState) mouseShape(disabled bool) MouseShape {
	if disabled {
		return MouseShapeDefault
	}
	shape := buttonTheme(MustDepend[Theme](s.Context())).Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *selectControlState) handleEvent(ctx EventContext, ev Event, disabled bool) EventResult {
	if disabled {
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
	return EventHandled
}
