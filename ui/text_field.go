package ui

// TextChangedCallback receives a text editing value change.
type TextChangedCallback func(EventContext, string)

// TextField is a controlled single-line text input.
type TextField struct {
	// Value is the current text. The widget does not mutate this field directly.
	Value string
	// Placeholder is shown when Value is empty and the field is not focused.
	Placeholder string
	// OnChanged is called with the next value after an edit.
	OnChanged TextChangedCallback
	// OnSubmitted is called with the current value when Enter is pressed.
	OnSubmitted TextChangedCallback
	// Padding overrides the default text field padding when non-zero.
	Padding Insets
	// MinWidth overrides the default text field minimum width when greater than zero.
	MinWidth int
	// ObscureText hides the displayed value, useful for password-style fields.
	ObscureText bool
}

func (w TextField) CreateState() State {
	return &textFieldState{}
}

type textFieldState struct {
	StateBase
	editor textEditorState
	scroll int
}

func (s *textFieldState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextField)
	s.editor.SyncValue(w.Value)
	s.editor.SetFocusChange(s.MarkNeedsBuild)
	chars := vaxisCharacters(s.editor.Text())
	cursor := s.editor.CursorOffset()
	theme := textFieldTheme(MustDepend[Theme](ctx))
	padding := textFieldPadding(w, theme)
	width := textFieldMinWidth(w, theme)
	contentWidth := max(1, width-padding.Left-padding.Right)
	displayValue := textFieldDisplayValue(s.editor.Text(), w.ObscureText)
	s.scroll = textFieldScroll(s.scroll, cursor, len(chars), contentWidth)
	style := theme.Normal
	if s.editor.HasFocus() {
		style = theme.Focused
	}
	content := s.content(w, displayValue, cursor, s.scroll, contentWidth, style, theme)
	content = SelectionContainer{Disabled: true, Child: content}
	if col, ok := textFieldCursorCell(cursor, s.scroll, len(chars), contentWidth); ok && s.editor.HasFocus() {
		content = Cursor{Col: col, Shape: CursorBlock, Child: content}
	}
	child := s.editor.Focus(SizedBox{Width: width, Height: 1 + padding.Top + padding.Bottom, Child: DecoratedBox(
		Decoration{Style: style},
		Padding(padding, content),
	)})
	return s.editor.DefaultActions(s.handleOptions(w), child)
}

func (s *textFieldState) content(w TextField, displayValue string, cursor, scroll, width int, style Style, theme TextFieldTheme) Widget {
	if s.editor.Text() == "" && !s.editor.HasFocus() && w.Placeholder != "" {
		return Text{Value: w.Placeholder, Style: mergeStyle(style, theme.Placeholder), Overflow: TextOverflowClip}
	}
	chars := vaxisCharacters(displayValue)
	selection := TextSelection{}
	if !w.ObscureText {
		selection = s.editor.Selection()
	}
	return RichText{Spans: textFieldSpans(chars, selection, cursor, scroll, width, style, mergeStyle(style, theme.Selection)), Overflow: TextOverflowClip}
}

func (s *textFieldState) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(TextField)
	return s.editor.HandleEvent(ctx, ev, s.handleOptions(w))
}

func (s *textFieldState) handleOptions(w TextField) textEditorHandleOptions {
	return textEditorHandleOptions{
		insertMode:       textEditorSingleLine,
		markNeedsBuild:   s.MarkNeedsBuild,
		onChanged:        w.OnChanged,
		submit:           s.submit,
		positionForMouse: s.positionForMouse,
	}
}

func (s *textFieldState) submit(ctx EventContext, value string) {
	w := s.Widget().(TextField)
	if w.OnSubmitted != nil {
		w.OnSubmitted(ctx, value)
	}
}

func (s *textFieldState) positionForMouse(mouse Mouse) (TextPosition, bool) {
	w := s.Widget().(TextField)
	theme := textFieldTheme(MustDepend[Theme](s.Context()))
	padding := textFieldPadding(w, theme)
	col := mouse.Col - padding.Left
	if s.scroll > 0 {
		col--
	}
	offset := clampInt(s.scroll+col, 0, s.editor.Len())
	return s.editor.PositionForOffset(offset), true
}

func (s *textFieldState) MouseShape(EventContext, Mouse) MouseShape {
	return MouseShapeTextInput
}

func textFieldPadding(w TextField, theme TextFieldTheme) Insets {
	if w.Padding != (Insets{}) {
		return w.Padding
	}
	if theme.Padding == (Insets{}) {
		return Symmetric(1, 0)
	}
	return theme.Padding
}

func textFieldMinWidth(w TextField, theme TextFieldTheme) int {
	if w.MinWidth > 0 {
		return w.MinWidth
	}
	if theme.MinWidth <= 0 {
		return defaultTextFieldMinWidth
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

func textFieldSpans(chars []Character, selection TextSelection, cursor, scroll, width int, style Style, selectionStyle Style) []TextSpan {
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
		cellStyle := style
		pos := TextPosition{ByteOffset: textFieldByteOffset(chars, scroll+i)}
		if selection.IntersectsCell(TextCell{Text: ch.Grapheme, Position: pos}) {
			cellStyle = selectionStyle
		}
		cells[left+i] = TextSpan{Text: ch.Grapheme, Style: cellStyle}
	}
	return coalesceTextSpans(cells)
}

func textFieldByteOffset(chars []Character, offset int) int {
	byteOffset := 0
	for _, ch := range chars[:clampInt(offset, 0, len(chars))] {
		byteOffset += len(ch.Grapheme)
	}
	return byteOffset
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
	if length <= width {
		return 0
	}
	scroll = min(scroll, length)
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
