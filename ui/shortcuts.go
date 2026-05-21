package ui

// ShortcutMap maps Key.MatchString patterns to intents.
type ShortcutMap map[string]Intent

// DefaultShortcuts returns the default app-level key-to-intent bindings.
func DefaultShortcuts() ShortcutMap {
	return ShortcutMap{
		"Escape":    IntentDismiss,
		"Tab":       IntentNextFocus,
		"Shift+Tab": IntentPreviousFocus,
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
