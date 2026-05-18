package ui

type TextArea struct {
	Value       string
	Placeholder string
	OnChanged   TextChangedCallback
	Padding     Insets
	MinWidth    int
	MinHeight   int
	SoftWrap    bool
}

func (w TextArea) CreateState() State {
	return &textAreaState{}
}

type textAreaState struct {
	StateBase
	node      FocusNode
	buffer    TextBuffer
	layout    TextLayout
	scrollRow int
	scrollCol int
}

func (s *textAreaState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextArea)
	if s.buffer.Text() != w.Value {
		s.buffer.SetText(w.Value)
	}
	s.node.onChange = s.MarkNeedsBuild
	theme := MustDepend[Theme](ctx).TextField
	padding := textAreaPadding(w, theme)
	style := theme.Normal
	if s.node.HasFocus() {
		style = theme.Focused
	}
	return Focus(&s.node, DecoratedBox(
		Decoration{Style: style},
		Padding(padding, textAreaView{
			State:            s,
			Value:            s.buffer.Text(),
			Placeholder:      w.Placeholder,
			CursorOffset:     s.buffer.CursorOffset(),
			Focused:          s.node.HasFocus(),
			Style:            style,
			PlaceholderStyle: mergeStyle(style, theme.Placeholder),
			MinWidth:         textAreaMinWidth(w, theme) - padding.Left - padding.Right,
			MinHeight:        textAreaMinHeight(w) - padding.Top - padding.Bottom,
			SoftWrap:         w.SoftWrap,
		}),
	))
}

func (s *textAreaState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	changed := false
	handled := true
	switch {
	case key.Keycode == KeyLeft:
		handled = s.buffer.MoveLeft()
	case key.Keycode == KeyRight:
		handled = s.buffer.MoveRight()
	case key.Keycode == KeyUp:
		handled = s.moveUp()
	case key.Keycode == KeyDown:
		handled = s.moveDown()
	case key.Keycode == KeyHome:
		handled = s.buffer.MoveHome()
	case key.Keycode == KeyEnd:
		handled = s.buffer.MoveEnd()
	case key.Keycode == KeyBackspace:
		changed = s.buffer.DeleteBackward()
	case key.Keycode == KeyDelete:
		changed = s.buffer.DeleteForward()
	case key.MatchString("Enter"):
		changed = s.buffer.Insert("\n")
	case key.Text != "":
		changed = s.buffer.Insert(key.Text)
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

func (s *textAreaState) MouseShape(EventContext, Mouse) MouseShape {
	return MouseShapeTextInput
}

func (s *textAreaState) moveUp() bool {
	if len(s.layout.Lines) > 0 {
		return s.buffer.MoveVisualUp(s.layout)
	}
	return s.buffer.MoveLineUp()
}

func (s *textAreaState) moveDown() bool {
	if len(s.layout.Lines) > 0 {
		return s.buffer.MoveVisualDown(s.layout)
	}
	return s.buffer.MoveLineDown()
}

func (s *textAreaState) change(ctx EventContext) {
	w := s.Widget().(TextArea)
	if w.OnChanged != nil {
		w.OnChanged(ctx, s.buffer.Text())
		return
	}
	s.SetState(func() {})
}

func textAreaPadding(w TextArea, theme TextFieldTheme) Insets {
	if w.Padding != (Insets{}) {
		return w.Padding
	}
	if theme.Padding == (Insets{}) {
		return DefaultTheme().TextField.Padding
	}
	return theme.Padding
}

func textAreaMinWidth(w TextArea, theme TextFieldTheme) int {
	if w.MinWidth > 0 {
		return w.MinWidth
	}
	if theme.MinWidth <= 0 {
		return DefaultTheme().TextField.MinWidth
	}
	return theme.MinWidth
}

func textAreaMinHeight(w TextArea) int {
	if w.MinHeight > 0 {
		return w.MinHeight
	}
	return 3
}

type textAreaView struct {
	State            *textAreaState
	Value            string
	Placeholder      string
	CursorOffset     int
	Focused          bool
	Style            Style
	PlaceholderStyle Style
	MinWidth         int
	MinHeight        int
	SoftWrap         bool
}

func (w textAreaView) CreateRenderObject(BuildContext) RenderObject {
	return &renderTextArea{
		State:            w.State,
		Value:            w.Value,
		Placeholder:      w.Placeholder,
		CursorOffset:     w.CursorOffset,
		Focused:          w.Focused,
		Style:            w.Style,
		PlaceholderStyle: w.PlaceholderStyle,
		MinWidth:         max(1, w.MinWidth),
		MinHeight:        max(1, w.MinHeight),
		SoftWrap:         w.SoftWrap,
	}
}

func (w textAreaView) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderTextArea)
	if r.State != w.State || r.Value != w.Value || r.Placeholder != w.Placeholder || r.CursorOffset != w.CursorOffset ||
		r.Focused != w.Focused || r.Style != w.Style || r.PlaceholderStyle != w.PlaceholderStyle ||
		r.MinWidth != max(1, w.MinWidth) || r.MinHeight != max(1, w.MinHeight) || r.SoftWrap != w.SoftWrap {
		r.State = w.State
		r.Value = w.Value
		r.Placeholder = w.Placeholder
		r.CursorOffset = w.CursorOffset
		r.Focused = w.Focused
		r.Style = w.Style
		r.PlaceholderStyle = w.PlaceholderStyle
		r.MinWidth = max(1, w.MinWidth)
		r.MinHeight = max(1, w.MinHeight)
		r.SoftWrap = w.SoftWrap
		r.MarkNeedsLayout()
	}
}

type renderTextArea struct {
	LeafRenderObject
	State            *textAreaState
	Value            string
	Placeholder      string
	CursorOffset     int
	Focused          bool
	Style            Style
	PlaceholderStyle Style
	MinWidth         int
	MinHeight        int
	SoftWrap         bool
	layout           TextLayout
}

func (r *renderTextArea) Layout(ctx LayoutContext, c Constraints) {
	r.layout = r.textLayout(c)
	size := r.sizeForLayout(c, r.layout)
	r.SetSize(size)
	r.keepCursorVisible(size)
	if r.State != nil {
		r.State.layout = r.layout
	}
}

func (r *renderTextArea) DryLayout(_ LayoutContext, c Constraints) Size {
	layout := r.textLayout(c)
	return r.sizeForLayout(c, layout)
}

func (r *renderTextArea) Paint(p *Painter, off Offset) {
	size := r.Size()
	p.PushClip(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height})
	defer p.PopClip()
	for row := r.scrollRow(); row < len(r.layout.Lines) && row < r.scrollRow()+size.Height; row++ {
		line := r.layout.Lines[row]
		y := off.Y + row - r.scrollRow()
		x := off.X + line.Offset - r.scrollCol()
		for _, run := range line.Runs {
			p.DrawText(Offset{X: x, Y: y}, run.Text, run.Style)
			x += textWidth(run.Text)
		}
	}
	if r.Focused && r.Value != "" {
		if row, col, ok := r.cursorCell(size.Width); ok {
			p.ShowCursor(off.X+col-r.scrollCol(), off.Y+row-r.scrollRow(), CursorBlock)
		}
	} else if r.Focused && r.Value == "" {
		p.ShowCursor(off.X, off.Y, CursorBlock)
	}
}

func (r *renderTextArea) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderTextArea) textLayout(c Constraints) TextLayout {
	text := r.Value
	style := r.Style
	if text == "" && !r.Focused && r.Placeholder != "" {
		text = r.Placeholder
		style = r.PlaceholderStyle
	}
	maxWidth := Unbounded
	if r.SoftWrap && c.HasBoundedWidth() {
		maxWidth = max(1, c.MaxWidth)
	}
	return LayoutText([]TextSpan{{Text: text, Style: style}}, Constraints{MaxWidth: maxWidth, MaxHeight: Unbounded}, TextLayoutOptions{SoftWrap: r.SoftWrap})
}

func (r *renderTextArea) sizeForLayout(c Constraints, layout TextLayout) Size {
	width := max(r.MinWidth, layout.Size.Width)
	if r.SoftWrap && c.HasBoundedWidth() {
		width = max(r.MinWidth, c.MaxWidth)
	}
	height := max(r.MinHeight, layout.Size.Height)
	return c.Constrain(Size{Width: width, Height: height})
}

func (r *renderTextArea) keepCursorVisible(size Size) {
	if r.State == nil || size.Width <= 0 || size.Height <= 0 {
		return
	}
	row, col, ok := r.cursorCell(size.Width)
	if !ok {
		row, col = 0, 0
	}
	if row < r.State.scrollRow {
		r.State.scrollRow = row
	}
	if row >= r.State.scrollRow+size.Height {
		r.State.scrollRow = row - size.Height + 1
	}
	if r.SoftWrap {
		r.State.scrollCol = 0
		return
	}
	if col < r.State.scrollCol {
		r.State.scrollCol = col
	}
	if col >= r.State.scrollCol+size.Width {
		r.State.scrollCol = col - size.Width + 1
	}
}

func (r *renderTextArea) cursorCell(width int) (row, col int, ok bool) {
	buffer := NewTextBuffer(r.Value)
	buffer.SetCursorOffset(r.CursorOffset)
	row, col, ok = buffer.CursorCell(r.layout)
	if ok && r.SoftWrap && width > 0 && col >= width {
		return row + 1, 0, true
	}
	return row, col, ok
}

func (r *renderTextArea) scrollRow() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollRow
}

func (r *renderTextArea) scrollCol() int {
	if r.State == nil {
		return 0
	}
	return r.State.scrollCol
}
