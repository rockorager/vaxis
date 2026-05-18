package vaxis

import "io"

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
