package ui

type TextAlign int

const (
	// TextAlignStart aligns text to the start edge.
	TextAlignStart TextAlign = iota
	// TextAlignEnd aligns text to the end edge.
	TextAlignEnd
	// TextAlignLeft aligns text to the left edge.
	TextAlignLeft
	// TextAlignRight aligns text to the right edge.
	TextAlignRight
	// TextAlignCenter centers text.
	TextAlignCenter
)

// TextOverflow controls how text behaves when it exceeds its layout bounds.
type TextOverflow int

const (
	// TextOverflowClip clips overflowing text.
	TextOverflowClip TextOverflow = iota
	// TextOverflowEllipsis replaces clipped text with an ellipsis where possible.
	TextOverflowEllipsis
	// TextOverflowVisible paints text outside its layout bounds.
	TextOverflowVisible
)

// Text displays a single styled string.
//
// Text participates in ancestor SelectionArea widgets. Mouse selection copies
// the laid-out visible text; Ctrl+A from SelectionArea copies the full value,
// including text hidden by clipping or ellipsis.
type Text struct {
	// Value is the string to display.
	Value string
	// Style overrides Theme foreground when non-zero fields are set.
	Style Style
	// SoftWrap wraps text to the available width.
	SoftWrap bool
	// Overflow controls painting when text exceeds its layout bounds.
	Overflow TextOverflow
	// MaxLines limits the number of laid-out display lines when greater than zero.
	MaxLines int
	// Align controls horizontal placement within the laid-out width.
	Align TextAlign
	// OnPressed is called when the visible text is clicked or activated while focused.
	OnPressed VoidCallback
}

func (w Text) CreateRenderObject(ctx BuildContext) RenderObject {
	if w.Style == (Style{}) {
		w.Style = textStyle(MustDepend[Theme](ctx))
	}
	w.Style = textInteractiveStyle(w.Style, w.OnPressed)
	return &renderText{Text: w.Value, Style: w.Style, Options: w.options(), OnPressed: w.OnPressed, focusedIndex: elementFocusIndex}
}

func (w Text) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	if w.Style == (Style{}) {
		w.Style = textStyle(MustDepend[Theme](ctx))
	}
	w.Style = textInteractiveStyle(w.Style, w.OnPressed)
	r := ro.(*renderText)
	r.Text, r.Style, r.Options, r.OnPressed = w.Value, w.Style, w.options(), w.OnPressed
	r.MarkNeedsLayout()
}

func (w Text) options() TextLayoutOptions {
	return TextLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

func textInteractiveStyle(style Style, cb VoidCallback) Style {
	if cb != nil && style.UnderlineStyle == UnderlineOff {
		style.UnderlineStyle = UnderlineSingle
	}
	return style
}

// renderText lays out and paints a Text widget.
type renderText struct {
	LeafRenderObject
	Text           string
	Style          Style
	Options        TextLayoutOptions
	OnPressed      VoidCallback
	layout         TextLayout
	selection      TextSelection
	selectionStyle Style
	focusedIndex   int
}

func (r *renderText) Layout(ctx LayoutContext, c Constraints) {
	r.layout = LayoutText([]TextSpan{{Text: r.Text, Style: r.Style}}, c, r.Options)
	r.SetSize(r.layout.Size)
}

func (r *renderText) DryLayout(ctx LayoutContext, c Constraints) Size {
	return LayoutText([]TextSpan{{Text: r.Text, Style: r.Style}}, c, r.Options).Size
}

func (r *renderText) Paint(p *Painter, off Offset) {
	if r.Options.Overflow != TextOverflowVisible {
		p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
		defer p.PopClip()
	}
	paintTextBackground(p, off, r.Size(), []TextSpan{{Text: r.Text, Style: r.Style}})
	if !r.selection.IsCollapsed() {
		paintVisibleTextLayout(p, off, r.layout, textLayoutPaintOptions{
			Size:           r.Size(),
			Selection:      r.selection,
			SelectionStyle: r.selectionStyle,
		})
		return
	}
	if r.focusedIndex >= 0 {
		paintTextLayoutFocusedSpan(p, off, r.layout, 0)
		return
	}
	paintLaidOutText(p, off, r.layout, r.Options)
}

func (r *renderText) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	if r.OnPressed != nil && r.hasPaintedCellAt(Point{X: mouse.Col, Y: mouse.Row}) {
		return MouseShapeClickable
	}
	return MouseShapeDefault
}

func (r *renderText) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase || r.OnPressed == nil {
		return EventIgnored
	}
	mouse, ok := ev.(Mouse)
	if !ok {
		key, ok := ev.(Key)
		if !ok || keyIsRelease(key) || !key.MatchString("Enter") && !key.MatchString("Space") || r.focusedIndex < 0 {
			return EventIgnored
		}
		r.OnPressed(ctx)
		return EventHandled
	}
	if mouse.EventType != EventPress || mouse.Button != MouseLeftButton || !r.hasPaintedCellAt(Point{X: mouse.Col, Y: mouse.Row}) {
		return EventIgnored
	}
	r.OnPressed(ctx)
	return EventHandled
}

func (r *renderText) FocusableCount() int {
	if r.OnPressed == nil {
		return 0
	}
	return 1
}

func (r *renderText) DebugFocusTargets() []DebugFocusTarget {
	if r.OnPressed == nil {
		return nil
	}
	return []DebugFocusTarget{{Index: 0, Label: r.Text}}
}

func (r *renderText) SetFocusedIndex(index int) {
	if r.focusedIndex == index {
		return
	}
	r.focusedIndex = index
	r.MarkNeedsPaint()
}

func (r *renderText) FocusRect(index int) (Rect, bool) {
	if index < 0 || r.OnPressed == nil {
		return Rect{}, false
	}
	return textLayoutSpanRect(r.layout, 0)
}

func (r *renderText) hasPaintedCellAt(pt Point) bool {
	if pt.Y < 0 || pt.Y >= len(r.layout.Lines) {
		return false
	}
	line := r.layout.Lines[pt.Y]
	x := line.Offset
	for _, cell := range line.Cells {
		if pt.X >= x && pt.X < x+cell.Width {
			return true
		}
		x += cell.Width
	}
	return false
}

func (r *renderText) PositionForPoint(pt Point) (TextPosition, bool) {
	return textLayoutPositionForPoint(r.layout, pt)
}

func (r *renderText) StartPosition() TextPosition {
	return TextPosition{}
}

func (r *renderText) EndPosition() TextPosition {
	return textEndPositionForSpans([]TextSpan{{Text: r.Text}})
}

func (r *renderText) SelectAll() TextSelection {
	return textSelectionForSpans([]TextSpan{{Text: r.Text}})
}

func (r *renderText) SelectWordAt(pos TextPosition) TextSelection {
	return textWordSelectionForSpans([]TextSpan{{Text: r.Text}}, pos)
}

func (r *renderText) SelectLineAt(pos TextPosition) TextSelection {
	return textLineSelectionForSpans([]TextSpan{{Text: r.Text}}, pos)
}

func (r *renderText) SelectedText(selection TextSelection) string {
	return selectedTextForSpans([]TextSpan{{Text: r.Text}}, selection)
}

func (r *renderText) SetSelection(selection TextSelection, style Style) {
	if r.selection == selection && r.selectionStyle == style {
		return
	}
	r.selection = selection
	r.selectionStyle = style
	r.MarkNeedsPaint()
}
