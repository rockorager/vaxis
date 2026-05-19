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
type Text struct {
	// Value is the string to display.
	Value string
	// Style overrides Theme.Text when non-zero fields are set.
	Style Style
	// SoftWrap wraps text to the available width.
	SoftWrap bool
	// Overflow controls painting when text exceeds its layout bounds.
	Overflow TextOverflow
	// MaxLines limits the number of laid-out display lines when greater than zero.
	MaxLines int
	// Align controls horizontal placement within the laid-out width.
	Align TextAlign
}

func (w Text) CreateRenderObject(ctx BuildContext) RenderObject {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	return &renderText{Text: w.Value, Style: w.Style, Options: w.options()}
}

func (w Text) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	r := ro.(*renderText)
	r.Text, r.Style, r.Options = w.Value, w.Style, w.options()
	r.MarkNeedsLayout()
}

func (w Text) options() TextLayoutOptions {
	return TextLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

// renderText lays out and paints a Text widget.
type renderText struct {
	LeafRenderObject
	Text           string
	Style          Style
	Options        TextLayoutOptions
	layout         TextLayout
	selection      TextSelection
	selectionStyle Style
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
	paintLaidOutText(p, off, r.layout, r.Options)
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
