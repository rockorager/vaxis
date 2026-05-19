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
	editor    textEditorState
	layout    TextLayout
	scrollRow int
	scrollCol int
}

func (s *textAreaState) Build(ctx BuildContext) Widget {
	w := s.Widget().(TextArea)
	s.editor.SyncValue(w.Value)
	s.editor.SetFocusChange(s.MarkNeedsBuild)
	theme := MustDepend[Theme](ctx).TextField
	padding := textAreaPadding(w, theme)
	style := theme.Normal
	if s.editor.HasFocus() {
		style = theme.Focused
	}
	return s.editor.Focus(DecoratedBox(
		Decoration{Style: style},
		Padding(padding, textAreaView{
			State:            s,
			Value:            s.editor.Text(),
			Placeholder:      w.Placeholder,
			CursorOffset:     s.editor.CursorOffset(),
			Selection:        s.editor.Selection(),
			Focused:          s.editor.HasFocus(),
			Style:            style,
			PlaceholderStyle: mergeStyle(style, theme.Placeholder),
			SelectionStyle:   mergeStyle(style, theme.Selection),
			MinWidth:         textAreaMinWidth(w, theme) - padding.Left - padding.Right,
			MinHeight:        textAreaMinHeight(w) - padding.Top - padding.Bottom,
			SoftWrap:         w.SoftWrap,
		}),
	))
}

func (s *textAreaState) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(TextArea)
	return s.editor.HandleEvent(ctx, ev, textEditorHandleOptions{
		insertMode:       textEditorMultiline,
		markNeedsBuild:   s.MarkNeedsBuild,
		onChanged:        w.OnChanged,
		positionForMouse: s.positionForMouse,
		moveUp:           s.moveUp,
		moveDown:         s.moveDown,
		extendUp:         s.extendUp,
		extendDown:       s.extendDown,
	})
}

func (s *textAreaState) MouseShape(EventContext, Mouse) MouseShape {
	return MouseShapeTextInput
}

func (s *textAreaState) positionForMouse(mouse Mouse) (TextPosition, bool) {
	w := s.Widget().(TextArea)
	theme := MustDepend[Theme](s.Context()).TextField
	padding := textAreaPadding(w, theme)
	if len(s.layout.Lines) == 0 {
		return TextPosition{}, true
	}
	row := mouse.Row - padding.Top + s.scrollRow
	col := mouse.Col - padding.Left + s.scrollCol
	if row < 0 {
		row = 0
	}
	if row >= len(s.layout.Lines) {
		return s.layout.Lines[len(s.layout.Lines)-1].End, true
	}
	pos, ok := s.layout.PositionForCell(row, col)
	if !ok {
		return TextPosition{}, false
	}
	return pos, true
}

func (s *textAreaState) moveUp() bool {
	return s.editor.MoveVisualUp(s.layout)
}

func (s *textAreaState) moveDown() bool {
	return s.editor.MoveVisualDown(s.layout)
}

func (s *textAreaState) extendUp() bool {
	return s.editor.ExtendVisualUp(s.layout)
}

func (s *textAreaState) extendDown() bool {
	return s.editor.ExtendVisualDown(s.layout)
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
	Selection        TextSelection
	Focused          bool
	Style            Style
	PlaceholderStyle Style
	SelectionStyle   Style
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
		Selection:        w.Selection,
		Focused:          w.Focused,
		Style:            w.Style,
		PlaceholderStyle: w.PlaceholderStyle,
		SelectionStyle:   w.SelectionStyle,
		MinWidth:         max(1, w.MinWidth),
		MinHeight:        max(1, w.MinHeight),
		SoftWrap:         w.SoftWrap,
	}
}

func (w textAreaView) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderTextArea)
	if r.State != w.State || r.Value != w.Value || r.Placeholder != w.Placeholder || r.CursorOffset != w.CursorOffset ||
		r.Selection != w.Selection || r.Focused != w.Focused || r.Style != w.Style || r.PlaceholderStyle != w.PlaceholderStyle ||
		r.SelectionStyle != w.SelectionStyle ||
		r.MinWidth != max(1, w.MinWidth) || r.MinHeight != max(1, w.MinHeight) || r.SoftWrap != w.SoftWrap {
		r.State = w.State
		r.Value = w.Value
		r.Placeholder = w.Placeholder
		r.CursorOffset = w.CursorOffset
		r.Selection = w.Selection
		r.Focused = w.Focused
		r.Style = w.Style
		r.PlaceholderStyle = w.PlaceholderStyle
		r.SelectionStyle = w.SelectionStyle
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
	Selection        TextSelection
	Focused          bool
	Style            Style
	PlaceholderStyle Style
	SelectionStyle   Style
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
	selection := TextSelection{}
	selectionStyle := Style{}
	if r.Focused {
		selection = r.Selection
		selectionStyle = r.SelectionStyle
	}
	paintVisibleTextLayout(p, off, r.layout, textLayoutPaintOptions{
		Size:           size,
		ScrollRow:      r.scrollRow(),
		ScrollCol:      r.scrollCol(),
		Selection:      selection,
		SelectionStyle: selectionStyle,
	})
	if r.Focused && r.Value != "" {
		if row, col, ok := r.layout.CursorCell(r.cursorPosition(), TextCursorCellOptions{SoftWrap: r.SoftWrap, Width: size.Width}); ok {
			p.ShowCursor(off.X+col-r.scrollCol(), off.Y+row-r.scrollRow(), CursorBlock)
		}
	} else if r.Focused && r.Value == "" {
		p.ShowCursor(off.X, off.Y, CursorBlock)
	}
}

func (r *renderTextArea) HitTest(*HitTestResult, Point) bool {
	return true
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
	row, col, ok := r.layout.CursorCell(r.cursorPosition(), TextCursorCellOptions{SoftWrap: r.SoftWrap, Width: size.Width})
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

func (r *renderTextArea) cursorPosition() TextPosition {
	buffer := NewTextBuffer(r.Value)
	buffer.SetCursorOffset(r.CursorOffset)
	return buffer.Position()
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
