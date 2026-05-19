package ui

// paddingWidget insets its child by a fixed number of cells.
type paddingWidget struct {
	// Insets is the empty space around the child.
	Insets Insets
	// Child is laid out inside the inset space.
	Child Widget
}

// Padding returns a widget that insets child.
func Padding(in Insets, child Widget) Widget {
	return paddingWidget{Insets: in, Child: child}
}

func (w paddingWidget) WidgetChild() Widget {
	return w.Child
}

func (w paddingWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderPadding{Insets: w.Insets}
}

func (w paddingWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	ro.(*renderPadding).Insets = w.Insets
}

// renderPadding lays out and paints one inset child.
type renderPadding struct {
	SingleChildRenderObject
	Insets Insets
}

func (r *renderPadding) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(r.layout(ctx, c, false))
}

func (r *renderPadding) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *renderPadding) layout(ctx LayoutContext, c Constraints, dry bool) Size {
	childSize := Size{}
	if child := r.Child(); child != nil {
		childConstraints := c.Deflate(r.Insets)
		if dry {
			childSize = DryLayout(ctx, child, childConstraints)
		} else {
			child.Layout(ctx, childConstraints)
			childSize = child.Base().Size()
		}
	}
	return c.Constrain(childSize.Inflate(r.Insets))
}

func (r *renderPadding) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(Offset{X: r.Insets.Left, Y: r.Insets.Top}))
	}
}

func (r *renderPadding) ChildOffset(RenderObject) Offset {
	return Offset{X: r.Insets.Left, Y: r.Insets.Top}
}

func (r *renderPadding) HitTest(*HitTestResult, Point) bool {
	return false
}
