//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos

package term

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// pty is a unix pseudo-terminal
type pty struct {
	state *term.State
	fd    int
}

func openPty() (Pty, error) {
	fd, err := syscall.Open("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	pty := &pty{
		fd: fd,
	}
	return pty, nil
}

func (p *pty) Read(b []byte) (n int, err error) {
	return syscall.Read(p.fd, b)
}

func (p *pty) Write(b []byte) (n int, err error) {
	return syscall.Write(p.fd, b)
}

func (p *pty) Close() error {
	return syscall.Close(p.fd)
}

func (p *pty) MakeRaw() error {
	termios, err := term.MakeRaw(p.fd)
	if err != nil {
		return err
	}
	p.state = termios
	return nil
}

func (p *pty) Restore() error {
	return term.Restore(p.fd, p.state)
}

func (p *pty) Size() (Size, error) {
	ws, err := unix.IoctlGetWinsize(p.fd, unix.TIOCGWINSZ)
	if err != nil {
		return Size{}, err
	}
	return Size{
		Row:    int(ws.Row),
		Col:    int(ws.Col),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}, nil
}

func (p *pty) Notify(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
