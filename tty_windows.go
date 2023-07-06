//go:build windows
// +build windows

package vaxis

// reportWinsize posts a Resize Msg
func reportWinsize() {
	ws, err := con.Size()
	if err != nil {
		log.Error("couldn't get winsize", "error", err)
		return
	}
	winsize = Resize{
		Cols: int(ws.Width),
		Rows: int(ws.Height),
	}
	PostMsg(winsize)
}
}
