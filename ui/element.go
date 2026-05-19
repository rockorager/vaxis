package ui

// Element is a mounted widget instance in the build tree.
type Element interface {
	Base() *ElementBase
	Rebuild()
	VisitChildren(func(Element))
}

// ElementBase stores common mounted element state.
type ElementBase struct {
	widget Widget
	parent Element
	owner  *BuildOwner
	dirty  bool
}

// Base returns the embedded element base.
func (e *ElementBase) Base() *ElementBase {
	return e
}

// Widget returns the element's current widget configuration.
func (e *ElementBase) Widget() Widget {
	return e.widget
}

// MarkNeedsBuild schedules this element to rebuild.
func (e *ElementBase) MarkNeedsBuild() {
	if e.owner == nil || e.self() == nil || e.dirty {
		return
	}
	e.dirty = true
	e.owner.dirty = append(e.owner.dirty, e.self())
	e.owner.app.RequestFrame()
}

// Context returns a build context for this element.
func (e *ElementBase) Context() BuildContext {
	return BuildContext{element: e.self()}
}

// FindRenderObject returns the nearest descendant render object.
func (e *ElementBase) FindRenderObject() RenderObject {
	var found RenderObject
	e.self().VisitChildren(func(child Element) {
		if found == nil {
			found = findRenderObject(child)
		}
	})
	return found
}

// UpdateChild reconciles one child element with a new widget.
func (e *ElementBase) UpdateChild(old Element, next Widget, slot any) Element {
	return e.owner.UpdateChild(e.self(), old, next, slot)
}

func (e *ElementBase) self() Element {
	return e.owner.elements[e]
}

// BuildContext exposes tree-local services while building widgets.
type BuildContext struct{ element Element }

// Widget returns the widget currently being built.
func (c BuildContext) Widget() Widget {
	return c.element.Base().widget
}

// Runtime returns a dispatcher for scheduling work on the UI event loop.
func (c BuildContext) Runtime() Runtime {
	return appRuntime{app: c.element.Base().owner.app}
}

// FindRenderObject returns the nearest render object for this context.
func (c BuildContext) FindRenderObject() RenderObject {
	return findRenderObject(c.element)
}

func findRenderObject(e Element) RenderObject {
	if r, ok := e.(interface{ RenderObject() RenderObject }); ok {
		return r.RenderObject()
	}
	return e.Base().FindRenderObject()
}

// BuildOwner owns element mounting, reconciliation, and dirty rebuilds.
type BuildOwner struct {
	root     Element
	dirty    []Element
	elements map[*ElementBase]Element
	app      *App
	building bool
}

// NewBuildOwner creates an empty build owner.
func NewBuildOwner() *BuildOwner {
	return &BuildOwner{elements: make(map[*ElementBase]Element)}
}

// Mount creates and builds a root element.
func (o *BuildOwner) Mount(root Widget) Element {
	o.root = createElement(root)
	o.mount(o.root, nil, root)
	o.root.Rebuild()
	return o.root
}

// Root returns the mounted root element.
func (o *BuildOwner) Root() Element {
	return o.root
}

// UpdateRoot reconciles the mounted root with a new root widget.
func (o *BuildOwner) UpdateRoot(root Widget) {
	o.root = o.UpdateChild(nil, o.root, root, nil)
}

// BuildScope rebuilds all dirty elements.
func (o *BuildOwner) BuildScope() {
	o.building = true
	defer func() { o.building = false }()
	for len(o.dirty) > 0 {
		dirty := o.dirty
		o.dirty = nil
		for _, e := range dirty {
			if e.Base().owner != nil && e.Base().dirty {
				e.Base().dirty = false
				e.Rebuild()
			}
		}
	}
}

// UpdateChild reconciles an old child element with next.
func (o *BuildOwner) UpdateChild(parent Element, old Element, next Widget, slot any) Element {
	if next == nil {
		if old != nil {
			o.unmount(old)
		}
		return nil
	}
	if old != nil && canUpdate(old.Base().widget, next) {
		previous := old.Base().widget
		old.Base().widget = next
		if u, ok := old.(interface{ update(Widget) }); ok {
			u.update(previous)
		}
		old.Rebuild()
		return old
	}
	if old != nil {
		o.unmount(old)
	}
	child := createElement(next)
	o.mount(child, parent, next)
	child.Rebuild()
	return child
}

func (o *BuildOwner) mount(e Element, parent Element, widget Widget) {
	b := e.Base()
	b.widget = widget
	b.parent = parent
	b.owner = o
	o.elements[b] = e
	if m, ok := e.(interface{ mounted() }); ok {
		m.mounted()
	}
}

func (o *BuildOwner) unmount(e Element) {
	e.VisitChildren(func(child Element) { o.unmount(child) })
	if u, ok := e.(interface{ unmounted() }); ok {
		u.unmounted()
	}
	if d, ok := e.(interface{ dispose() }); ok {
		d.dispose()
	}
	delete(o.elements, e.Base())
	e.Base().owner = nil
	e.Base().parent = nil
}
