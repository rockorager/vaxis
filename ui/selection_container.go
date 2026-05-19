package ui

// SelectionContainer controls how a subtree participates in ancestor selection.
//
// A non-disabled SelectionContainer is transparent. When Disabled is true,
// descendant selectable widgets are skipped by the nearest SelectionArea for
// drag selection, Ctrl+A, and copy. Use it around content such as controls,
// embedded editors, or decorative text that should not be copied as part of the
// surrounding read-only selection.
type SelectionContainer struct {
	// Disabled excludes Child from ancestor SelectionArea traversal when true.
	Disabled bool
	// Child is the wrapped subtree.
	Child Widget
}

func (w SelectionContainer) WidgetChild() Widget {
	return w.Child
}

func (w SelectionContainer) CreateRenderObject(BuildContext) RenderObject {
	return &renderSelectionContainer{Disabled: w.Disabled}
}

func (w SelectionContainer) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderSelectionContainer)
	if r.Disabled != w.Disabled {
		r.Disabled = w.Disabled
		r.MarkNeedsPaint()
	}
}

type renderSelectionContainer struct {
	SingleChildRenderObject
	Disabled bool
}

func (r *renderSelectionContainer) SelectionDisabled() bool {
	return r.Disabled
}

func (r *renderSelectionContainer) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.SetSize(c.Constrain(Size{}))
		return
	}
	child.Layout(ctx, c)
	r.SetSize(c.Constrain(child.Base().Size()))
}

func (r *renderSelectionContainer) DryLayout(ctx LayoutContext, c Constraints) Size {
	return DryLayout(ctx, r.Child(), c)
}

func (r *renderSelectionContainer) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderSelectionContainer) HitTest(*HitTestResult, Point) bool {
	return false
}
