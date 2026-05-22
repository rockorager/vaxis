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
	// Disabled prevents focus, hover, and activation when true.
	Disabled bool
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
	child := s.build(ctx, w.Disabled, radioSpans(w, s.styles(ctx, w.Disabled, w.Value == w.GroupValue)))
	if w.Disabled {
		return child
	}
	return DefaultActions{
		Bindings: map[IntentType]ActionFunc{
			ActivateIntentType: func(ctx EventContext, intent Intent) EventResult {
				w := s.Widget().(Radio[T])
				if w.OnChanged != nil {
					w.OnChanged(ctx, w.Value)
				}
				return EventHandled
			},
		},
		Child: child,
	}
}

func radioSpans[T comparable](w Radio[T], styles selectControlStyles) []TextSpan {
	mark := " "
	if w.Value == w.GroupValue {
		mark = "•"
	}
	return selectControlSpans("(", mark, ")", w.Label, styles)
}

func (s *radioState[T]) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	return s.mouseShape(s.Widget().(Radio[T]).Disabled)
}

func (s *radioState[T]) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(Radio[T])
	if s.handleEvent(ctx, ev, w.Disabled) == EventIgnored {
		return EventIgnored
	}
	return ctx.Invoke(ActivateIntent{})
}
