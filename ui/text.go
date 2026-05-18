package ui

type TextAlign int

const (
	TextAlignStart TextAlign = iota
	TextAlignEnd
	TextAlignLeft
	TextAlignRight
	TextAlignCenter
)

type TextOverflow int

const (
	TextOverflowClip TextOverflow = iota
	TextOverflowEllipsis
	TextOverflowVisible
)

type Text struct {
	Value    string
	Style    Style
	SoftWrap bool
	Overflow TextOverflow
	MaxLines int
	Align    TextAlign
}

func (w Text) CreateRenderObject(ctx BuildContext) RenderObject {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	return &RenderText{Text: w.Value, Style: w.Style, Options: w.options()}
}

func (w Text) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	r := ro.(*RenderText)
	r.Text, r.Style, r.Options = w.Value, w.Style, w.options()
	r.MarkNeedsLayout()
}

func (w Text) options() textLayoutOptions {
	return textLayoutOptions{SoftWrap: w.SoftWrap, Overflow: w.Overflow, MaxLines: w.MaxLines, Align: w.Align}
}

type RenderText struct {
	LeafRenderObject
	Text    string
	Style   Style
	Options textLayoutOptions
	layout  laidOutText
}

func (r *RenderText) Layout(ctx LayoutContext, c Constraints) {
	r.layout = layoutText([]TextSpan{{Text: r.Text, Style: r.Style}}, c, r.Options)
	r.SetSize(r.layout.Size)
}

func (r *RenderText) Paint(p *Painter, off Offset) {
	if r.Options.Overflow != TextOverflowVisible {
		p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
		defer p.PopClip()
	}
	paintTextBackground(p, off, r.Size(), []TextSpan{{Text: r.Text, Style: r.Style}})
	paintLaidOutText(p, off, r.layout, r.Options)
}
