package vaxis

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

func PostMsg(msg Msg) {
	msgs.push(msg)
}

type FuncMsg struct {
	Func func()
}

// InitMsg will always be the first Msg delivered
type InitMsg struct{}

// QuitMsg is delivered whenever the application is about to close
type QuitMsg struct{}

// Resize is delivered whenever a window size change is detected (likely via
// SIGWINCH)
type Resize struct {
	Cols int
	Rows int
}

// PasteMsg is delivered when a bracketed paste was detected. The value of
// PasteMsg if the pasted content
type PasteMsg string

// SendMsg sends a message to a given model from the main thread
type SendMsg struct {
	Msg   Msg
	Model Model
}

// DrawModelMsg draws the provided model with the provided window. It doesn't call
// draw on the primary model.
type DrawModelMsg struct {
	Model  Model
	Window Window
}
