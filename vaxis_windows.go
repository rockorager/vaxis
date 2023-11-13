//go:build windows

package vaxis

import (
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/vaxis/log"
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
		if atomicLoad(&vx.resize) {
			continue
		}
		ws, err := vx.reportWinsize()
		if err != nil {
			log.Error("couldn't report winsize", "error", err)
			return
		}
		if ws.Cols != vx.winSize.Cols || ws.Rows != vx.winSize.Rows {
			atomicStore(&vx.resize, true)
			vx.PostEvent(Redraw{})
		}
	}
}

func (vx *Vaxis) reportWinsize() (Resize, error) {
	if vx.caps.reportSizeChars && vx.caps.reportSizePixels {
		log.Trace("requesting screen size from terminal")
		io.WriteString(vx.console, textAreaSize)
		deadline := time.NewTimer(100 * time.Millisecond)
		select {
		case <-deadline.C:
			return Resize{}, fmt.Errorf("screen size request deadline exceeded")
		case <-vx.chSizeDone:
			return vx.nextSize, nil
		}
	}
	log.Trace("requesting screen size from console")
	ws, err := vx.console.Size()
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols: int(ws.Width),
		Rows: int(ws.Height),
	}, nil
}
