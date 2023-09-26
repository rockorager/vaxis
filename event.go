package vaxis

// Event is an empty interface used to pass data within a Vaxis application.
// Vaxis will emit user input events as well as other input-related events.
// Users can use PostEvent to post their own events into the loop
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

// PasteStartEvent is sent at the beginning of a bracketed paste. Each [Key]
// within the paste will also have the EventPaste set as the EventType
type PasteStartEvent struct{}

// PasteEndEvent is sent at the end of a bracketed paste. Each [Key]
// within the paste will also have the EventPaste set as the EventType
type PasteEndEvent struct{}

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

// QuitEvent is sent when the application is closing. It is emitted when the
// application calls vaxis.Close, and often times won't be seen by the
// application.
type QuitEvent struct{}
