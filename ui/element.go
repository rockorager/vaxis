package ui

type Element interface {
	Base() *ElementBase
	Rebuild()
	VisitChildren(func(Element))
}

type ElementBase struct {
	widget Widget
	parent Element
	owner  *BuildOwner
	dirty  bool
}

func (e *ElementBase) Base() *ElementBase { return e }
func (e *ElementBase) Widget() Widget     { return e.widget }
func (e *ElementBase) MarkNeedsBuild() {
	if e.owner == nil || e.dirty {
		return
	}
	e.dirty = true
	e.owner.dirty = append(e.owner.dirty, e.self())
}

func (e *ElementBase) Context() BuildContext { return BuildContext{element: e.self()} }
func (e *ElementBase) FindRenderObject() RenderObject {
	var found RenderObject
	e.self().VisitChildren(func(child Element) {
		if found == nil {
			found = findRenderObject(child)
		}
	})
	return found
}

func (e *ElementBase) UpdateChild(old Element, next Widget, slot any) Element {
	return e.owner.UpdateChild(e.self(), old, next, slot)
}

func (e *ElementBase) self() Element { return e.owner.elements[e] }

type BuildContext struct{ element Element }

func (c BuildContext) Widget() Widget { return c.element.Base().widget }
func (c BuildContext) FindRenderObject() RenderObject {
	return findRenderObject(c.element)
}

func findRenderObject(e Element) RenderObject {
	if r, ok := e.(interface{ RenderObject() RenderObject }); ok {
		return r.RenderObject()
	}
	return e.Base().FindRenderObject()
}

type BuildOwner struct {
	root     Element
	dirty    []Element
	elements map[*ElementBase]Element
}

func NewBuildOwner() *BuildOwner { return &BuildOwner{elements: make(map[*ElementBase]Element)} }

func (o *BuildOwner) Mount(root Widget) Element {
	o.root = createElement(root)
	o.mount(o.root, nil, root)
	o.root.Rebuild()
	return o.root
}

func (o *BuildOwner) Root() Element { return o.root }

func (o *BuildOwner) UpdateRoot(root Widget) {
	o.root = o.UpdateChild(nil, o.root, root, nil)
}

func (o *BuildOwner) BuildScope() {
	for len(o.dirty) > 0 {
		dirty := o.dirty
		o.dirty = nil
		for _, e := range dirty {
			if e.Base().dirty {
				e.Base().dirty = false
				e.Rebuild()
			}
		}
	}
}

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
}

func (o *BuildOwner) unmount(e Element) {
	e.VisitChildren(func(child Element) { o.unmount(child) })
	if d, ok := e.(interface{ dispose() }); ok {
		d.dispose()
	}
	delete(o.elements, e.Base())
}
