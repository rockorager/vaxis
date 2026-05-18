package ui

type RichText struct {
	Spans    []TextSpan
	SoftWrap bool
	Overflow TextOverflow
	MaxLines int
	Align    TextAlign
}

type TextSpan struct {
	Text  string
	Style Style
}

func (w RichText) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderRichText{Spans: themedSpans(ctx, w.Spans), Options: w.options()}
}

func (w RichText) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderRichText)
	r.Spans, r.Options = themedSpans(ctx, w.Spans), w.options()
	r.MarkNeedsLayout()
}

func (w RichText) options() textLayoutOptions {
	return textLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

type RenderRichText struct {
	LeafRenderObject
	Spans   []TextSpan
	Options textLayoutOptions
	layout  laidOutText
}

func (r *RenderRichText) Layout(ctx LayoutContext, c Constraints) {
	r.layout = layoutText(r.Spans, c, r.Options)
	r.SetSize(r.layout.Size)
}

func (r *RenderRichText) Paint(p *Painter, off Offset) {
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
