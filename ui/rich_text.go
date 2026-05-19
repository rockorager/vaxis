package ui

// RichText displays multiple styled spans as one text layout.
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
	// Style is merged over Theme.Text for this span.
	Style Style
}

func (w RichText) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderRichText{Spans: themedSpans(ctx, w.Spans), Options: w.options()}
}

func (w RichText) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*renderRichText)
	r.Spans, r.Options = themedSpans(ctx, w.Spans), w.options()
	r.MarkNeedsLayout()
}

func (w RichText) options() TextLayoutOptions {
	return TextLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

// renderRichText lays out and paints a RichText widget.
type renderRichText struct {
	LeafRenderObject
	Spans   []TextSpan
	Options TextLayoutOptions
	layout  TextLayout
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
	paintLaidOutText(p, off, r.layout, r.Options)
}

func themedSpans(ctx BuildContext, spans []TextSpan) []TextSpan {
	base := MustDepend[Theme](ctx).Text
	out := make([]TextSpan, len(spans))
	for i, span := range spans {
		out[i] = TextSpan{Text: span.Text, Style: mergeStyle(base, span.Style)}
	}
	return out
}
