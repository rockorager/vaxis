package ui

// ConstrainedBox applies additional constraints to its child.
type ConstrainedBox struct {
	// Constraints are enforced inside the parent constraints.
	Constraints Constraints
	// Child is laid out with the combined constraints.
	Child Widget
}

func (w ConstrainedBox) WidgetChild() Widget {
	return w.Child
}

func (w ConstrainedBox) CreateRenderObject(BuildContext) RenderObject {
	return &renderConstrainedBox{AdditionalConstraints: w.Constraints}
}

func (w ConstrainedBox) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderConstrainedBox)
	if r.AdditionalConstraints != w.Constraints {
		r.AdditionalConstraints = w.Constraints
		r.MarkNeedsLayout()
	}
}

// renderConstrainedBox enforces additional constraints during layout.
type renderConstrainedBox struct {
	SingleChildRenderObject
	AdditionalConstraints Constraints
}

func (r *renderConstrainedBox) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(r.layout(ctx, c, false))
}

func (r *renderConstrainedBox) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *renderConstrainedBox) layout(ctx LayoutContext, c Constraints, dry bool) Size {
	constraints := normalizeAdditionalConstraints(r.AdditionalConstraints).Enforce(c)
	child := r.Child()
	if child == nil {
		return constraints.Constrain(Size{})
	}
	if dry {
		return c.Constrain(DryLayout(ctx, child, constraints))
	}
	child.Layout(ctx, constraints)
	return c.Constrain(child.Base().Size())
}

func (r *renderConstrainedBox) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderConstrainedBox) HitTest(*HitTestResult, Point) bool {
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
