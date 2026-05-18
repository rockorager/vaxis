package ui

type SizedBox struct {
	Width, Height int
	Child         Widget
}

func (w SizedBox) ChildWidget() Widget {
	return w.Child
}

func (w SizedBox) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderSizedBox{Width: w.Width, Height: w.Height}
}

func (w SizedBox) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderSizedBox)
	if r.Width != w.Width || r.Height != w.Height {
		r.Width, r.Height = w.Width, w.Height
		r.MarkNeedsLayout()
	}
}

type RenderSizedBox struct {
	SingleChildRenderObject
	Width, Height int
}

func (r *RenderSizedBox) Layout(ctx LayoutContext, c Constraints) {
	size := c.Constrain(Size{Width: r.Width, Height: r.Height})
	if child := r.Child(); child != nil {
		child.Layout(ctx, Tight(size))
	}
	r.SetSize(size)
}

func (r *RenderSizedBox) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
		defer p.PopClip()
		child.Paint(p, off)
	}
}

func (r *RenderSizedBox) HitTest(*HitTestResult, Point) bool {
	return false
}
