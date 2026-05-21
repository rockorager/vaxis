package ui

// Dialog presents modal content with trapped focus.
type Dialog struct {
	// Title is painted at the top of the dialog when non-empty.
	Title string
	// Child is the main dialog content.
	Child Widget
	// Actions are laid out horizontally at the bottom-right.
	Actions []Widget
	// Width fixes the dialog width when greater than zero.
	Width int
	// OnDismiss is called when Escape is pressed while focus is inside the dialog.
	OnDismiss VoidCallback
}

func (w Dialog) CreateState() State {
	return &dialogState{}
}

type dialogState struct {
	StateBase
}

func (s *dialogState) Build(ctx BuildContext) Widget {
	w := s.Widget().(Dialog)
	theme := MustDepend[Theme](ctx)
	body := []Widget{}
	if w.Title != "" {
		body = append(body, Text{Value: w.Title, Style: Style{Attribute: AttrBold}})
	}
	if w.Title != "" && w.Child != nil {
		body = append(body, SizedBox{Height: 1})
	}
	if w.Child != nil {
		body = append(body, w.Child)
	}
	if len(w.Actions) > 0 {
		if len(body) > 0 {
			body = append(body, SizedBox{Height: 1})
		}
		body = append(body, Flex{
			Axis:               Horizontal,
			MainAxisAlignment:  MainAxisEnd,
			CrossAxisAlignment: CrossAxisCenter,
			Children:           intersperseWidgets(w.Actions, SizedBox{Width: 1, Height: 1}),
		})
	}
	child := Widget(Flex{
		Axis:               Vertical,
		MainAxisSize:       MainAxisSizeMin,
		CrossAxisAlignment: CrossAxisStretch,
		Children:           body,
	})
	if w.Width > 0 {
		child = ConstrainedBox{Constraints: Constraints{MinWidth: w.Width, MaxWidth: w.Width}, Child: child}
	}
	return Actions{
		Bindings: map[Intent]ActionFunc{
			IntentDismiss: func(ctx EventContext) EventResult {
				if cb := w.OnDismiss; cb != nil {
					cb(ctx)
				}
				return EventHandled
			},
		},
		Child: FocusScope{
			Trap:      true,
			AutoFocus: true,
			Child: DecoratedBox(
				Decoration{
					Style:  theme.Button.Normal,
					Border: BorderAll(theme.Text),
				},
				Padding(All(1), child),
			),
		},
	}
}

func intersperseWidgets(children []Widget, separator Widget) []Widget {
	if len(children) <= 1 {
		return children
	}
	out := make([]Widget, 0, len(children)*2-1)
	for i, child := range children {
		if i > 0 {
			out = append(out, separator)
		}
		out = append(out, child)
	}
	return out
}
