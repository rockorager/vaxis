package vaxis

import (
	"bytes"
	"fmt"
	"io"
)

// writer is a buffered writer for a terminal. If the terminal supports
// synchronized output, all writes will be wrapped with synchronized mode
// set/reset. The internal buffer will be reset upon flushing
type writer struct {
	buf *bytes.Buffer
	w   io.Writer
	vx  *Vaxis
}

func newWriter(vx *Vaxis) *writer {
	return &writer{
		buf: bytes.NewBuffer(make([]byte, 8192)),
		w:   vx.pty,
		vx:  vx,
	}
}

func (w *writer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if w.buf.Len() == 0 {
		if w.vx.caps.synchronizedUpdate {
			w.buf.WriteString(decset(synchronizedUpdate))
		}
		if w.vx.cursorLast.visible && w.vx.cursorNext.visible {
			// Hide cursor if it's visible, and only write this if
			// the next cursor is visible also. we'll explicitly
			// turn the cursor off in the render loop if there is a
			// change to the state of cursor visibility
			w.buf.WriteString(decrst(cursorVisibility))
		}
	}
	return w.buf.Write(p)
}

func (w *writer) WriteString(s string) (n int, err error) {
	if s == "" {
		return 0, nil
	}
	if w.buf.Len() == 0 {
		if w.vx.cursorLast.visible {
			// Hide cursor if it's visible
			w.buf.WriteString(decrst(cursorVisibility))
		}
		if w.vx.caps.synchronizedUpdate {
			w.buf.WriteString(decset(synchronizedUpdate))
		}
	}
	return w.buf.WriteString(s)
}

func (w *writer) Printf(s string, args ...any) (n int, err error) {
	return fmt.Fprintf(w, s, args...)
}

func (w *writer) Len() int {
	return w.buf.Len()
}

func (w *writer) Flush() (n int, err error) {
	if w.buf.Len() == 0 {
		// If we didn't write any visual changes, make sure we make any
		// cursor changes here. Write directly to tty for these as
		// they are short and don't require synchronization
		switch {
		case !w.vx.cursorNext.visible && w.vx.cursorLast.visible:
			return w.w.Write([]byte(decrst(cursorVisibility)))
		case w.vx.cursorNext.row != w.vx.cursorLast.row:
			return w.w.Write([]byte(w.vx.showCursor()))
		case w.vx.cursorNext.col != w.vx.cursorLast.col:
			return w.w.Write([]byte(w.vx.showCursor()))
		case w.vx.cursorNext.style != w.vx.cursorLast.style:
			return w.w.Write([]byte(w.vx.showCursor()))
		default:
			return 0, nil
		}
	}
	defer w.buf.Reset()
	w.buf.WriteString(sgrReset)
	// We check against both. If the state changed, this was written in the
	// render loop. this portion only restores where teh cursor was prior to
	// the render
	if w.vx.cursorNext.visible && w.vx.cursorLast.visible {
		w.buf.WriteString(w.vx.showCursor())
	}
	if w.vx.caps.synchronizedUpdate {
		w.buf.WriteString(decrst(synchronizedUpdate))
	}
	return w.w.Write(w.buf.Bytes())
}
