package text

import (
	"bufio"
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"github.com/rivo/uniseg"
)

type Text struct {
	// The content of the Text widget
	Content string

	// The style to draw the text as
	Style vaxis.Style

	// Whether to softwrap the text or not
	Softwrap bool
}

func New(content string) *Text {
	return &Text{
		Content:  content,
		Softwrap: true,
	}
}

// Noop for text
func (t *Text) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (t *Text) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if t.Softwrap {
		return t.drawSoftwrap(ctx)
	}

	size := t.findContainerSize(ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := bufio.NewScanner(strings.NewReader(t.Content))
	var row uint16
	for scanner.Scan() {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}
		chars := ctx.Characters(scanner.Text())
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
					Style: t.Style,
				}
				s.WriteCell(col, row, cell)

				break cols
			} else {
				cell := vaxis.Cell{
					Character: char,
					Style:     t.Style,
				}
				s.WriteCell(col, row, cell)
				col += uint16(char.Width)
			}
		}
		row += 1
	}
	return s, nil
}

func (t *Text) drawSoftwrap(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	size := t.findContainerSize(ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := NewSoftwrapScanner(t.Content, ctx.Max.Width)
	var row uint16
	for scanner.Scan(ctx) {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}
		chars := ctx.Characters(scanner.Text())
		for _, char := range chars {
			// We should never get here because we softwrapped, but
			// we check just in case
			if col >= ctx.Max.Width {
				break
			}

			cell := vaxis.Cell{
				Character: char,
				Style:     t.Style,
			}
			s.WriteCell(col, row, cell)
			col += uint16(char.Width)
		}
		row += 1
	}
	return s, nil
}

func (t *Text) findContainerSize(ctx vxfw.DrawContext) vxfw.Size {
	var size vxfw.Size
	if t.Softwrap {
		scanner := NewSoftwrapScanner(t.Content, ctx.Max.Width)
		for scanner.Scan(ctx) {
			if size.Height > ctx.Max.Height {
				return size
			}
			size.Height += 1
			chars := ctx.Characters(scanner.Text())
			var w uint16
			for _, char := range chars {
				w += uint16(char.Width)
			}
			// Size is limited to the Max.Width
			size.Width = min(ctx.Max.Width, max(w, size.Width))
		}
		return size
	}
	scanner := bufio.NewScanner(strings.NewReader(t.Content))
	for scanner.Scan() {
		if size.Height > ctx.Max.Height {
			return size
		}
		size.Height += 1
		chars := ctx.Characters(scanner.Text())
		var w uint16
		for _, char := range chars {
			w += uint16(char.Width)
		}
		// Size is limited to the Max.Width
		size.Width = min(ctx.Max.Width, max(w, size.Width))
	}

	return size
}

type SoftwrapScanner struct {
	state int
	rest  []byte
	token []byte
	width uint16
}

func NewSoftwrapScanner(s string, width uint16) SoftwrapScanner {
	return SoftwrapScanner{
		state: -1,
		rest:  []byte(s),
		width: width,
	}
}

func (s *SoftwrapScanner) Scan(ctx vxfw.DrawContext) bool {
	if len(s.rest) == 0 || s.width == 0 {
		return false
	}
	// Clear token
	s.token = []byte{}

	var w uint16
	for {
		seg, rest, br, state := uniseg.FirstLineSegment(s.rest, s.state)

		// trim trailing whitespace to get our word
		word := bytes.TrimRightFunc(seg, unicode.IsSpace)
		// trailing space
		trSpace := seg[len(word):]

		wordChars := ctx.Characters(string(word))
		var wordLen uint16
		for _, char := range wordChars {
			wordLen += uint16(char.Width)
		}

		spaceChars := ctx.Characters(string(trSpace))
		var spaceLen uint16
		for _, char := range spaceChars {
			spaceLen += uint16(char.Width)
		}

		// This word is longer than the line. We have to break on
		// graphemes
		if wordLen > s.width {
			s.rest = []byte{}
			// Append characters to token until we reach the end
			for _, char := range wordChars {
				if w >= s.width {
					// Append the rest to rest
					s.rest = append(s.rest, []byte(char.Grapheme)...)
					continue
				}
				s.token = append(s.token, []byte(char.Grapheme)...)
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
		s.state = state

		// Check if this segment contains a hard break. If it does, we
		// remove the hard break before adding it to token and then
		// return
		if br {
			if uniseg.HasTrailingLineBreak(seg) {
				_, l := utf8.DecodeLastRune(seg)
				// trim the trailing rune
				seg = seg[:len(seg)-l]
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

func (s *SoftwrapScanner) Text() string {
	return string(s.token)
}

// Verify we meet the Widget interface
var _ vxfw.Widget = &Text{}
