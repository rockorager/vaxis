//go:build darwin || freebsd || linux || netbsd || openbsd || zos

package vaxis

import (
	"os/signal"
	"syscall"

	"git.sr.ht/~rockorager/vaxis/log"
	"golang.org/x/sys/unix"
)

func (vx *Vaxis) setupSignals() {
	signal.Notify(vx.chSigWinSz,
		syscall.SIGWINCH,
	)
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
	// if vx.caps.reportSizeChars && vx.caps.reportSizePixels {
	// 	log.Trace("requesting screen size from terminal")
	// 	io.WriteString(vx.console, textAreaSize)
	// 	deadline := time.NewTimer(100 * time.Millisecond)
	// 	select {
	// 	case <-deadline.C:
	// 		return Resize{}, fmt.Errorf("screen size request deadline exceeded")
	// 	case <-vx.chSizeDone:
	// 		return vx.nextSize, nil
	// 	}
	// }
	log.Trace("requesting screen size from ioctl")
	ws, err := unix.IoctlGetWinsize(int(vx.console.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}, nil
}
