//go:build darwin || freebsd || linux || netbsd || openbsd || zos

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
	vx.pty.Notify(vx.chSigWinSz)
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

// reportWinsize
func (vx *Vaxis) reportWinsize() (Resize, error) {
	if vx.caps.reportSizeChars && vx.caps.reportSizePixels {
		log.Trace("requesting screen size from terminal")
		io.WriteString(vx.pty, textAreaSize)
		deadline := time.NewTimer(100 * time.Millisecond)
		select {
		case <-deadline.C:
			return Resize{}, fmt.Errorf("screen size request deadline exceeded")
		case <-vx.chSizeDone:
			return vx.nextSize, nil
		}
	}
	log.Trace("requesting screen size from ioctl")
	ws, err := vx.pty.Size()
	// ws, err := unix.IoctlGetWinsize(int(vx.pty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.XPixel),
		YPixel: int(ws.YPixel),
	}, nil
}
