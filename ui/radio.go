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
	selectControlState
}

func (s *radioState[T]) Build(ctx BuildContext) Widget {
	w := s.Widget().(Radio[T])
	return Focus(&s.node, RichText{Spans: radioSpans(w, s.styles(ctx))})
}

func radioSpans[T comparable](w Radio[T], styles selectControlStyles) []TextSpan {
	mark := " "
	if w.Value == w.GroupValue {
		mark = "•"
	}
	return selectControlSpans("(", mark, ")", w.Label, styles)
}

func (s *radioState[T]) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	return s.mouseShape()
}

func (s *radioState[T]) HandleEvent(ctx EventContext, ev Event) EventResult {
	if s.handleEvent(ctx, ev) == EventIgnored {
		return EventIgnored
	}
	w := s.Widget().(Radio[T])
	if w.OnChanged != nil {
		w.OnChanged(ctx, w.Value)
	}
	return EventHandled
}
