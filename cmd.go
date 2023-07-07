package vaxis

// cmds is an asynchronous queue, provided as a helper for applications
var cmds = newQueue[Cmd]()

type Cmd interface{}

func PostCmd(cmd Cmd) {
	if cmd == nil {
		return
	}
	cmds.push(cmd)
}

func PollCmd() Cmd {
	var c Cmd
	for cmd := range cmds.ch {
		c = cmd
		break
	}
	return c
}

func CmdChannel() chan Cmd {
	return cmds.Chan()
}
