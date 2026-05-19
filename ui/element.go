package ui

import "sort"

// element is a mounted widget instance in the build tree.
type element interface {
	Base() *elementBase
	Rebuild()
	VisitChildren(func(element))
}

// elementBase stores common mounted element state.
type elementBase struct {
	widget Widget
	parent element
	owner  *buildOwner
	dirty  bool
}

// Base returns the embedded element base.
func (e *elementBase) Base() *elementBase {
	return e
}

// Widget returns the element's current widget configuration.
func (e *elementBase) Widget() Widget {
	return e.widget
}

// MarkNeedsBuild schedules this element to rebuild.
func (e *elementBase) MarkNeedsBuild() {
	if e.owner == nil || e.self() == nil || e.dirty {
		return
	}
	e.dirty = true
	e.owner.dirty = append(e.owner.dirty, e.self())
	e.owner.app.RequestFrame()
}

// Context returns a build context for this element.
func (e *elementBase) Context() BuildContext {
	return BuildContext{element: e.self()}
}

// FindRenderObject returns the nearest descendant render object.
func (e *elementBase) FindRenderObject() RenderObject {
	var found RenderObject
	e.self().VisitChildren(func(child element) {
		if found == nil {
			found = findRenderObject(child)
		}
	})
	return found
}

// UpdateChild reconciles one child element with a new widget.
func (e *elementBase) UpdateChild(old element, next Widget, slot any) element {
	return e.owner.UpdateChild(e.self(), old, next, slot)
}

func (e *elementBase) self() element {
	return e.owner.elements[e]
}

// BuildContext exposes tree-local services while building widgets.
type BuildContext struct{ element element }

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

func findRenderObject(e element) RenderObject {
	if r, ok := e.(interface{ RenderObject() RenderObject }); ok {
		return r.RenderObject()
	}
	return e.Base().FindRenderObject()
}

// buildOwner owns element mounting, reconciliation, and dirty rebuilds.
type buildOwner struct {
	root     element
	dirty    []element
	elements map[*elementBase]element
	app      *App
	building bool
}

// newBuildOwner creates an empty build owner.
func newBuildOwner() *buildOwner {
	return &buildOwner{elements: make(map[*elementBase]element)}
}

// Mount creates and builds a root element.
func (o *buildOwner) Mount(root Widget) element {
	o.root = createElement(root)
	o.mount(o.root, nil, root)
	o.root.Rebuild()
	return o.root
}

// Root returns the mounted root element.
func (o *buildOwner) Root() element {
	return o.root
}

// UpdateRoot reconciles the mounted root with a new root widget.
func (o *buildOwner) UpdateRoot(root Widget) {
	o.root = o.UpdateChild(nil, o.root, root, nil)
}

// BuildScope rebuilds all dirty elements.
func (o *buildOwner) BuildScope() {
	o.building = true
	defer func() { o.building = false }()
	for len(o.dirty) > 0 {
		dirty := o.dirty
		o.dirty = nil
		sort.SliceStable(dirty, func(i, j int) bool {
			return elementDepth(dirty[i]) < elementDepth(dirty[j])
		})
		for _, e := range dirty {
			if e.Base().owner != nil && e.Base().dirty {
				e.Base().dirty = false
				e.Rebuild()
			}
		}
	}
}

// UpdateChild reconciles an old child element with next.
func (o *buildOwner) UpdateChild(parent element, old element, next Widget, slot any) element {
	if next == nil {
		if old != nil {
			o.unmount(old)
		}
		return nil
	}
	if old != nil && canUpdate(old.Base().widget, next) {
		previous := old.Base().widget
		old.Base().widget = next
		old.Base().dirty = false
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

func elementDepth(e element) int {
	depth := 0
	for parent := e.Base().parent; parent != nil; parent = parent.Base().parent {
		depth++
	}
	return depth
}

func (o *buildOwner) mount(e element, parent element, widget Widget) {
	b := e.Base()
	b.widget = widget
	b.parent = parent
	b.owner = o
	o.elements[b] = e
	if m, ok := e.(interface{ mounted() }); ok {
		m.mounted()
	}
}

func (o *buildOwner) unmount(e element) {
	e.VisitChildren(func(child element) { o.unmount(child) })
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
