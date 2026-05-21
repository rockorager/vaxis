package ui

// ShortcutMap maps Key.MatchString patterns to intents.
type ShortcutMap map[string]Intent

// DefaultShortcuts returns the default app-level key-to-intent bindings.
//
// The returned map is a fresh copy that callers may modify before passing it
// to WithShortcuts.
func DefaultShortcuts() ShortcutMap {
	return ShortcutMap{
		"Escape":    DismissIntent{},
		"Tab":       NextFocusIntent{},
		"Shift+Tab": PreviousFocusIntent{},
	}
}

func cloneShortcuts(shortcuts ShortcutMap) ShortcutMap {
	if shortcuts == nil {
		return nil
	}
	clone := make(ShortcutMap, len(shortcuts))
	for binding, intent := range shortcuts {
		clone[binding] = intent
	}
	return clone
}

// Shortcuts maps key bindings to intents.
//
// Shortcuts only handles a key when invoking the mapped intent is handled by an
// Actions or DefaultActions provider. Otherwise, the key event continues down
// the normal event path.
type Shortcuts struct {
	// Bindings maps Key.MatchString patterns to intents.
	Bindings ShortcutMap
	// Child is the subtree that receives events after unhandled shortcuts.
	Child Widget
}

func (w Shortcuts) CreateElement() element {
	return &shortcutsElement{}
}

type shortcutsElement struct {
	elementBase
	child element
}

func (e *shortcutsElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(Shortcuts).Child, nil)
}

func (e *shortcutsElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *shortcutsElement) HandleEvent(ctx EventContext, ev Event) EventResult {
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	if keyIsRelease(key) {
		return EventIgnored
	}
	for binding, intent := range e.widget.(Shortcuts).Bindings {
		if key.MatchString(binding) && ctx.Invoke(intent) == EventHandled {
			return EventHandled
		}
	}
	return EventIgnored
}
