package term

import (
	"time"
)

// EventTerminal is a generic terminal event
type EventTerminal struct {
	when time.Time
	vt   *Model
}

func newEventTerminal(vt *Model) *EventTerminal {
	return &EventTerminal{
		when: time.Now(),
		vt:   vt,
	}
}

func (ev *EventTerminal) When() time.Time {
	return ev.when
}

func (ev *EventTerminal) VT() *Model {
	return ev.vt
}

// EventTitle is emitted when the terminal's title changes
type EventTitle struct {
	*EventTerminal
	title string
}

func (ev *EventTitle) Title() string {
	return ev.title
}

// EventMouseMode is emitted when the terminal mouse mode changes
// type EventMouseMode struct {
// 	modes []tcell.MouseFlags
//
// 	*EventTerminal
// }
//
// func (ev *EventMouseMode) Flags() []tcell.MouseFlags {
// 	return ev.modes
// }

// EventBell is emitted when BEL is received
type EventBell struct {
	*EventTerminal
}

type EventPanic struct {
	*EventTerminal
	Error error
}
