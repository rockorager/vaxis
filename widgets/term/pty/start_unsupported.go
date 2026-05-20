//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd

package pty

import (
	"os"
	"os/exec"
)

// StartWithSize returns ErrUnsupported on platforms without local PTY support.
func StartWithSize(*exec.Cmd, *Winsize) (*os.File, error) {
	return nil, ErrUnsupported
}
