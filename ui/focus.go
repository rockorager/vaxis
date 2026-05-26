package ui

const elementFocusIndex = -1

type focusTarget struct {
	element element
	index   int
}

type renderFocusHandler interface {
	SetFocusedIndex(int)
}

// FocusScope controls traversal for focusable descendants.
type FocusScope struct {
	// Trap keeps Tab and Shift+Tab traversal inside Child while focus is inside
	// this scope.
	Trap bool
	// AutoFocus moves focus to the first descendant focus target after build
	// when focus is outside the scope.
	AutoFocus bool
	// ReclaimFocus moves focus back into the scope on rebuild when focus is
	// outside the scope after the initial autofocus.
	ReclaimFocus bool
	// Child is the scoped subtree.
	Child Widget
}

func (w FocusScope) CreateElement() element {
	return &focusScopeElement{}
}

type focusScopeElement struct {
	elementBase
	child       element
	autoFocused bool
}

func (e *focusScopeElement) Rebuild() {
	w := e.widget.(FocusScope)
	e.child = e.UpdateChild(e.child, w.Child, nil)
	if !w.AutoFocus {
		e.autoFocused = false
	}
	if w.AutoFocus && e.owner.app.focusedWithin(e) {
		e.autoFocused = true
		return
	}
	if w.AutoFocus && (!e.autoFocused || w.ReclaimFocus) && !e.owner.app.focusedWithin(e) {
		e.owner.app.focusFirstWithin(e)
		e.autoFocused = true
	}
}

func (e *focusScopeElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

// focusWidget makes its child focusable through a FocusNode.
type focusWidget struct {
	// Node controls and observes focus for this widget.
	Node *FocusNode
	// Options controls how this focus target participates in traversal.
	Options FocusOptions
	// Child is the focusable subtree.
	Child Widget
}

// FocusOptions controls focus behavior.
type FocusOptions struct {
	// SkipTraversal removes this target from Tab and Shift+Tab traversal while
	// still allowing it to request focus directly.
	SkipTraversal bool
}

// Focus returns a focusable wrapper around child.
func Focus(node *FocusNode, child Widget) Widget {
	return focusWidget{Node: node, Child: child}
}

// FocusWithOptions returns a focusable wrapper around child with options.
func FocusWithOptions(node *FocusNode, options FocusOptions, child Widget) Widget {
	return focusWidget{Node: node, Options: options, Child: child}
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
	if !w.Options.SkipTraversal {
		e.owner.app.registerFocusable(e)
	}
}

func (e *focusElement) unmounted() {
	w := e.widget.(focusWidget)
	if w.Node != nil {
		w.Node.detach(e)
	}
	if !w.Options.SkipTraversal {
		e.owner.app.unregisterFocusable(e)
	}
}

func (e *focusElement) update(old Widget) {
	oldWidget := old.(focusWidget)
	nextWidget := e.widget.(focusWidget)
	oldNode := oldWidget.Node
	nextNode := nextWidget.Node
	if oldNode != nextNode && oldNode != nil {
		oldNode.detach(e)
		if e.owner.app.focused == (focusTarget{element: e, index: elementFocusIndex}) && oldNode.onChange != nil {
			oldNode.onChange()
		}
	}
	if oldNode != nextNode && nextNode != nil {
		nextNode.attach(e.owner.app, e)
		if e.owner.app.focused == (focusTarget{element: e, index: elementFocusIndex}) && nextNode.onChange != nil {
			nextNode.onChange()
		}
	}
	if oldWidget.Options.SkipTraversal != nextWidget.Options.SkipTraversal {
		if oldWidget.Options.SkipTraversal {
			e.owner.app.registerFocusable(e)
		} else {
			e.owner.app.unregisterFocusable(e)
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
