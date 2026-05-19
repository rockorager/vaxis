package ui

// Keymap handles matching key bindings before events reach its focused descendants.
type Keymap struct {
	// Bindings maps Key.MatchString patterns to callbacks.
	Bindings map[string]VoidCallback
	// Child is the subtree that receives events after unhandled bindings.
	Child Widget
}

func (w Keymap) CreateElement() element {
	return &keymapElement{}
}

type keymapElement struct {
	elementBase
	child element
}

func (e *keymapElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(Keymap).Child, nil)
}

func (e *keymapElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *keymapElement) HandleEvent(ctx EventContext, ev Event) EventResult {
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	if keyIsRelease(key) {
		return EventIgnored
	}
	for binding, cb := range e.widget.(Keymap).Bindings {
		if key.MatchString(binding) {
			if cb != nil {
				cb(ctx)
			}
			return EventHandled
		}
	}
	return EventIgnored
}
