package ui

type TextChangedCallback func(EventContext, string)

type TextField struct {
	Value       string
	Placeholder string
	OnChanged   TextChangedCallback
	Padding     Insets
	MinWidth    int
}

func (w TextField) CreateState() State { return &textFieldState{} }

type textFieldState struct {
	StateBase
	node   FocusNode
	cursor int
}

func (s *textFieldState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextField)
	s.node.onChange = s.MarkNeedsBuild
	chars := vaxisCharacters(w.Value)
	if s.cursor > len(chars) {
		s.cursor = len(chars)
	}
	theme := MustDepend[Theme](ctx).TextField
	padding := textFieldPadding(w, theme)
	width := max(textFieldMinWidth(w, theme), textWidth(w.Value)+1+padding.Left+padding.Right)
	style := theme.Normal
	if s.node.HasFocus() {
		style = theme.Focused
	}
	return Focus(&s.node, SizedBox{Width: width, Height: 1 + padding.Top + padding.Bottom, Child: DecoratedBox(
		Decoration{Style: style},
		Padding(padding, s.content(w, chars, style, theme)),
	)})
}

func (s *textFieldState) content(w TextField, chars []Character, style Style, theme TextFieldTheme) Widget {
	if w.Value == "" && !s.node.HasFocus() && w.Placeholder != "" {
		return Text{Value: w.Placeholder, Style: mergeStyle(style, theme.Placeholder), Overflow: TextOverflowClip}
	}
	spans := make([]TextSpan, 0, 3)
	before := charactersString(chars[:s.cursor])
	if before != "" {
		spans = append(spans, TextSpan{Text: before, Style: style})
	}
	if s.node.HasFocus() {
		cursor := " "
		if s.cursor < len(chars) {
			cursor = chars[s.cursor].Grapheme
		}
		spans = append(spans, TextSpan{Text: cursor, Style: theme.Cursor})
		if s.cursor < len(chars) {
			if after := charactersString(chars[s.cursor+1:]); after != "" {
				spans = append(spans, TextSpan{Text: after, Style: style})
			}
		}
	} else if after := charactersString(chars[s.cursor:]); after != "" {
		spans = append(spans, TextSpan{Text: after, Style: style})
	}
	return RichText{Spans: spans, Overflow: TextOverflowClip}
}

func (s *textFieldState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	w := s.Widget().(TextField)
	chars := vaxisCharacters(w.Value)
	if s.cursor > len(chars) {
		s.cursor = len(chars)
	}
	switch {
	case key.Keycode == KeyLeft:
		if s.cursor > 0 {
			s.SetState(func() { s.cursor-- })
		}
		return EventHandled
	case key.Keycode == KeyRight:
		if s.cursor < len(chars) {
			s.SetState(func() { s.cursor++ })
		}
		return EventHandled
	case key.Keycode == KeyHome:
		s.SetState(func() { s.cursor = 0 })
		return EventHandled
	case key.Keycode == KeyEnd:
		s.SetState(func() { s.cursor = len(chars) })
		return EventHandled
	case key.Keycode == KeyBackspace:
		if s.cursor > 0 {
			chars = append(chars[:s.cursor-1], chars[s.cursor:]...)
			s.cursor--
			s.change(ctx, charactersString(chars))
		}
		return EventHandled
	case key.Keycode == KeyDelete:
		if s.cursor < len(chars) {
			chars = append(chars[:s.cursor], chars[s.cursor+1:]...)
			s.change(ctx, charactersString(chars))
		}
		return EventHandled
	case key.Text != "":
		insert := vaxisCharacters(key.Text)
		chars = append(append(append([]Character{}, chars[:s.cursor]...), insert...), chars[s.cursor:]...)
		s.cursor += len(insert)
		s.change(ctx, charactersString(chars))
		return EventHandled
	}
	return EventIgnored
}

func (s *textFieldState) change(ctx EventContext, value string) {
	w := s.Widget().(TextField)
	if w.OnChanged != nil {
		w.OnChanged(ctx, value)
	}
	s.SetState(func() {})
}

func (s *textFieldState) MouseShape(EventContext, Mouse) MouseShape { return MouseShapeTextInput }

func textFieldPadding(w TextField, theme TextFieldTheme) Insets {
	if w.Padding != (Insets{}) {
		return w.Padding
	}
	if theme.Padding == (Insets{}) {
		return DefaultTheme().TextField.Padding
	}
	return theme.Padding
}

func textFieldMinWidth(w TextField, theme TextFieldTheme) int {
	if w.MinWidth > 0 {
		return w.MinWidth
	}
	if theme.MinWidth <= 0 {
		return DefaultTheme().TextField.MinWidth
	}
	return theme.MinWidth
}

func charactersString(chars []Character) string {
	out := ""
	for _, ch := range chars {
		out += ch.Grapheme
	}
	return out
}
