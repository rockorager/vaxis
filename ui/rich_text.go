package ui

// RichText displays multiple styled spans as one text layout.
//
// RichText participates in ancestor SelectionArea widgets as one selectable
// text run. Selection and copy preserve the rendered span order, but not style.
type RichText struct {
	// Spans are the styled text runs to display.
	Spans []TextSpan
	// SoftWrap wraps text to the available width.
	SoftWrap bool
	// Overflow controls painting when text exceeds its layout bounds.
	Overflow TextOverflow
	// MaxLines limits the number of laid-out display lines when greater than zero.
	MaxLines int
	// Align controls horizontal placement within the laid-out width.
	Align TextAlign
}

// TextSpan is a styled run of text.
type TextSpan struct {
	// Text is the span contents.
	Text string
	// Style is merged over Theme.Text for this span. Set Style.Hyperlink for OSC
	// 8 terminal links.
	Style Style
	// OnPressed is called when the span is clicked.
	OnPressed VoidCallback
}

func (w RichText) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderRichText{Spans: themedSpans(ctx, w.Spans), RawSpans: w.Spans, Options: w.options(), focusedIndex: elementFocusIndex}
}

func (w RichText) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*renderRichText)
	r.Spans, r.RawSpans, r.Options = themedSpans(ctx, w.Spans), w.Spans, w.options()
	r.MarkNeedsLayout()
}

func (w RichText) options() TextLayoutOptions {
	return TextLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

// renderRichText lays out and paints a RichText widget.
type renderRichText struct {
	LeafRenderObject
	Spans          []TextSpan
	RawSpans       []TextSpan
	Options        TextLayoutOptions
	layout         TextLayout
	selection      TextSelection
	selectionStyle Style
	focusedIndex   int
}

func (r *renderRichText) Layout(ctx LayoutContext, c Constraints) {
	r.layout = LayoutText(r.Spans, c, r.Options)
	r.SetSize(r.layout.Size)
}

func (r *renderRichText) DryLayout(ctx LayoutContext, c Constraints) Size {
	return LayoutText(r.Spans, c, r.Options).Size
}

func (r *renderRichText) Paint(p *Painter, off Offset) {
	if r.Options.Overflow != TextOverflowVisible {
		p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
		defer p.PopClip()
	}
	if !r.selection.IsCollapsed() {
		paintVisibleTextLayout(p, off, r.layout, textLayoutPaintOptions{
			Size:           r.Size(),
			Selection:      r.selection,
			SelectionStyle: r.selectionStyle,
		})
		return
	}
	if r.focusedIndex >= 0 {
		paintTextLayoutFocusedSpan(p, off, r.layout, r.focusedSpanIndex())
		return
	}
	paintLaidOutText(p, off, r.layout, r.Options)
}

func (r *renderRichText) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	if span, ok := r.spanAtPoint(Point{X: mouse.Col, Y: mouse.Row}); ok && span.OnPressed != nil {
		return MouseShapeClickable
	}
	return MouseShapeDefault
}

func (r *renderRichText) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	mouse, ok := ev.(Mouse)
	if !ok {
		key, ok := ev.(Key)
		if !ok || keyIsRelease(key) || !key.MatchString("Enter") && !key.MatchString("Space") {
			return EventIgnored
		}
		span, ok := r.focusedSpan()
		if !ok || span.OnPressed == nil {
			return EventIgnored
		}
		span.OnPressed(ctx)
		return EventHandled
	}
	if mouse.EventType != EventPress || mouse.Button != MouseLeftButton {
		return EventIgnored
	}
	span, ok := r.spanAtPoint(Point{X: mouse.Col, Y: mouse.Row})
	if !ok || span.OnPressed == nil {
		return EventIgnored
	}
	span.OnPressed(ctx)
	return EventHandled
}

func (r *renderRichText) FocusableCount() int {
	count := 0
	for _, span := range r.RawSpans {
		if span.OnPressed != nil {
			count++
		}
	}
	return count
}

func (r *renderRichText) DebugFocusTargets() []DebugFocusTarget {
	var out []DebugFocusTarget
	for _, span := range r.RawSpans {
		if span.OnPressed == nil {
			continue
		}
		out = append(out, DebugFocusTarget{Index: len(out), Label: span.Text})
	}
	return out
}

func (r *renderRichText) SetFocusedIndex(index int) {
	if r.focusedIndex == index {
		return
	}
	r.focusedIndex = index
	r.MarkNeedsPaint()
}

func (r *renderRichText) focusedSpan() (TextSpan, bool) {
	idx := r.focusedSpanIndex()
	if idx < 0 || idx >= len(r.RawSpans) {
		return TextSpan{}, false
	}
	return r.RawSpans[idx], true
}

func (r *renderRichText) focusedSpanIndex() int {
	if r.focusedIndex < 0 {
		return -1
	}
	focusable := 0
	for i, span := range r.RawSpans {
		if span.OnPressed == nil {
			continue
		}
		if focusable == r.focusedIndex {
			return i
		}
		focusable++
	}
	return -1
}

func (r *renderRichText) spanAtPoint(pt Point) (TextSpan, bool) {
	pos, ok := r.positionAtPaintedCell(pt)
	if !ok || pos.Span < 0 || pos.Span >= len(r.RawSpans) {
		return TextSpan{}, false
	}
	return r.RawSpans[pos.Span], true
}

func paintTextLayoutFocusedSpan(p *Painter, off Offset, layout TextLayout, spanIndex int) {
	for y, line := range layout.Lines {
		x := off.X + line.Offset
		for _, cell := range line.Cells {
			style := cell.Style
			if cell.Position.Span == spanIndex {
				style.UnderlineStyle = UnderlineDouble
			}
			p.DrawText(Offset{X: x, Y: off.Y + y}, cell.Text, style)
			x += cell.Width
		}
	}
}

func (r *renderRichText) positionAtPaintedCell(pt Point) (TextPosition, bool) {
	if pt.Y < 0 || pt.Y >= len(r.layout.Lines) {
		return TextPosition{}, false
	}
	line := r.layout.Lines[pt.Y]
	x := line.Offset
	for _, cell := range line.Cells {
		if pt.X >= x && pt.X < x+cell.Width {
			return cell.Position, true
		}
		x += cell.Width
	}
	return TextPosition{}, false
}

func (r *renderRichText) PositionForPoint(pt Point) (TextPosition, bool) {
	return textLayoutPositionForPoint(r.layout, pt)
}

func (r *renderRichText) StartPosition() TextPosition {
	return TextPosition{}
}

func (r *renderRichText) EndPosition() TextPosition {
	return textEndPositionForSpans(r.Spans)
}

func (r *renderRichText) SelectAll() TextSelection {
	return textSelectionForSpans(r.Spans)
}

func (r *renderRichText) SelectWordAt(pos TextPosition) TextSelection {
	return textWordSelectionForSpans(r.Spans, pos)
}

func (r *renderRichText) SelectLineAt(pos TextPosition) TextSelection {
	return textLineSelectionForSpans(r.Spans, pos)
}

func (r *renderRichText) SelectedText(selection TextSelection) string {
	return selectedTextForSpans(r.Spans, selection)
}

func (r *renderRichText) SetSelection(selection TextSelection, style Style) {
	if r.selection == selection && r.selectionStyle == style {
		return
	}
	r.selection = selection
	r.selectionStyle = style
	r.MarkNeedsPaint()
}

func themedSpans(ctx BuildContext, spans []TextSpan) []TextSpan {
	base := MustDepend[Theme](ctx).Text
	out := make([]TextSpan, len(spans))
	for i, span := range spans {
		style := mergeStyle(base, span.Style)
		if span.OnPressed != nil && style.UnderlineStyle == UnderlineOff {
			style.UnderlineStyle = UnderlineSingle
		}
		out[i] = TextSpan{Text: span.Text, Style: style, OnPressed: span.OnPressed}
	}
	return out
}
