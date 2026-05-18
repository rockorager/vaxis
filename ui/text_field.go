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
	node      FocusNode
	buffer    TextBuffer
	scroll    int
	selecting bool
}

func (s *textFieldState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextField)
	if s.buffer.Text() != w.Value {
		s.buffer.SetText(w.Value)
	}
	s.node.onChange = s.MarkNeedsBuild
	chars := vaxisCharacters(s.buffer.Text())
	cursor := s.buffer.CursorOffset()
	theme := MustDepend[Theme](ctx).TextField
	padding := textFieldPadding(w, theme)
	width := textFieldMinWidth(w, theme)
	contentWidth := max(1, width-padding.Left-padding.Right)
	displayValue := textFieldDisplayValue(s.buffer.Text(), w.ObscureText)
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
	if s.buffer.Text() == "" && !s.node.HasFocus() && w.Placeholder != "" {
		return Text{Value: w.Placeholder, Style: mergeStyle(style, theme.Placeholder), Overflow: TextOverflowClip}
	}
	chars := vaxisCharacters(displayValue)
	selection := TextSelection{}
	if !w.ObscureText {
		selection = s.buffer.Selection()
	}
	return RichText{Spans: textFieldSpans(chars, selection, cursor, scroll, width, style, mergeStyle(style, theme.Selection)), Overflow: TextOverflowClip}
}

func (s *textFieldState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	key, ok := ev.(Key)
	if !ok {
		mouse, ok := ev.(Mouse)
		if !ok {
			return EventIgnored
		}
		return s.handleMouse(mouse)
	}
	if keyIsRelease(key) {
		return EventIgnored
	}
	w := s.Widget().(TextField)
	changed := false
	handled := true
	switch {
	case key.MatchString("Ctrl+a"):
		handled = s.buffer.SelectAll()
	case key.MatchString("Ctrl+c"):
		if s.buffer.HasSelection() {
			ctx.Copy(s.buffer.SelectedText())
		}
	case key.MatchString("Ctrl+Shift+Left"):
		handled = s.buffer.ExtendWordLeft()
	case key.MatchString("Ctrl+Shift+Right"):
		handled = s.buffer.ExtendWordRight()
	case key.MatchString("Ctrl+Left"):
		handled = s.buffer.MoveWordLeft()
	case key.MatchString("Ctrl+Right"):
		handled = s.buffer.MoveWordRight()
	case key.MatchString("Shift+Left"):
		handled = s.buffer.ExtendLeft()
	case key.MatchString("Shift+Right"):
		handled = s.buffer.ExtendRight()
	case key.MatchString("Shift+Home"):
		handled = s.buffer.ExtendHome()
	case key.MatchString("Shift+End"):
		handled = s.buffer.ExtendEnd()
	case key.Keycode == KeyLeft:
		handled = s.buffer.MoveLeft()
	case key.Keycode == KeyRight:
		handled = s.buffer.MoveRight()
	case key.Keycode == KeyHome:
		handled = s.buffer.MoveHome()
	case key.Keycode == KeyEnd:
		handled = s.buffer.MoveEnd()
	case key.MatchString("Enter"):
		if w.OnSubmitted != nil {
			w.OnSubmitted(ctx, s.buffer.Text())
		}
		return EventHandled
	case key.Keycode == KeyBackspace:
		if key.MatchString("Ctrl+Backspace") {
			changed = s.buffer.DeleteWordBackward()
		} else {
			changed = s.buffer.DeleteBackward()
		}
	case key.Keycode == KeyDelete:
		if key.MatchString("Ctrl+Delete") {
			changed = s.buffer.DeleteWordForward()
		} else {
			changed = s.buffer.DeleteForward()
		}
	case key.Text != "":
		changed = s.buffer.InsertSingleLine(key.Text)
	default:
		return EventIgnored
	}
	if changed {
		s.change(ctx)
		return EventHandled
	}
	if handled {
		s.MarkNeedsBuild()
		return EventHandled
	}
	return EventHandled
}

func (s *textFieldState) change(ctx EventContext) {
	w := s.Widget().(TextField)
	value := s.buffer.Text()
	if w.OnChanged != nil {
		w.OnChanged(ctx, value)
		return
	}
	s.SetState(func() {})
}

func (s *textFieldState) handleMouse(mouse Mouse) EventResult {
	if mouse.Button != MouseLeftButton {
		if mouse.EventType == EventRelease {
			s.selecting = false
			return EventHandled
		}
		return EventIgnored
	}
	pos := s.positionForMouse(mouse)
	switch mouse.EventType {
	case EventPress:
		s.node.RequestFocus()
		s.selecting = true
		s.buffer.CollapseSelection(pos)
		s.MarkNeedsBuild()
		return EventHandled
	case EventMotion:
		if !s.selecting {
			return EventIgnored
		}
		s.buffer.ExtendSelection(pos)
		s.MarkNeedsBuild()
		return EventHandled
	case EventRelease:
		if !s.selecting {
			return EventIgnored
		}
		s.selecting = false
		s.buffer.ExtendSelection(pos)
		s.MarkNeedsBuild()
		return EventHandled
	default:
		return EventIgnored
	}
}

func (s *textFieldState) positionForMouse(mouse Mouse) TextPosition {
	w := s.Widget().(TextField)
	theme := MustDepend[Theme](s.Context()).TextField
	padding := textFieldPadding(w, theme)
	col := mouse.Col - padding.Left
	if s.scroll > 0 {
		col--
	}
	offset := clampInt(s.scroll+col, 0, s.buffer.Len())
	return s.buffer.positionForOffset(offset)
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
		if textCellSelected(TextCell{Text: ch.Grapheme, Position: pos}, selection) {
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
