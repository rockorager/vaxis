package ui

// centerWidget centers its child within the space allowed by its parent.
type centerWidget struct{ Child Widget }

// Center returns a widget that centers child.
func Center(child Widget) Widget {
	return centerWidget{Child: child}
}

func (w centerWidget) WidgetChild() Widget {
	return w.Child
}

func (w centerWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderCenter{}
}

func (w centerWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
}

// renderCenter lays out one child and paints it centered.
type renderCenter struct {
	SingleChildRenderObject
	offset Offset
}

func (r *renderCenter) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	size := alignOuterSize(c, Size{})
	if child != nil {
		child.Layout(ctx, alignChildConstraints(c))
		cs := child.Base().Size()
		size = alignOuterSize(c, cs)
		r.offset = Offset{X: max(0, (size.Width-cs.Width)/2), Y: max(0, (size.Height-cs.Height)/2)}
	}
	r.SetSize(size)
}

func (r *renderCenter) DryLayout(ctx LayoutContext, c Constraints) Size {
	if child := r.Child(); child != nil {
		return alignOuterSize(c, DryLayout(ctx, child, alignChildConstraints(c)))
	}
	return alignOuterSize(c, Size{})
}

func (r *renderCenter) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *renderCenter) ChildOffset(RenderObject) Offset {
	return r.offset
}

func (r *renderCenter) HitTest(*HitTestResult, Point) bool {
	return false
}
