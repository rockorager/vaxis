package term

import (
	"io"
	"os"
)

type Pty interface {
	io.ReadWriteCloser

	MakeRaw() error
	Restore() error
	// Size reports the Pty's current size
	Size() (Size, error)
	// Notify reports terminal events which can't otherwise be included in
	// the input stream. Currently only size change signals will be sent.
	// The provided channel should have a buffer of at least 1: signals will
	// be dropped if they cannot immediately be sent to the channel
	Notify(chan os.Signal)
}

// OpenPty opens a handle to the controlling terminal's pty
func OpenPty() (Pty, error) {
	return openPty()
}

type Size struct {
	Row    int
	Col    int
	XPixel int
	YPixel int
}
