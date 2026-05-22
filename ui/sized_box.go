package ui

// SizedBox forces its child to a fixed cell size on specified axes.
type SizedBox struct {
	// Width and Height are the fixed size requested for the child. Values less
	// than or equal to zero leave that axis unconstrained by the SizedBox.
	Width, Height int
	// Child is laid out with tight constraints for specified axes.
	Child Widget
}

func (w SizedBox) WidgetChild() Widget {
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

// renderSizedBox lays out one child with tight constraints on specified axes.
type renderSizedBox struct {
	SingleChildRenderObject
	Width, Height int
}

func (r *renderSizedBox) Layout(ctx LayoutContext, c Constraints) {
	if child := r.Child(); child != nil {
		child.Layout(ctx, r.childConstraints(c))
		r.SetSize(c.Constrain(child.Base().Size()))
		return
	}
	r.SetSize(c.Constrain(r.emptySize()))
}

func (r *renderSizedBox) DryLayout(ctx LayoutContext, c Constraints) Size {
	if child := r.Child(); child != nil {
		return c.Constrain(DryLayout(ctx, child, r.childConstraints(c)))
	}
	return c.Constrain(r.emptySize())
}

func (r *renderSizedBox) childConstraints(c Constraints) Constraints {
	if r.Width > 0 {
		width := clamp(r.Width, c.MinWidth, c.MaxWidth)
		c.MinWidth = width
		c.MaxWidth = width
	}
	if r.Height > 0 {
		height := clamp(r.Height, c.MinHeight, c.MaxHeight)
		c.MinHeight = height
		c.MaxHeight = height
	}
	return c
}

func (r *renderSizedBox) emptySize() Size {
	return Size{Width: max(0, r.Width), Height: max(0, r.Height)}
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

func (r *renderSizedBox) SelectionSize() Size {
	child := r.Child()
	if child == nil {
		return r.Size()
	}
	childSize := selectionSize(child)
	size := r.Size()
	size.Width = max(size.Width, childSize.Width)
	size.Height = max(size.Height, childSize.Height)
	return size
}
