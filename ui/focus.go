package ui

// FocusWidget makes its child focusable through a FocusNode.
type FocusWidget struct {
	// Node controls and observes focus for this widget.
	Node *FocusNode
	// ChildWidget is the focusable subtree.
	ChildWidget Widget
}

// Focus returns a focusable wrapper around child.
func Focus(node *FocusNode, child Widget) Widget {
	return FocusWidget{Node: node, ChildWidget: child}
}

func (w FocusWidget) CreateElement() Element {
	return &focusElement{}
}

type focusElement struct {
	ElementBase
	child Element
}

func (e *focusElement) mounted() {
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.owner.app.registerFocusable(e)
}

func (e *focusElement) unmounted() {
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.detach(e)
	}
	e.owner.app.unregisterFocusable(e)
}

func (e *focusElement) update(old Widget) {
	oldNode := old.(FocusWidget).Node
	nextNode := e.widget.(FocusWidget).Node
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
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.child = e.UpdateChild(e.child, w.ChildWidget, nil)
}

func (e *focusElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}
