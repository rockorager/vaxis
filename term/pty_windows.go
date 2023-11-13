package term

import (
	"os"
	"syscall"
)

var k32 = syscall.NewLazyDLL("kernel32.dll")

type conPty struct {
	stdin  syscall.Handle
	stdout syscall.Handle
}

func openPty() (Pty, error) {
	stdin, err := syscall.Open("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	stdout, err := syscall.Open("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	pty := &conPty{
		stdin:  stdin,
		stdout: stdout,
	}
	return pty, nil
}

func (pty *conPty) Read(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (pty *conPty) Write(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (pty *conPty) Close() error {
	syscall.Close(pty.stdin)
	syscall.Close(pty.stdout)
	return nil
}

func (pty *conPty) MakeRaw() error {
	panic("not implemented") // TODO: Implement
}

func (pty *conPty) Restore() error {
	panic("not implemented") // TODO: Implement
}

// Size reports the Pty's current size
func (pty *conPty) Size() (term.Size, error) {
	panic("not implemented") // TODO: Implement
}

// Notify reports terminal events which can't otherwise be included in
// the input stream. Currently only size change signals will be sent.
// The provided channel should have a buffer of at least 1: signals will
// be dropped if they cannot immediately be sent to the channel
func (pty *conPty) Notify(_ chan os.Signal) {
	panic("not implemented") // TODO: Implement
}
