package term

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
