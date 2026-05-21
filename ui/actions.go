package ui

// ActionFunc handles an intent.
type ActionFunc func(EventContext) EventResult

// Actions provides intent handlers for its subtree.
type Actions struct {
	// Bindings maps intents to handlers.
	Bindings map[Intent]ActionFunc
	// Child is the subtree that can invoke these actions.
	Child Widget
}

func (w Actions) CreateElement() element {
	return &actionsElement{}
}

type actionsElement struct {
	elementBase
	child element
}

func (e *actionsElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(Actions).Child, nil)
}

func (e *actionsElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *actionsElement) action(intent Intent) (ActionFunc, bool) {
	action, ok := e.widget.(Actions).Bindings[intent]
	return action, ok
}
