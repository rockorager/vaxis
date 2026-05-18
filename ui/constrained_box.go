package ui

type ConstrainedBox struct {
	Constraints Constraints
	Child       Widget
}

func (w ConstrainedBox) ChildWidget() Widget {
	return w.Child
}

func (w ConstrainedBox) CreateRenderObject(BuildContext) RenderObject {
	return &RenderConstrainedBox{AdditionalConstraints: w.Constraints}
}

func (w ConstrainedBox) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*RenderConstrainedBox)
	if r.AdditionalConstraints != w.Constraints {
		r.AdditionalConstraints = w.Constraints
		r.MarkNeedsLayout()
	}
}

type RenderConstrainedBox struct {
	SingleChildRenderObject
	AdditionalConstraints Constraints
}

func (r *RenderConstrainedBox) Layout(ctx LayoutContext, c Constraints) {
	constraints := normalizeAdditionalConstraints(r.AdditionalConstraints).Enforce(c)
	child := r.Child()
	if child == nil {
		r.SetSize(constraints.Constrain(Size{}))
		return
	}
	child.Layout(ctx, constraints)
	r.SetSize(c.Constrain(child.Base().Size()))
}

func (r *RenderConstrainedBox) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *RenderConstrainedBox) HitTest(*HitTestResult, Point) bool {
	return false
}

func normalizeAdditionalConstraints(c Constraints) Constraints {
	if c.MaxWidth == 0 {
		c.MaxWidth = Unbounded
	}
	if c.MaxHeight == 0 {
		c.MaxHeight = Unbounded
	}
	return c
}
