//go:build darwin || freebsd || linux || netbsd || openbsd || zos
// +build darwin freebsd linux netbsd openbsd zos

package vaxis

import (
	"os/signal"
	"syscall"

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
