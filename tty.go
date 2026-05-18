package vaxis

import "io"

// Console is the interface required to provide a custom terminal transport to
// Vaxis. It is intentionally small so callers can adapt their own console or
// PTY implementation without depending on a specific third-party package.
type Console interface {
	io.Reader
	io.Writer
	io.Closer
	Fd() uintptr
	SetRaw() error
	Reset() error
	Size() (Resize, error)
}

type tty interface {
	io.Reader
	io.Writer
	Fd() uintptr
	SetRaw() error
	Reset() error
	Size() (Resize, error)
	StartInput(*Vaxis) error
	StopInput() error
	Close() error
}

type consoleTTY struct {
	Console
}

func (t consoleTTY) StartInput(*Vaxis) error {
	return nil
}

func (t consoleTTY) StopInput() error {
	return nil
}
