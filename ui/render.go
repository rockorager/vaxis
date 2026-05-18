package ui

type LayoutContext struct{}

func (LayoutContext) Characters(s string) []Character { return vaxisCharacters(s) }
func (LayoutContext) MeasureText(s string, style Style) Size {
	chars := vaxisCharacters(s)
	w := 0
	for _, ch := range chars {
		w += ch.Width
	}
	return Size{Width: w, Height: 1}
}

type RenderObject interface {
	Base() *RenderObjectBase
	Layout(LayoutContext, Constraints)
	Paint(*Painter, Offset)
	HitTest(*HitTestResult, Point) bool
	VisitChildren(func(RenderObject))
}

type RenderObjectBase struct {
	size             Size
	parentData       any
	owner            *App
	parent           RenderObject
	needsLayout      bool
	needsPaint       bool
	relayoutBoundary bool
}

func (r *RenderObjectBase) Base() *RenderObjectBase { return r }
func (r *RenderObjectBase) Size() Size              { return r.size }
func (r *RenderObjectBase) SetSize(size Size)       { r.size = size }
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
func (r *RenderObjectBase) MarkNeedsPaint() {
	if r.needsPaint {
		return
	}
	r.needsPaint = true
	if r.owner != nil {
		r.owner.RequestFrame()
	}
}
func (r *RenderObjectBase) ParentData() any            { return r.parentData }
func (r *RenderObjectBase) SetParentData(v any)        { r.parentData = v }
func (r *RenderObjectBase) SetRelayoutBoundary(v bool) { r.relayoutBoundary = v }
func (r *RenderObjectBase) NeedsLayout() bool          { return r.needsLayout }
func (r *RenderObjectBase) NeedsPaint() bool           { return r.needsPaint }
func (r *RenderObjectBase) ClearNeedsLayout()          { r.needsLayout = false }
func (r *RenderObjectBase) ClearNeedsPaint()           { r.needsPaint = false }

type LeafRenderObject struct{ RenderObjectBase }

func (r *LeafRenderObject) VisitChildren(func(RenderObject))   {}
func (r *LeafRenderObject) HitTest(*HitTestResult, Point) bool { return false }

type SingleChildRenderObject struct {
	RenderObjectBase
	child RenderObject
}

func (r *SingleChildRenderObject) Child() RenderObject         { return r.child }
func (r *SingleChildRenderObject) SetChild(child RenderObject) { r.child = child }
func (r *SingleChildRenderObject) VisitChildren(fn func(RenderObject)) {
	if r.child != nil {
		fn(r.child)
	}
}

type MultiChildRenderObject struct {
	RenderObjectBase
	children []RenderObject
}

func (r *MultiChildRenderObject) Children() []RenderObject            { return r.children }
func (r *MultiChildRenderObject) SetChildren(children []RenderObject) { r.children = children }
func (r *MultiChildRenderObject) VisitChildren(fn func(RenderObject)) {
	for _, child := range r.children {
		fn(child)
	}
}

type ChildOffsetProvider interface {
	ChildOffset(RenderObject) Offset
}

func (r *SingleChildRenderObject) ChildOffset(RenderObject) Offset { return Offset{} }

func (r *MultiChildRenderObject) ChildOffset(child RenderObject) Offset {
	if pd, ok := child.Base().ParentData().(interface{ RenderOffset() Offset }); ok {
		return pd.RenderOffset()
	}
	return Offset{}
}

type HitTestResult struct{ Path []RenderObject }

type renderObjectElement struct {
	ElementBase
	renderObject RenderObject
	children     []Element
}

func newRenderObjectElement(w RenderObjectWidget) Element { return &renderObjectElement{} }

func (e *renderObjectElement) RenderObject() RenderObject { return e.renderObject }

func (e *renderObjectElement) Rebuild() {
	w := e.widget.(RenderObjectWidget)
	if e.renderObject == nil {
		e.renderObject = w.CreateRenderObject(e.Context())
		e.renderObject.Base().owner = e.owner.app
	} else {
		w.UpdateRenderObject(e.Context(), e.renderObject)
	}
	children := widgetChildren(e.widget)
	next := make([]Element, len(children))
	for i, child := range children {
		next[i] = e.UpdateChild(oldAt(e.children, i), child, i)
	}
	for i := len(children); i < len(e.children); i++ {
		e.UpdateChild(e.children[i], nil, i)
	}
	e.children = next
	e.syncRenderChildren()
}

func (e *renderObjectElement) VisitChildren(fn func(Element)) {
	for _, child := range e.children {
		if child != nil {
			fn(child)
		}
	}
}
func (e *renderObjectElement) Base() *ElementBase { return &e.ElementBase }

func (e *renderObjectElement) FindRenderObject() RenderObject { return e.renderObject }

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

func oldAt(children []Element, i int) Element {
	if i >= 0 && i < len(children) {
		return children[i]
	}
	return nil
}

type childProvider interface{ Children() []Widget }
type singleChildProvider interface{ Child() Widget }
type childWidgetProvider interface{ ChildWidget() Widget }

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
	ElementBase
	child Element
}

func newParentDataElement(w ParentDataWidget) Element { return &parentDataElement{} }
func (e *parentDataElement) Rebuild() {
	w := e.widget.(ParentDataWidget)
	e.child = e.UpdateChild(e.child, w.Child(), nil)
	if e.child != nil {
		if ro := findRenderObject(e.child); ro != nil {
			w.ApplyParentData(ro)
		}
	}
}
func (e *parentDataElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}
