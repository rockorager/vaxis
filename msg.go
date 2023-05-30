package rtk

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

// Init will always be the first Msg delivered
type Init struct{}

// Quit is delivered whenever the application is about to close
type Quit struct{}

// Resize is delivered whenever a window size change is detected (likely via
// SIGWINCH)
type Resize struct {
	Cols int
	Rows int
}

// Paste is delivered when a bracketed paste was detected. The value of
// Paste if the pasted content
type Paste string

type sendMsg struct {
	msg   Msg
	model Model
}
