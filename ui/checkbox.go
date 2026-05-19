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
	selectControlState
}

func (s *checkboxState) Build(ctx BuildContext) Widget {
	w := s.Widget().(Checkbox)
	return Focus(&s.node, RichText{Spans: checkboxSpans(w, s.styles(ctx))})
}

func checkboxSpans(w Checkbox, styles selectControlStyles) []TextSpan {
	mark := " "
	if w.Checked {
		mark = "✓"
	}
	return selectControlSpans("[", mark, "]", w.Label, styles)
}

func (s *checkboxState) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	return s.mouseShape()
}

func (s *checkboxState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if s.handleEvent(ctx, ev) == EventIgnored {
		return EventIgnored
	}
	w := s.Widget().(Checkbox)
	if w.OnChanged != nil {
		w.OnChanged(ctx, !w.Checked)
	}
	return EventHandled
}
