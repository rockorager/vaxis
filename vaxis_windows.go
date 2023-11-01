package vaxis

import (
	"os/signal"
	"syscall"
	"time"
)

func (vx *Vaxis) setupSignals() {
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
	// TODO: Use ReadConsoleInput for events??
	go vx.winch()
}

func (vx *Vaxis) winch() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		<-ticker.C
		ws, err := vx.reportWinsize()
		if err != nil {
			log.Error("couldn't report winsize", "error", err)
			return
		}
		if ws.Cols != vx.winSize.Cols || ws.Rows != vx.winSize.Rows {
			vx.PostEvent(ws)
		}
	}
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
