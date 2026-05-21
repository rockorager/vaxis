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
	// Disabled prevents focus, hover, and activation when true.
	Disabled bool
	// Label is painted after the checkbox when non-empty.
	Label string
	// OnChanged is called with the next checked value when the checkbox is activated.
	OnChanged BoolChangedCallback
}

func (w Checkbox) CreateState() State {
	return &checkboxState{}
}

type checkboxState struct {
	selectControlState
}

func (s *checkboxState) Build(ctx BuildContext) Widget {
	w := s.Widget().(Checkbox)
	child := s.build(ctx, w.Disabled, checkboxSpans(w, s.styles(ctx, w.Disabled)))
	if w.Disabled {
		return child
	}
	return DefaultActions{
		Bindings: map[IntentType]ActionFunc{
			ActivateIntentType: func(ctx EventContext, intent Intent) EventResult {
				w := s.Widget().(Checkbox)
				if w.OnChanged != nil {
					w.OnChanged(ctx, !w.Checked)
				}
				return EventHandled
			},
		},
		Child: child,
	}
}

func checkboxSpans(w Checkbox, styles selectControlStyles) []TextSpan {
	mark := " "
	if w.Checked {
		mark = "✓"
	}
	return selectControlSpans("[", mark, "]", w.Label, styles)
}

func (s *checkboxState) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	return s.mouseShape(s.Widget().(Checkbox).Disabled)
}

func (s *checkboxState) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(Checkbox)
	if s.handleEvent(ctx, ev, w.Disabled) == EventIgnored {
		return EventIgnored
	}
	return ctx.Invoke(ActivateIntent{})
}
