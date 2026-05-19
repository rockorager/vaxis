package ui

// centerWidget centers its child within the space allowed by its parent.
type centerWidget struct{ ChildWidget Widget }

// Center returns a widget that centers child.
func Center(child Widget) Widget {
	return centerWidget{ChildWidget: child}
}

func (w centerWidget) Child() Widget {
	return w.ChildWidget
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
	size := c.Constrain(Size{Width: maxFinite(c.MaxWidth), Height: maxFinite(c.MaxHeight)})
	child := r.Child()
	if child != nil {
		child.Layout(ctx, Loose(size))
		cs := child.Base().Size()
		r.offset = Offset{X: max(0, (size.Width-cs.Width)/2), Y: max(0, (size.Height-cs.Height)/2)}
	}
	r.SetSize(size)
}

func (r *renderCenter) DryLayout(_ LayoutContext, c Constraints) Size {
	return c.Constrain(Size{Width: maxFinite(c.MaxWidth), Height: maxFinite(c.MaxHeight)})
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
