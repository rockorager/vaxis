// Package pty provides pseudo-terminal support for the terminal widget.
//
// Portions of this package are adapted from github.com/creack/pty.
package pty

import (
	"errors"
	"os"
)

// ErrUnsupported is returned if a function is not available on the current
// platform.
var ErrUnsupported = errors.New("unsupported")

// Open opens a pty and its corresponding tty.
func Open() (pty, tty *os.File, err error) {
	return open()
}
