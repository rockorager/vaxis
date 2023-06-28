package vaxis

// Cmd is some action which shouldn't happen in the main thread. vaxis provides
// this as a helper for any asynchronous requests that should occur as part of
// the normal application. Cmds are typically processed in another
// goroutine/thread
type Cmd interface{}

var cmds = newQueue[Cmd]()

// PostCmd posts a Cmd into the Cmd queue. PostCmd is non-blocking and will
// always accept a Cmd
func PostCmd(cmd Cmd) {
	cmds.push(cmd)
}

// PollCmd blocks until a Cmd can be returned. Nil Cmds will not be returned.
func PollCmd() Cmd {
	var c Cmd
	for cmd := range cmds.Chan() {
		if cmd == nil {
			continue
		}
		c = cmd
		break
	}
	return c
}

func Cmds() chan Cmd {
	return cmds.Chan()
}
