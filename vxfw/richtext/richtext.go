package richtext

import (
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"github.com/rivo/uniseg"
)

type RichText struct {
	// The content of the Text widget
	Content []vaxis.Segment

	// Whether to softwrap the text or not
	Softwrap bool
}

func New(segments []vaxis.Segment) *RichText {
	return &RichText{
		Content:  segments,
		Softwrap: true,
	}
}

// Noop for text
func (t *RichText) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (t *RichText) cells(ctx vxfw.DrawContext) []vaxis.Cell {
	cells := []vaxis.Cell{}
	for _, seg := range t.Content {
		for _, char := range ctx.Characters(seg.Text) {
			cell := vaxis.Cell{
				Character: char,
				Style:     seg.Style,
			}
			cells = append(cells, cell)
		}
	}
	return cells
}

func (t *RichText) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if t.Softwrap {
		return t.drawSoftwrap(ctx)
	}

	cells := t.cells(ctx)
	size := t.findContainerSize(cells, ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := NewHardwrapScanner(cells)
	var row uint16
	for scanner.Scan() {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}

		chars := scanner.Line()
	cols:
		for i, char := range chars {
			if col >= ctx.Max.Width {
				break
			}

			// If this char would get us to or beyond the max width
			// (and we aren't the last char), then we print an
			// ellipse
			if col+uint16(char.Width) >= ctx.Max.Width &&
				i < len(chars) {
				cell := vaxis.Cell{
					Character: vaxis.Character{
						Grapheme: "…",
						Width:    1,
					},
					Style: char.Style,
				}
				s.WriteCell(col, row, cell)

				break cols
			} else {
				s.WriteCell(col, row, char)
				col += uint16(char.Width)
			}
		}
		row += 1
	}
	return s, nil
}

func (t *RichText) drawSoftwrap(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	cells := t.cells(ctx)
	size := t.findContainerSize(cells, ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := NewSoftwrapScanner(cells, ctx.Max.Width)
	var row uint16
	for scanner.Scan() {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}

		chars := scanner.Text()
		for _, char := range chars {
			// We should never get here because we softwrapped, but
			// we check just in case
			if col >= ctx.Max.Width {
				break
			}

			s.WriteCell(col, row, char)
			col += uint16(char.Width)
		}
		row += 1
	}
	return s, nil
}

func (t *RichText) findContainerSize(cells []vaxis.Cell, ctx vxfw.DrawContext) vxfw.Size {
	var size vxfw.Size
	if t.Softwrap {
		scanner := NewSoftwrapScanner(cells, ctx.Max.Width)
		for scanner.Scan() {
			if size.Height > ctx.Max.Height {
				return size
			}
			size.Height += 1
			chars := scanner.Text()
			var w uint16
			for _, char := range chars {
				w += uint16(char.Width)
			}
			// Size is limited to the Max.Width
			if size.Width < w {
				size.Width = w
			}
			if size.Width > ctx.Max.Width {
				size.Width = ctx.Max.Width
			}
		}
		return size
	}

	scanner := NewHardwrapScanner(cells)
	for scanner.Scan() {
		if size.Height > ctx.Max.Height {
			return size
		}
		size.Height += 1
		chars := scanner.Line()
		var w uint16
		for _, char := range chars {
			w += uint16(char.Width)
		}
		// Size is limited to the Max.Width
		if size.Width < w {
			size.Width = w
		}
		if size.Width > ctx.Max.Width {
			size.Width = ctx.Max.Width
		}
	}

	return size
}

type SoftwrapScanner struct {
	rest  []vaxis.Cell
	token []vaxis.Cell
	width uint16
}

func NewSoftwrapScanner(s []vaxis.Cell, width uint16) SoftwrapScanner {
	return SoftwrapScanner{
		rest:  s,
		width: width,
	}
}

// Returns the first line segment in s.cells
func firstLineSegment(cells []vaxis.Cell) ([]vaxis.Cell, bool) {
	var rest string

	for i, cell := range cells {
		if i == len(cells)-1 {
			// last one
			return cells, true
		}

		next := cells[i+1]

		// Check if we have a leading line break. We only need to do
		// this on the first iteration
		if i == 0 && uniseg.HasTrailingLineBreakInString(cell.Grapheme) {
			return cells[:i+1], true
		}

		// Check if the next grapheme has a trailing line break. We do
		// this here because we *always* have to check for it anyways,
		// and we can do so before the FirstLineSegment call
		if uniseg.HasTrailingLineBreakInString(next.Grapheme) {
			return cells[:i+2], true
		}

		_, rest, _, _ = uniseg.FirstLineSegmentInString(
			cell.Grapheme+next.Grapheme,
			-1,
		)
		if len(rest) > 0 {
			return cells[:i+1], false
		}
	}
	return cells, false
}

func (s *SoftwrapScanner) Scan() bool {
	if len(s.rest) == 0 || s.width == 0 {
		return false
	}
	// Clear token
	s.token = []vaxis.Cell{}

	var w uint16
	for {
		seg, br := firstLineSegment(s.rest)
		rest := []vaxis.Cell{}
		if len(seg) < len(s.rest) {
			rest = s.rest[len(seg):]
		}

		var (
			word     []vaxis.Cell
			wordLen  uint16
			spaceLen uint16
		)

		// "TrimRight"
		for i := len(seg) - 1; i >= 0; i -= 1 {
			cell := seg[i]
			r, _ := utf8.DecodeLastRuneInString(cell.Grapheme)
			if unicode.IsSpace(r) {
				continue
			}
			// First non-space char. Set our word here
			word = seg[:i+1]
			break
		}

		// Trailing space is anything after word
		trSpace := seg[len(word):]
		for _, ch := range word {
			wordLen += uint16(ch.Width)
		}
		for _, ch := range trSpace {
			spaceLen += uint16(ch.Width)
		}

		// This word is longer than the line. We have to break on
		// graphemes
		if wordLen > s.width {
			s.rest = []vaxis.Cell{}
			// Append characters to token until we reach the end
			for _, char := range word {
				if w >= s.width {
					// Append the rest to rest
					s.rest = append(s.rest, char)
					continue
				}
				s.token = append(s.token, char)
				w += uint16(char.Width)
			}
			// Append the trailing space
			s.rest = append(s.rest, trSpace...)
			// Append the rest...
			s.rest = append(s.rest, rest...)
			return true
		}

		// Check if this segment fits. If it doesn't we are done
		if w+wordLen > s.width {
			return true
		}

		s.rest = rest

		// Check if this segment contains a hard break. If it does, we
		// remove the hard break before adding it to token and then
		// return
		if br {
			last := seg[len(seg)-1]
			if uniseg.HasTrailingLineBreakInString(last.Grapheme) {
				seg = seg[:len(seg)-1]
			}
			s.token = append(s.token, seg...)
			return true
		}

		// Otherwise, add this word
		s.token = append(s.token, word...)
		w += wordLen

		// If the space doesn't fit, we return now
		if w+spaceLen > s.width {
			return true
		}

		s.token = append(s.token, trSpace...)
		w += spaceLen
	}
}

func (s *SoftwrapScanner) Text() []vaxis.Cell {
	return s.token
}

type HardwrapScanner struct {
	cells []vaxis.Cell

	line []vaxis.Cell
}

func NewHardwrapScanner(cells []vaxis.Cell) HardwrapScanner {
	return HardwrapScanner{
		cells: cells,
	}
}

func (h *HardwrapScanner) Scan() bool {
	if len(h.cells) == 0 {
		return false
	}
	h.line = []vaxis.Cell{}
	// Iterate through cells until we find a linebreak
	for i, cell := range h.cells {
		if cell.Grapheme == "\n" {
			if i == len(h.cells)-1 {
				break
			}
			h.cells = h.cells[i+1:]
			return true
		}
		h.line = append(h.line, cell)
	}

	h.cells = []vaxis.Cell{}
	return true
}

func (h *HardwrapScanner) Line() []vaxis.Cell {
	return h.line
}

// Verify we meet the Widget interface
var _ vxfw.Widget = &RichText{}
