package ui

// focusWidget makes its child focusable through a FocusNode.
type focusWidget struct {
	// Node controls and observes focus for this widget.
	Node *FocusNode
	// Child is the focusable subtree.
	Child Widget
}

// Focus returns a focusable wrapper around child.
func Focus(node *FocusNode, child Widget) Widget {
	return focusWidget{Node: node, Child: child}
}

func (w focusWidget) CreateElement() element {
	return &focusElement{}
}

type focusElement struct {
	elementBase
	child element
}

func (e *focusElement) mounted() {
	w := e.widget.(focusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.owner.app.registerFocusable(e)
}

func (e *focusElement) unmounted() {
	w := e.widget.(focusWidget)
	if w.Node != nil {
		w.Node.detach(e)
	}
	e.owner.app.unregisterFocusable(e)
}

func (e *focusElement) update(old Widget) {
	oldNode := old.(focusWidget).Node
	nextNode := e.widget.(focusWidget).Node
	if oldNode == nextNode {
		return
	}
	if oldNode != nil {
		oldNode.detach(e)
		if e.owner.app.focused == e && oldNode.onChange != nil {
			oldNode.onChange()
		}
	}
	if nextNode != nil {
		nextNode.attach(e.owner.app, e)
		if e.owner.app.focused == e && nextNode.onChange != nil {
			nextNode.onChange()
		}
	}
}

func (e *focusElement) Rebuild() {
	w := e.widget.(focusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.child = e.UpdateChild(e.child, w.Child, nil)
}

func (e *focusElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}
