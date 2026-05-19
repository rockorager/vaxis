package ui

// Stack paints children on top of each other.
//
// Non-positioned children are laid out loosely and determine the stack's
// natural size. Positioned children are then laid out and painted at their
// requested offsets within that size. Later children paint above earlier
// children and receive pointer events first.
type Stack struct {
	// Alignment places non-positioned children inside the stack. The zero value
	// is CenterAlign.
	Alignment Alignment
	// Children is the ordered back-to-front child list.
	Children []Widget
}

func (w Stack) WidgetChildren() []Widget {
	return w.Children
}

func (w Stack) CreateRenderObject(BuildContext) RenderObject {
	return &renderStack{Alignment: w.Alignment}
}

func (w Stack) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderStack)
	if r.Alignment != w.Alignment {
		r.Alignment = w.Alignment
		r.MarkNeedsLayout()
	}
}

// Positioned places a child at an offset inside an ancestor Stack.
type Positioned struct {
	// Left is the child X offset.
	Left int
	// Top is the child Y offset.
	Top int
	// Child is the positioned child.
	Child Widget
}

func (w Positioned) WidgetChild() Widget {
	return w.Child
}

func (w Positioned) ApplyParentData(ro RenderObject) {
	pd, _ := ro.Base().ParentData().(StackParentData)
	next := StackParentData{Positioned: true, Left: w.Left, Top: w.Top, Offset: pd.Offset}
	if pd != next {
		ro.Base().SetParentData(next)
		if parent := ro.Base().parent; parent != nil {
			parent.Base().MarkNeedsLayout()
		}
	}
}

// StackParentData stores layout data for children of Stack.
type StackParentData struct {
	// Positioned reports whether Left and Top should be used.
	Positioned bool
	// Left is the positioned X offset.
	Left int
	// Top is the positioned Y offset.
	Top int
	// Offset is the child paint offset computed by renderStack.
	Offset Offset
}

// RenderOffset returns the child's paint offset.
func (d StackParentData) RenderOffset() Offset {
	return d.Offset
}

type renderStack struct {
	MultiChildRenderObject
	Alignment Alignment
}

func (r *renderStack) Layout(ctx LayoutContext, c Constraints) {
	size, childSizes := r.layoutSizes(ctx, c, false)
	for i, child := range r.Children() {
		pd, _ := child.Base().ParentData().(StackParentData)
		if pd.Positioned {
			pd.Offset = Offset{X: pd.Left, Y: pd.Top}
		} else {
			pd.Offset = alignOffset(size, childSizes[i], stackAlignment(r.Alignment))
		}
		child.Base().SetParentData(pd)
	}
	r.SetSize(size)
}

func (r *renderStack) DryLayout(ctx LayoutContext, c Constraints) Size {
	size, _ := r.layoutSizes(ctx, c, true)
	return size
}

func (r *renderStack) layoutSizes(ctx LayoutContext, c Constraints, dry bool) (Size, []Size) {
	children := r.Children()
	childSizes := make([]Size, len(children))
	size := Size{}
	for i, child := range children {
		pd, _ := child.Base().ParentData().(StackParentData)
		if pd.Positioned {
			continue
		}
		childSize := r.layoutChild(ctx, child, stackLooseConstraints(c), dry)
		childSizes[i] = childSize
		size.Width = max(size.Width, childSize.Width)
		size.Height = max(size.Height, childSize.Height)
	}
	size = c.Constrain(size)
	for i, child := range children {
		pd, _ := child.Base().ParentData().(StackParentData)
		if !pd.Positioned {
			continue
		}
		childSize := r.layoutChild(ctx, child, Loose(size), dry)
		childSizes[i] = childSize
	}
	return size, childSizes
}

func stackLooseConstraints(c Constraints) Constraints {
	return Constraints{MaxWidth: c.MaxWidth, MaxHeight: c.MaxHeight}
}

func (r *renderStack) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
	if dry {
		return DryLayout(ctx, child, c)
	}
	child.Layout(ctx, c)
	return child.Base().Size()
}

func (r *renderStack) Paint(p *Painter, off Offset) {
	for _, child := range r.Children() {
		pd, _ := child.Base().ParentData().(StackParentData)
		child.Paint(p, off.Add(pd.Offset))
	}
}

func (r *renderStack) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderStack) HitTestChildrenReverse() bool {
	return true
}

func stackAlignment(a Alignment) Alignment {
	if a == (Alignment{}) {
		return CenterAlign
	}
	return a
}

// IndexedStack keeps every child mounted but paints only one child.
//
// All children are laid out with the same loose constraints and the stack size
// is the maximum child size. Only Children[Index] is painted and hit-tested.
type IndexedStack struct {
	// Index selects the visible child.
	Index int
	// Alignment places the visible child inside the stack. The zero value is
	// CenterAlign.
	Alignment Alignment
	// Children is the ordered child list.
	Children []Widget
}

func (w IndexedStack) WidgetChildren() []Widget {
	return w.Children
}

func (w IndexedStack) CreateRenderObject(BuildContext) RenderObject {
	return &renderIndexedStack{Index: w.Index, Alignment: w.Alignment}
}

func (w IndexedStack) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderIndexedStack)
	if r.Index != w.Index || r.Alignment != w.Alignment {
		r.Index = w.Index
		r.Alignment = w.Alignment
		r.MarkNeedsLayout()
	}
}

type renderIndexedStack struct {
	MultiChildRenderObject
	Index     int
	Alignment Alignment
}

func (r *renderIndexedStack) Layout(ctx LayoutContext, c Constraints) {
	size, childSizes := r.layoutSizes(ctx, c, false)
	for i, child := range r.Children() {
		pd, _ := child.Base().ParentData().(StackParentData)
		pd.Offset = alignOffset(size, childSizes[i], stackAlignment(r.Alignment))
		child.Base().SetParentData(pd)
	}
	r.SetSize(size)
}

func (r *renderIndexedStack) DryLayout(ctx LayoutContext, c Constraints) Size {
	size, _ := r.layoutSizes(ctx, c, true)
	return size
}

func (r *renderIndexedStack) layoutSizes(ctx LayoutContext, c Constraints, dry bool) (Size, []Size) {
	children := r.Children()
	childSizes := make([]Size, len(children))
	size := Size{}
	for i, child := range children {
		childSize := r.layoutChild(ctx, child, stackLooseConstraints(c), dry)
		childSizes[i] = childSize
		size.Width = max(size.Width, childSize.Width)
		size.Height = max(size.Height, childSize.Height)
	}
	return c.Constrain(size), childSizes
}

func (r *renderIndexedStack) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
	if dry {
		return DryLayout(ctx, child, c)
	}
	child.Layout(ctx, c)
	return child.Base().Size()
}

func (r *renderIndexedStack) Paint(p *Painter, off Offset) {
	child := r.activeChild()
	if child == nil {
		return
	}
	pd, _ := child.Base().ParentData().(StackParentData)
	child.Paint(p, off.Add(pd.Offset))
}

func (r *renderIndexedStack) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderIndexedStack) HitTestChild(child RenderObject) bool {
	return child == r.activeChild()
}

func (r *renderIndexedStack) activeChild() RenderObject {
	children := r.Children()
	if r.Index < 0 || r.Index >= len(children) {
		return nil
	}
	return children[r.Index]
}
