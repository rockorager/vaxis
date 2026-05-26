package ui

// autoFocus requests focus for the first focusable descendant when mounted.
type autoFocus struct {
	Child Widget
}

func (w autoFocus) CreateElement() element {
	return &autoFocusElement{}
}

type autoFocusElement struct {
	elementBase
	child   element
	focused bool
}

func (e *autoFocusElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(autoFocus).Child, nil)
	if !e.focused {
		e.owner.app.focusFirstWithin(e)
		e.focused = true
	}
}

func (e *autoFocusElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}
