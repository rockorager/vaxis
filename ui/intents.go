package ui

// Intent identifies a semantic command.
type Intent string

const (
	// IntentNextFocus moves focus to the next focusable widget.
	IntentNextFocus Intent = "vaxis.next-focus"
	// IntentPreviousFocus moves focus to the previous focusable widget.
	IntentPreviousFocus Intent = "vaxis.previous-focus"
	// IntentToggleProfileOverlay toggles the UI profiling overlay.
	IntentToggleProfileOverlay Intent = "vaxis.toggle-profile-overlay"
)
