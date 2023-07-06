//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package vaxis

import (
	"golang.org/x/sys/unix"
)

// reportWinsize posts a Resize Msg
func reportWinsize() {
	ws, err := unix.IoctlGetWinsize(int(stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		log.Error("couldn't get winsize", "error", err)
		return
	}
	winsize = Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}
	PostMsg(winsize)
}
