package vaxis

import (
	"bytes"
	"io"
	"strconv"
	"sync"
)

type terminalWriter struct {
	w   io.Writer
	mut sync.Mutex
}

func (tw *terminalWriter) WriteRaw(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	tw.mut.Lock()
	defer tw.mut.Unlock()
	return tw.w.Write(p)
}

func (tw *terminalWriter) WriteRawString(s string) (n int, err error) {
	if s == "" {
		return 0, nil
	}
	tw.mut.Lock()
	defer tw.mut.Unlock()
	return io.WriteString(tw.w, s)
}

// writer buffers render-frame output. Render writes get wrapped with cursor and
// synchronized-output state at flush time; control writes bypass the frame buffer
// and go straight to terminalWriter.
type writer struct {
	buf      *bytes.Buffer
	terminal *terminalWriter
	vx       *Vaxis
}

func newWriter(vx *Vaxis) *writer {
	return &writer{
		buf:      bytes.NewBuffer(make([]byte, 0, 8192)),
		terminal: &terminalWriter{w: vx.tty},
		vx:       vx,
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
	switch style {
	default:
		// Encode below.
	case UnderlineOff:
		_, _ = w.WriteString(underlineReset)
		return
	case UnderlineSingle:
		_, _ = w.WriteString(underlineSet)
		return
	}
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

func (w *writer) startFrame() {
	if w.buf.Len() != 0 {
		return
	}
	w.buf.WriteString(hideCursorSeq)
	if w.vx.caps.synchronizedUpdate {
		w.buf.WriteString(syncUpdateStartSeq)
	}
}

func (w *writer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.startFrame()
	return w.buf.Write(p)
}

func (w *writer) WriteString(s string) (n int, err error) {
	if s == "" {
		return 0, nil
	}
	w.startFrame()
	return w.buf.WriteString(s)
}

func (w *writer) Len() int {
	return w.buf.Len()
}

func (w *writer) WriteControl(p []byte) (n int, err error) {
	return w.terminal.WriteRaw(p)
}

func (w *writer) WriteControlString(s string) (n int, err error) {
	return w.terminal.WriteRawString(s)
}

func (w *writer) writeControlCUP(row int, col int) {
	buf := [32]byte{}
	b := buf[:0]
	b = append(b, '\x1b', '[')
	b = strconv.AppendInt(b, int64(row), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(col), 10)
	b = append(b, 'H')
	_, _ = w.WriteControl(b)
}

func (w *writer) writeControlExplicitWidth(width int, grapheme string) {
	buf := bytes.NewBuffer(make([]byte, 0, 16+len(grapheme)))
	buf.WriteString("\x1b]66;w=")
	tmp := [24]byte{}
	buf.Write(strconv.AppendInt(tmp[:0], int64(width), 10))
	buf.WriteByte(';')
	buf.WriteString(grapheme)
	buf.WriteString("\x1b\\")
	_, _ = w.WriteControl(buf.Bytes())
}

func (w *writer) Flush() (n int, err error) {
	if w.buf.Len() == 0 {
		// If we didn't write any visual changes, make sure we make any
		// cursor changes here. These still go through terminalWriter so
		// cursor-only frames serialize with control writes.
		switch {
		case !w.vx.cursorNext.visible && w.vx.cursorLast.visible:
			return w.WriteControlString(hideCursorSeq)
		case w.vx.cursorNext.visible && !w.vx.cursorLast.visible:
			return w.WriteControlString(w.vx.showCursor())
		case w.vx.cursorNext.visible && w.vx.cursorNext.row != w.vx.cursorLast.row:
			return w.WriteControlString(w.vx.showCursor())
		case w.vx.cursorNext.visible && w.vx.cursorNext.col != w.vx.cursorLast.col:
			return w.WriteControlString(w.vx.showCursor())
		case w.vx.cursorNext.visible && w.vx.cursorNext.style != w.vx.cursorLast.style:
			return w.WriteControlString(w.vx.showCursor())
		default:
			return 0, nil
		}
	}
	defer w.buf.Reset()
	w.buf.WriteString(sgrReset)
	if w.vx.cursorNext.visible {
		w.buf.WriteString(w.vx.showCursor())
	}
	if w.vx.caps.synchronizedUpdate {
		w.buf.WriteString(syncUpdateEndSeq)
	}
	return w.WriteControl(w.buf.Bytes())
}
