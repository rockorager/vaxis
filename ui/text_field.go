package ui

type TextChangedCallback func(EventContext, string)

type TextField struct {
	Value       string
	Placeholder string
	OnChanged   TextChangedCallback
	OnSubmitted TextChangedCallback
	Padding     Insets
	MinWidth    int
	ObscureText bool
}

func (w TextField) CreateState() State {
	return &textFieldState{}
}

type textFieldState struct {
	StateBase
	node   FocusNode
	value  string
	cursor int
	scroll int
}

func (s *textFieldState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextField)
	s.value = w.Value
	s.node.onChange = s.MarkNeedsBuild
	chars := vaxisCharacters(s.value)
	cursor := min(s.cursor, len(chars))
	theme := MustDepend[Theme](ctx).TextField
	padding := textFieldPadding(w, theme)
	width := textFieldMinWidth(w, theme)
	contentWidth := max(1, width-padding.Left-padding.Right)
	displayValue := textFieldDisplayValue(s.value, w.ObscureText)
	s.scroll = textFieldScroll(s.scroll, cursor, len(chars), contentWidth)
	style := theme.Normal
	if s.node.HasFocus() {
		style = theme.Focused
	}
	content := s.content(w, displayValue, cursor, s.scroll, contentWidth, style, theme)
	if col, ok := textFieldCursorCell(cursor, s.scroll, len(chars), contentWidth); ok && s.node.HasFocus() {
		content = Cursor{Col: col, Shape: CursorBlock, Child: content}
	}
	return Focus(&s.node, SizedBox{Width: width, Height: 1 + padding.Top + padding.Bottom, Child: DecoratedBox(
		Decoration{Style: style},
		Padding(padding, content),
	)})
}

func (s *textFieldState) content(w TextField, displayValue string, cursor, scroll, width int, style Style, theme TextFieldTheme) Widget {
	if s.value == "" && !s.node.HasFocus() && w.Placeholder != "" {
		return Text{Value: w.Placeholder, Style: mergeStyle(style, theme.Placeholder), Overflow: TextOverflowClip}
	}
	chars := vaxisCharacters(displayValue)
	return RichText{Spans: textFieldSpans(chars, cursor, scroll, width, style), Overflow: TextOverflowClip}
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
	chars := vaxisCharacters(s.value)
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
	case key.MatchString("Enter"):
		if w.OnSubmitted != nil {
			w.OnSubmitted(ctx, s.value)
		}
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
	s.value = value
	if w.OnChanged != nil {
		w.OnChanged(ctx, value)
		return
	}
	s.SetState(func() {})
}

func (s *textFieldState) MouseShape(EventContext, Mouse) MouseShape {
	return MouseShapeTextInput
}

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

func textFieldDisplayValue(value string, obscure bool) string {
	if !obscure {
		return value
	}
	out := ""
	for range vaxisCharacters(value) {
		out += "•"
	}
	return out
}

func textFieldSpans(chars []Character, cursor, scroll, width int, style Style) []TextSpan {
	if width <= 0 {
		return nil
	}
	cells := make([]TextSpan, width)
	for i := range cells {
		cells[i] = TextSpan{Text: " ", Style: style}
	}
	leftOverflow := scroll > 0
	left := 0
	if leftOverflow {
		cells[0] = TextSpan{Text: "…", Style: style}
		left = 1
	}
	rightOverflow := len(chars)-scroll > width-left || scroll > 0 && cursor < len(chars) && len(chars)-scroll == width-left
	right := width
	if rightOverflow && right > left {
		right--
		cells[right] = TextSpan{Text: "…", Style: style}
	}
	for i, ch := range chars[scroll:min(len(chars), scroll+max(0, right-left))] {
		cells[left+i] = TextSpan{Text: ch.Grapheme, Style: style}
	}
	return coalesceTextSpans(cells)
}

func coalesceTextSpans(cells []TextSpan) []TextSpan {
	out := make([]TextSpan, 0, len(cells))
	for _, cell := range cells {
		if len(out) > 0 && out[len(out)-1].Style == cell.Style {
			out[len(out)-1].Text += cell.Text
			continue
		}
		out = append(out, cell)
	}
	return out
}

func textFieldScroll(scroll, cursor, length, width int) int {
	if width <= 0 {
		return cursor
	}
	if cursor < 0 {
		cursor = 0
	}
	if scroll < 0 {
		scroll = 0
	}
	if textFieldCursorVisible(scroll, cursor, length, width) {
		return scroll
	}
	if cursor < scroll {
		for next := cursor; next >= 0; next-- {
			if textFieldCursorVisible(next, cursor, length, width) {
				return next
			}
		}
		return 0
	}
	for next := 0; next <= cursor; next++ {
		if textFieldCursorVisible(next, cursor, length, width) {
			return next
		}
	}
	return scroll
}

func textFieldCursorVisible(scroll, cursor, length, width int) bool {
	if width == 1 && cursor >= length && scroll >= length {
		return true
	}
	left := 0
	if scroll > 0 {
		left = 1
	}
	right := width
	if (length-scroll > width-left || scroll > 0 && cursor < length && length-scroll == width-left) && right > left {
		right--
	}
	localCursor := left + cursor - scroll
	return localCursor >= left && localCursor < right
}

func textFieldCursorCell(cursor, scroll, length, width int) (int, bool) {
	if width <= 0 {
		return 0, false
	}
	left := 0
	if scroll > 0 {
		left = 1
	}
	right := width
	if (length-scroll > width-left || scroll > 0 && cursor < length && length-scroll == width-left) && right > left {
		right--
	}
	if left >= right && cursor >= length {
		return width - 1, true
	}
	localCursor := left + cursor - scroll
	return localCursor, localCursor >= left && localCursor < right
}
