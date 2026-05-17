package ui

type ButtonWidget struct {
	Label     string
	OnPressed VoidCallback
}

func Button(label string, onPressed VoidCallback) Widget {
	return ButtonWidget{Label: label, OnPressed: onPressed}
}
func (w ButtonWidget) CreateState() State { return &buttonState{} }

type buttonState struct {
	StateBase
	node FocusNode
}

func (s *buttonState) Build(ctx BuildContext) Widget {
	w := s.Widget().(ButtonWidget)
	s.node.onChange = s.MarkNeedsBuild
	style := MustDepend[Theme](ctx).Button.Normal
	if s.node.HasFocus() {
		style = MustDepend[Theme](ctx).Button.Focused
	}
	return Focus(&s.node, Padding(Symmetric(1, 0), Text(w.Label, TextStyle(style))))
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
	case Mouse:
		if ev.EventType != EventPress || ev.Button != MouseLeftButton {
			return EventIgnored
		}
	default:
		return EventIgnored
	}
	if cb := s.Widget().(ButtonWidget).OnPressed; cb != nil {
		cb(ctx)
	}
	return EventHandled
}
