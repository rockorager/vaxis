package vaxis

// Event is an empty interface used to pass data within a Vaxis application.
// Vaxis will emit user input events as well as other input-related events.
// Users can use PostEvent to post their own events into the loop
type Event interface{}

type (
	primaryDeviceAttribute struct{}
	capabilitySixel        struct{}
	synchronizedUpdates    struct{}
	unicodeCoreCap         struct{}
	kittyKeyboard          struct{}
	kittyGraphics          struct{}
	styledUnderlines       struct{}
	truecolor              struct{}
	notifyColorChange      struct{}
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

// ColorThemeMode is the current color theme of the terminal. The raw value is
// equivalent to the DSR response value for each mode.
type ColorThemeMode int

const (
	// The terminal has a dark color theme
	DarkMode ColorThemeMode = 1
	// The terminal has a light color theme
	LightMode ColorThemeMode = 2
)

// ColorThemeUpdate is sent when the terminal color scheme has changed. This
// event is only delivered if supported by the terminal
type ColorThemeUpdate struct {
	Mode ColorThemeMode
}
