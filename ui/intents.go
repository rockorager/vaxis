package ui

// IntentType identifies the action used to handle an intent.
type IntentType string

// Intent is a typed semantic command. Implementations may carry payload data.
type Intent interface {
	IntentType() IntentType
}

const (
	// ActivateIntentType activates the focused control.
	ActivateIntentType IntentType = "vaxis.activate"
	// DismissIntentType dismisses the nearest dismissible UI surface.
	DismissIntentType IntentType = "vaxis.dismiss"
	// NextFocusIntentType moves focus to the next focusable widget.
	NextFocusIntentType IntentType = "vaxis.next-focus"
	// PreviousFocusIntentType moves focus to the previous focusable widget.
	PreviousFocusIntentType IntentType = "vaxis.previous-focus"
	// ToggleProfileOverlayIntentType toggles the UI profiling overlay.
	ToggleProfileOverlayIntentType IntentType = "vaxis.toggle-profile-overlay"
)

// ActivateIntent activates the focused control.
type ActivateIntent struct{}

func (ActivateIntent) IntentType() IntentType {
	return ActivateIntentType
}

// DismissIntent dismisses the nearest dismissible UI surface.
type DismissIntent struct{}

func (DismissIntent) IntentType() IntentType {
	return DismissIntentType
}

// NextFocusIntent moves focus to the next focusable widget.
type NextFocusIntent struct{}

func (NextFocusIntent) IntentType() IntentType {
	return NextFocusIntentType
}

// PreviousFocusIntent moves focus to the previous focusable widget.
type PreviousFocusIntent struct{}

func (PreviousFocusIntent) IntentType() IntentType {
	return PreviousFocusIntentType
}

// ToggleProfileOverlayIntent toggles the UI profiling overlay.
type ToggleProfileOverlayIntent struct{}

func (ToggleProfileOverlayIntent) IntentType() IntentType {
	return ToggleProfileOverlayIntentType
}
