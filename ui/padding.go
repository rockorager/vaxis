package ui

type PaddingWidget struct {
	Insets      Insets
	ChildWidget Widget
}

func Padding(in Insets, child Widget) Widget {
	return PaddingWidget{Insets: in, ChildWidget: child}
}

func (w PaddingWidget) Child() Widget {
	return w.ChildWidget
}

func (w PaddingWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderPadding{Insets: w.Insets}
}

func (w PaddingWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	ro.(*RenderPadding).Insets = w.Insets
}

type RenderPadding struct {
	SingleChildRenderObject
	Insets Insets
}

func (r *RenderPadding) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(r.layout(ctx, c, false))
}

func (r *RenderPadding) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *RenderPadding) layout(ctx LayoutContext, c Constraints, dry bool) Size {
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

func (r *RenderPadding) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(Offset{X: r.Insets.Left, Y: r.Insets.Top}))
	}
}

func (r *RenderPadding) ChildOffset(RenderObject) Offset {
	return Offset{X: r.Insets.Left, Y: r.Insets.Top}
}

func (r *RenderPadding) HitTest(*HitTestResult, Point) bool {
	return false
}
