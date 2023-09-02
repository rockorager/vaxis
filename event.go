package vaxis

type Event interface{}

type (
	primaryDeviceAttribute struct{}
	capabilitySixel        struct{}
	synchronizedUpdates    struct{}
	kittyKeyboard          struct{}
	kittyGraphics          struct{}
	styledUnderlines       struct{}
	truecolor              struct{}
	unicodeSupport         struct{}
)

// Resize is delivered whenever a window size change is detected (likely via
// SIGWINCH)
type Resize struct {
	Cols   int
	Rows   int
	XPixel int
	YPixel int
}

// PasteEvent is delivered when a bracketed paste was detected. The value of
// PasteEvent if the pasted content
type PasteEvent string

// FocusIn is sent when the terminal has gained focus
type FocusIn struct{}

// FocusOut is sent when the terminal has lost focus
type FocusOut struct{}

// Redraw is a generic event which can be sent to the host application to tell
// it some update has occurred it may not know about otherwise and it must
// redraw. These are always issued after a SyncFunc has been called
type Redraw struct{}

// SyncFunc is a function which will be called in the main thread. vaxis will
// call the function and send an empty SyncFunc event to the application to
// signal that something has been updated (probably the application needs to
// redraw itself)
type syncFunc func()