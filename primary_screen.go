package vaxis

import (
	"fmt"
	"strings"
	"time"
)

// PrimaryScreenMode selects the normal-buffer presentation strategy.
type PrimaryScreenMode uint8

const (
	PrimaryInline PrimaryScreenMode = iota
	PrimaryMain
	PrimarySplit
)

// PrimaryScreenOrigin returns the zero-based physical row of the root Window.
// It is valid only after a successful RenderFrame, until resize or suspension.
// Mouse events retain terminal coordinates. Only translate an event with the
// matching SurfaceGeneration; queued events from an older placement must not
// target the new surface. Retain outside releases for an existing drag.
// Serialize event dispatch and rendering, as with other Window operations.
func (vx *Vaxis) PrimaryScreenOrigin() (row int, generation uint64, valid bool) {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	if vx.primaryScreen == nil {
		return 0, 0, false
	}
	p := vx.primaryScreen
	return p.origin, p.generation, p.positioned && p.rendered
}

// RenderFrame is the error-reporting rendering entry point. In primary-screen
// mode it establishes a physical surface using CPR before painting. Unlike the
// legacy inline Render path, it handles initial partial lines and uses the normal
// cell/graphics renderer. Positioning failures leave preceding output untouched.
// Calls must be serialized with Resize, Suspend, Resume and Close.
func (vx *Vaxis) RenderFrame() error {
	if vx.closed || vx.suspended {
		return fmt.Errorf("terminal is not active")
	}
	if vx.primaryScreen == nil {
		if vx.renderSuppressed() {
			return nil
		}
		start := time.Now()
		vx.render()
		if _, err := vx.tw.Flush(); err != nil {
			vx.refresh = true
			return err
		}
		vx.cursorLast = vx.cursorNext
		vx.refresh = false
		vx.elapsed += time.Since(start)
		vx.renders++
		return nil
	}
	if vx.renderSuppressed() {
		return nil
	}
	start := time.Now()
	vx.mu.Lock()
	p := vx.primaryScreen
	if !p.rendered {
		// A failed first query must not make a later Render fall back to the
		// legacy path that assumes ownership of the current cursor line.
		p.checked = true
	}
	width, height := vx.surfaceSize(vx.winSize)
	if width <= 0 || height <= 0 {
		vx.mu.Unlock()
		return nil
	}
	positioned := p.positioned
	vx.mu.Unlock()
	if !positioned {
		if err := vx.positionPrimary(); err != nil {
			return err
		}
	}
	if err := vx.appendPrimary(); err != nil {
		return err
	}
	vx.render()
	// Save the region start separately from the input cursor. On reflow the
	// terminal moves this anchor with the preceding shell output.
	vx.tw.writeCUP(1, 1)
	_, _ = vx.tw.WriteString(sgrReset + "\x1b7")
	vx.mu.Lock()
	p.checked = true
	p.rendered = true
	p.visualRows = height
	p.resized = false
	vx.mu.Unlock()
	_, err := vx.tw.Flush()
	if err != nil {
		vx.refresh = true
		return err
	}
	vx.cursorLast = vx.cursorNext
	vx.refresh = false
	vx.elapsed += time.Since(start)
	vx.renders++
	return nil
}

// positionPrimary preserves the previous surface while querying its location.
// Only after a successful query can we erase or scroll owned rows.
func (vx *Vaxis) positionPrimary() error {
	p := vx.primaryScreen
	if p.rendered {
		move := "\x1b8\r"
		if !p.checked && p.visualRows > 1 {
			move += tparm("\x1b[%dA", p.visualRows-1)
		}
		if _, err := vx.tw.WriteControlString(move); err != nil {
			return err
		}
	}
	row, col := vx.CursorPosition()
	if row < 0 || row >= vx.winSize.Rows || col < 0 || col >= vx.winSize.Cols {
		return fmt.Errorf("cannot establish primary-screen origin")
	}
	var out strings.Builder
	if p.rendered {
		for y := row; y < min(row+p.visualRows, vx.winSize.Rows); y++ {
			out.WriteString(tparm(cup, y+1, 1))
			out.WriteString("\x1b[2K")
		}
		out.WriteString(tparm(cup, row+1, 1))
	} else if col != 0 {
		out.WriteString("\r\n")
		row = min(row+1, vx.winSize.Rows-1)
	}
	_, height := vx.surfaceSize(vx.winSize)
	if overflow := row + height - vx.winSize.Rows; overflow > 0 {
		out.WriteString(tparm(cup, vx.winSize.Rows, 1))
		out.WriteString(strings.Repeat("\r\n", overflow))
		row -= overflow
	}
	out.WriteString(tparm(cup, row+1, 1))
	if _, err := vx.tw.WriteControlString(out.String()); err != nil {
		return err
	}
	vx.mu.Lock()
	p.origin = row
	p.generation++
	p.positioned = true
	vx.refresh = true
	vx.mu.Unlock()
	return nil
}

func (vx *Vaxis) appendPrimary() error {
	vx.mu.Lock()
	p := vx.primaryScreen
	pending := p.append
	p.append = nil
	vx.mu.Unlock()
	if len(pending) == 0 {
		return nil
	}
	var text strings.Builder
	for _, block := range pending {
		text.WriteString(block)
	}
	output := primaryAppendString(text.String())
	if output == "" {
		return nil
	}
	// A pinned footer scrolls only its output area. Keep the last output line
	// directly above the footer instead of leaving a trailing empty row.
	_, height := vx.surfaceSize(vx.winSize)
	if p.mode == PrimarySplit && p.origin == vx.winSize.Rows-height && p.origin > 1 {
		out := tparm("\x1b[1;%dr", p.origin) + tparm(cup, p.origin, 1)
		out += "\r\n" + strings.TrimSuffix(output, "\r\n")
		out += "\x1b[r"
		_, err := vx.tw.WriteControlString(out)
		return err
	}
	// Until pinned (and for inline/full-height surfaces), replace the live
	// block with output and reserve the following surface. This also handles
	// a footer as tall as the terminal without creating invalid margins.
	var out strings.Builder
	for y := p.origin; y < p.origin+height; y++ {
		out.WriteString(tparm(cup, y+1, 1))
		out.WriteString("\x1b[2K")
	}
	out.WriteString(tparm(cup, p.origin+1, 1))
	out.WriteString(output)
	if !strings.HasSuffix(output, "\n") {
		out.WriteString("\r\n")
	}
	vx.mu.Lock()
	p.positioned = false
	p.rendered = false
	vx.mu.Unlock()
	if _, err := vx.tw.WriteControlString(out.String()); err != nil {
		return err
	}
	return vx.positionPrimary()
}
