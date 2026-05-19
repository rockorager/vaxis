package ui

// SizedBox forces its child to a fixed cell size.
type SizedBox struct {
	// Width and Height are the fixed size requested for the child.
	Width, Height int
	// Child is laid out with tight constraints for Width and Height.
	Child Widget
}

func (w SizedBox) ChildWidget() Widget {
	return w.Child
}

func (w SizedBox) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderSizedBox{Width: w.Width, Height: w.Height}
}

func (w SizedBox) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*renderSizedBox)
	if r.Width != w.Width || r.Height != w.Height {
		r.Width, r.Height = w.Width, w.Height
		r.MarkNeedsLayout()
	}
}

// renderSizedBox lays out one child with a tight fixed size.
type renderSizedBox struct {
	SingleChildRenderObject
	Width, Height int
}

func (r *renderSizedBox) Layout(ctx LayoutContext, c Constraints) {
	size := c.Constrain(Size{Width: r.Width, Height: r.Height})
	if child := r.Child(); child != nil {
		child.Layout(ctx, Tight(size))
	}
	r.SetSize(size)
}

func (r *renderSizedBox) DryLayout(_ LayoutContext, c Constraints) Size {
	return c.Constrain(Size{Width: r.Width, Height: r.Height})
}

func (r *renderSizedBox) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		p.PushClip(Rect{X: off.X, Y: off.Y, Width: r.Size().Width, Height: r.Size().Height})
		defer p.PopClip()
		child.Paint(p, off)
	}
}

func (r *renderSizedBox) HitTest(*HitTestResult, Point) bool {
	return false
}
