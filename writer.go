package vaxis

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// writer is a buffered writer for a terminal. If the terminal supports
// synchronized output, all writes will be wrapped with synchronized mode
// set/reset. The internal buffer will be reset upon flushing
type writer struct {
	buf *bytes.Buffer
	w   io.Writer
	vx  *Vaxis
	mut sync.Mutex
}

func newWriter(vx *Vaxis) *writer {
	return &writer{
		buf: bytes.NewBuffer(make([]byte, 0, 8192)),
		w:   vx.console,
		vx:  vx,
	}
}

func (w *writer) writeCUP(row int, col int) {
	buf := [32]byte{}
	b := buf[:0]
	b = append(b, '\x1b', '[')
	b = strconv.AppendInt(b, int64(row), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(col), 10)
	b = append(b, 'H')
	_, _ = w.Write(b)
}

func (w *writer) writeOSC8(params string, link string) {
	_, _ = w.WriteString("\x1b]8;")
	_, _ = w.WriteString(params)
	_, _ = w.WriteString(";")
	_, _ = w.WriteString(link)
	_, _ = w.WriteString("\x1b\\")
}

func (w *writer) writeExplicitWidth(width int, grapheme string) {
	_, _ = w.WriteString("\x1b]66;w=")
	buf := [24]byte{}
	b := strconv.AppendInt(buf[:0], int64(width), 10)
	_, _ = w.Write(b)
	_, _ = w.WriteString(";")
	_, _ = w.WriteString(grapheme)
	_, _ = w.WriteString("\x1b\\")
}

func (w *writer) writeUnderlineStyle(style UnderlineStyle) {
	buf := [24]byte{}
	b := buf[:0]
	b = append(b, '\x1b', '[', '4', sgrParamSeparator)
	b = strconv.AppendInt(b, int64(style), 10)
	b = append(b, 'm')
	_, _ = w.Write(b)
}

func (w *writer) writeSGRIndexed(param int, val uint8) {
	buf := [32]byte{}
	b := buf[:0]
	b = append(b, '\x1b', '[')
	b = strconv.AppendInt(b, int64(param), 10)
	b = append(b, sgrParamSeparator, '5', sgrParamSeparator)
	b = strconv.AppendInt(b, int64(val), 10)
	b = append(b, 'm')
	_, _ = w.Write(b)
}

func (w *writer) writeSGRRGB(param int, r uint8, g uint8, b2 uint8) {
	buf := [48]byte{}
	b := buf[:0]
	b = append(b, '\x1b', '[')
	b = strconv.AppendInt(b, int64(param), 10)
	b = append(b, sgrParamSeparator, '2', sgrParamSeparator)
	b = strconv.AppendInt(b, int64(r), 10)
	b = append(b, sgrParamSeparator)
	b = strconv.AppendInt(b, int64(g), 10)
	b = append(b, sgrParamSeparator)
	b = strconv.AppendInt(b, int64(b2), 10)
	b = append(b, 'm')
	_, _ = w.Write(b)
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

// WriteStringLocked writes to the underlying terminal while the mutex is held.
// This does not handle any mouse nor synchronization state and is intended to
// be used for one-off synchronized sequence writes to the terminal
func (w *writer) WriteStringLocked(s string) (n int, err error) {
	w.mut.Lock()
	defer w.mut.Unlock()
	return io.WriteString(w.w, s)
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
	w.mut.Lock()
	defer w.mut.Unlock()
	return w.w.Write(w.buf.Bytes())
}
