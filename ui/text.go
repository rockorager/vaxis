package ui

type TextWidget struct {
	Value string
	Style Style
}

func Text(s string, opts ...TextOption) Widget {
	w := TextWidget{Value: s}
	for _, opt := range opts {
		opt(&w)
	}
	return w
}

type TextOption func(*TextWidget)

func TextStyle(style Style) TextOption { return func(w *TextWidget) { w.Style = style } }

func (w TextWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	return &RenderText{Text: w.Value, Style: w.Style}
}

func (w TextWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	if w.Style == (Style{}) {
		w.Style = MustDepend[Theme](ctx).Text
	}
	r := ro.(*RenderText)
	r.Text, r.Style = w.Value, w.Style
}

type RenderText struct {
	LeafRenderObject
	Text  string
	Style Style
}

func (r *RenderText) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(c.Constrain(ctx.MeasureText(r.Text, r.Style)))
}

func (r *RenderText) Paint(p *Painter, off Offset) { p.DrawText(off, r.Text, r.Style) }
