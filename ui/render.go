package ui

// LayoutContext carries helpers available while measuring and laying out render objects.
type LayoutContext struct{}

// Characters splits s into terminal-width grapheme characters.
func (LayoutContext) Characters(s string) []Character {
	return vaxisCharacters(s)
}

// MeasureText returns the terminal cell size needed to draw s on one line.
func (LayoutContext) MeasureText(s string, style Style) Size {
	chars := vaxisCharacters(s)
	w := 0
	for _, ch := range chars {
		w += ch.Width
	}
	return Size{Width: w, Height: 1}
}

// RenderObject is the layout, paint, and hit-test object produced by a widget.
type RenderObject interface {
	Base() *RenderObjectBase
	Layout(LayoutContext, Constraints)
	Paint(*Painter, Offset)
	HitTest(*HitTestResult, Point) bool
	VisitChildren(func(RenderObject))
}

// DryLayouter can compute a size for constraints without mutating layout state.
type DryLayouter interface {
	DryLayout(LayoutContext, Constraints) Size
}

// DryLayout computes ro's size for c without requiring a full layout pass.
func DryLayout(ctx LayoutContext, ro RenderObject, c Constraints) Size {
	if ro == nil {
		return c.Constrain(Size{})
	}
	if d, ok := ro.(DryLayouter); ok {
		return c.Constrain(d.DryLayout(ctx, c))
	}
	return c.Constrain(ro.Base().Size())
}

// RenderObjectBase stores common render tree state.
type RenderObjectBase struct {
	size             Size
	parentData       any
	owner            *App
	parent           RenderObject
	needsLayout      bool
	needsPaint       bool
	relayoutBoundary bool
}

// Base returns the embedded render object base.
func (r *RenderObjectBase) Base() *RenderObjectBase {
	return r
}

// Size returns the render object's most recent layout size.
func (r *RenderObjectBase) Size() Size {
	return r.size
}

// SetSize records the render object's layout size.
func (r *RenderObjectBase) SetSize(size Size) {
	r.size = size
}

// MarkNeedsLayout marks this object and eligible ancestors dirty for layout.
func (r *RenderObjectBase) MarkNeedsLayout() {
	if r.needsLayout {
		return
	}
	r.needsLayout = true
	r.MarkNeedsPaint()
	if r.parent != nil && !r.relayoutBoundary {
		r.parent.Base().MarkNeedsLayout()
	}
}

// MarkNeedsPaint marks this object dirty for paint and requests a frame.
func (r *RenderObjectBase) MarkNeedsPaint() {
	if r.needsPaint {
		return
	}
	r.needsPaint = true
	if r.owner != nil {
		r.owner.RequestFrame()
	}
}

// ParentData returns parent-specific layout data for this render object.
func (r *RenderObjectBase) ParentData() any {
	return r.parentData
}

// SetParentData stores parent-specific layout data for this render object.
func (r *RenderObjectBase) SetParentData(v any) {
	r.parentData = v
}

// SetRelayoutBoundary controls whether layout invalidation bubbles to ancestors.
func (r *RenderObjectBase) SetRelayoutBoundary(v bool) {
	r.relayoutBoundary = v
}

// NeedsLayout reports whether this object has pending layout work.
func (r *RenderObjectBase) NeedsLayout() bool {
	return r.needsLayout
}

// NeedsPaint reports whether this object has pending paint work.
func (r *RenderObjectBase) NeedsPaint() bool {
	return r.needsPaint
}

// ClearNeedsLayout clears the layout dirty flag.
func (r *RenderObjectBase) ClearNeedsLayout() {
	r.needsLayout = false
}

// ClearNeedsPaint clears the paint dirty flag.
func (r *RenderObjectBase) ClearNeedsPaint() {
	r.needsPaint = false
}

// LeafRenderObject is a RenderObjectBase for render objects without children.
type LeafRenderObject struct{ RenderObjectBase }

func (r *LeafRenderObject) VisitChildren(func(RenderObject)) {
}

func (r *LeafRenderObject) HitTest(*HitTestResult, Point) bool {
	return false
}

// SingleChildRenderObject is a RenderObjectBase for render objects with one child.
type SingleChildRenderObject struct {
	RenderObjectBase
	child RenderObject
}

// Child returns the current child render object.
func (r *SingleChildRenderObject) Child() RenderObject {
	return r.child
}

// SetChild replaces the current child render object.
func (r *SingleChildRenderObject) SetChild(child RenderObject) {
	if r.child != nil && r.child != child {
		detachRenderTree(r.child)
	}
	r.child = child
}

func (r *SingleChildRenderObject) VisitChildren(fn func(RenderObject)) {
	if r.child != nil {
		fn(r.child)
	}
}

// MultiChildRenderObject is a RenderObjectBase for render objects with ordered children.
type MultiChildRenderObject struct {
	RenderObjectBase
	children []RenderObject
}

// Children returns the current child render objects.
func (r *MultiChildRenderObject) Children() []RenderObject {
	return r.children
}

// SetChildren replaces the current child render objects.
func (r *MultiChildRenderObject) SetChildren(children []RenderObject) {
	for _, old := range r.children {
		kept := false
		for _, child := range children {
			if old == child {
				kept = true
				break
			}
		}
		if !kept {
			detachRenderTree(old)
		}
	}
	r.children = children
}

func (r *MultiChildRenderObject) VisitChildren(fn func(RenderObject)) {
	for _, child := range r.children {
		fn(child)
	}
}

// ChildOffsetProvider reports the paint offset of a child for hit testing.
type ChildOffsetProvider interface {
	ChildOffset(RenderObject) Offset
}

func (r *SingleChildRenderObject) ChildOffset(RenderObject) Offset {
	return Offset{}
}

func (r *MultiChildRenderObject) ChildOffset(child RenderObject) Offset {
	if pd, ok := child.Base().ParentData().(interface{ RenderOffset() Offset }); ok {
		return pd.RenderOffset()
	}
	return Offset{}
}

// HitTestResult stores a render-object hit path.
type HitTestResult struct{ Path []RenderObject }

type renderObjectElement struct {
	elementBase
	renderObject RenderObject
	children     []element
}

func newRenderObjectElement(w RenderObjectWidget) element {
	return &renderObjectElement{}
}

func (e *renderObjectElement) RenderObject() RenderObject {
	return e.renderObject
}

func (e *renderObjectElement) Rebuild() {
	w := e.widget.(RenderObjectWidget)
	if e.renderObject == nil {
		e.renderObject = w.CreateRenderObject(e.Context())
		e.renderObject.Base().owner = e.owner.app
	} else {
		w.UpdateRenderObject(e.Context(), e.renderObject)
	}
	children := widgetChildren(e.widget)
	next := make([]element, len(children))
	for i, child := range children {
		next[i] = e.UpdateChild(oldAt(e.children, i), child, i)
	}
	for i := len(children); i < len(e.children); i++ {
		e.UpdateChild(e.children[i], nil, i)
	}
	e.children = next
	e.syncRenderChildren()
}

func (e *renderObjectElement) VisitChildren(fn func(element)) {
	for _, child := range e.children {
		if child != nil {
			fn(child)
		}
	}
}

func (e *renderObjectElement) Base() *elementBase {
	return &e.elementBase
}

func (e *renderObjectElement) FindRenderObject() RenderObject {
	return e.renderObject
}

func (e *renderObjectElement) syncRenderChildren() {
	renders := make([]RenderObject, 0, len(e.children))
	for _, child := range e.children {
		if child == nil {
			continue
		}
		if ro := findRenderObject(child); ro != nil {
			renders = append(renders, ro)
		}
	}
	switch r := e.renderObject.(type) {
	case interface{ SetChild(RenderObject) }:
		if len(renders) > 0 {
			r.SetChild(renders[0])
			attachRenderTree(renders[0], e.owner.app, e.renderObject)
		} else {
			r.SetChild(nil)
		}
	case interface{ SetChildren([]RenderObject) }:
		r.SetChildren(renders)
		for _, child := range renders {
			attachRenderTree(child, e.owner.app, e.renderObject)
		}
	}
}

func attachRenderTree(ro RenderObject, owner *App, parent RenderObject) {
	ro.Base().owner = owner
	ro.Base().parent = parent
	ro.VisitChildren(func(child RenderObject) { attachRenderTree(child, owner, ro) })
}

func detachRenderTree(ro RenderObject) {
	ro.Base().owner = nil
	ro.Base().parent = nil
	ro.VisitChildren(detachRenderTree)
}

func oldAt(children []element, i int) element {
	if i >= 0 && i < len(children) {
		return children[i]
	}
	return nil
}

type (
	childProvider       interface{ Children() []Widget }
	singleChildProvider interface{ Child() Widget }
	childWidgetProvider interface{ ChildWidget() Widget }
)

func widgetChildren(w Widget) []Widget {
	if c, ok := w.(childProvider); ok {
		return c.Children()
	}
	if c, ok := w.(singleChildProvider); ok {
		if child := c.Child(); child != nil {
			return []Widget{child}
		}
	}
	if c, ok := w.(childWidgetProvider); ok {
		if child := c.ChildWidget(); child != nil {
			return []Widget{child}
		}
	}
	return nil
}

type parentDataElement struct {
	elementBase
	child element
}

func newParentDataElement(w ParentDataWidget) element {
	return &parentDataElement{}
}

func (e *parentDataElement) Rebuild() {
	w := e.widget.(ParentDataWidget)
	e.child = e.UpdateChild(e.child, w.Child(), nil)
	if e.child != nil {
		if ro := findRenderObject(e.child); ro != nil {
			w.ApplyParentData(ro)
		}
	}
}

func (e *parentDataElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}
