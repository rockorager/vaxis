package vaxis

import (
	"os/signal"
	"syscall"
)

func (vx *Vaxis) setupSignals() {
	// TODO: set up a winsize loop to send window change signals via polling
	signal.Notify(vx.chSigKill,
		// kill signals
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGTERM,
	)
}

// TODO: implement pixel size reporting. Need to get this from the terminal
func (vx *Vaxis) reportWinsize() (Resize, error) {
	ws, err := vx.console.Size()
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols: int(ws.Width),
		Rows: int(ws.Height),
	}, nil
}
