package term

import "go.rockorager.dev/vaxis"

// EventBell is emitted when BEL is received
type EventBell struct{}

type EventPanic error

type EventClosed struct {
	Term  *Model
	Error error
}

type EventTitle string

type EventNotify struct {
	Title string
	Body  string
}

type EventWorkingDirectory struct {
	URL string
}

type EventMouseShape struct {
	Shape vaxis.MouseShape
}

type ProgressState int

const (
	ProgressRemove ProgressState = iota
	ProgressSet
	ProgressError
	ProgressIndeterminate
	ProgressPause
)

type EventProgress struct {
	State       ProgressState
	Progress    int
	HasProgress bool
}

// EventAPC is emitted when an APC sequence is received in the terminal
type EventAPC struct {
	Payload string
}
