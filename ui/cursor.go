package ui

// Cursor requests a terminal cursor at a child-relative cell position.
type Cursor struct {
	// Col and Row locate the cursor relative to the child origin.
	Col, Row int
	// Shape is the terminal cursor style to request.
	Shape CursorStyle
	// Hidden suppresses the cursor while still painting the child.
	Hidden bool
	// Child is painted before the cursor is requested.
	Child Widget
}

func (w Cursor) ChildWidget() Widget {
	return w.Child
}

func (w Cursor) CreateRenderObject(BuildContext) RenderObject {
	return &RenderCursor{Col: w.Col, Row: w.Row, Shape: w.Shape, Hidden: w.Hidden}
}

func (w Cursor) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*RenderCursor)
	if r.Col != w.Col || r.Row != w.Row || r.Shape != w.Shape || r.Hidden != w.Hidden {
		r.Col = w.Col
		r.Row = w.Row
		r.Shape = w.Shape
		r.Hidden = w.Hidden
		r.MarkNeedsPaint()
	}
}

// RenderCursor paints its child and records cursor state.
type RenderCursor struct {
	SingleChildRenderObject
	Col, Row int
	Shape    CursorStyle
	Hidden   bool
}

func (r *RenderCursor) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(r.layout(ctx, c, false))
}

func (r *RenderCursor) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *RenderCursor) layout(ctx LayoutContext, c Constraints, dry bool) Size {
	if child := r.Child(); child != nil {
		if dry {
			return c.Constrain(DryLayout(ctx, child, c))
		}
		child.Layout(ctx, c)
		return c.Constrain(child.Base().Size())
	}
	return c.Constrain(Size{})
}

func (r *RenderCursor) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
	if !r.Hidden {
		p.ShowCursor(off.X+r.Col, off.Y+r.Row, r.Shape)
	}
}

func (r *RenderCursor) HitTest(*HitTestResult, Point) bool {
	return false
}
