package richtext

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rockorager/go-uucode"
	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/vxfw"
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

type styleSpan struct {
	start int
	end   int
	style vaxis.Style
}

type richTextLayout struct {
	text  string
	spans []styleSpan
}

func richTextLayoutFromSegments(segments []vaxis.Segment) richTextLayout {
	var b strings.Builder
	var spans []styleSpan
	for _, seg := range segments {
		start := b.Len()
		for _, char := range vaxis.Characters(seg.Text) {
			b.WriteString(char.Grapheme)
		}
		end := b.Len()
		if end == start {
			continue
		}
		if len(spans) > 0 && spans[len(spans)-1].end == start && spans[len(spans)-1].style == seg.Style {
			spans[len(spans)-1].end = end
			continue
		}
		spans = append(spans, styleSpan{
			start: start,
			end:   end,
			style: seg.Style,
		})
	}
	return richTextLayout{text: b.String(), spans: spans}
}

func characters(ctx vxfw.DrawContext, s string) []vaxis.Character {
	if ctx.Characters != nil {
		return ctx.Characters(s)
	}
	return vaxis.Characters(s)
}

func (l richTextLayout) styleAt(offset int) vaxis.Style {
	i := sort.Search(len(l.spans), func(i int) bool {
		return l.spans[i].end > offset
	})
	if i < len(l.spans) && l.spans[i].start <= offset {
		return l.spans[i].style
	}
	return vaxis.Style{}
}

func (l richTextLayout) cells(ctx vxfw.DrawContext, start, end int) []vaxis.Cell {
	if start < 0 {
		start = 0
	}
	if end > len(l.text) {
		end = len(l.text)
	}
	if start >= end {
		return nil
	}

	text := l.text[start:end]
	chars := characters(ctx, text)
	cells := make([]vaxis.Cell, 0, len(chars))
	it := uucode.NewGraphemeIterator(text)
	for i := 0; i < len(chars); i++ {
		g, ok := it.Next()
		if !ok {
			break
		}
		cells = append(cells, vaxis.Cell{
			Character: chars[i],
			Style:     l.styleAt(start + g.Start),
		})
	}
	return cells
}

func (l richTextLayout) width(ctx vxfw.DrawContext, start, end int) int {
	width := 0
	for _, char := range characters(ctx, l.text[start:end]) {
		width += char.Width
	}
	return width
}

func (l richTextLayout) firstGraphemeEnd(start, end int) int {
	it := uucode.NewGraphemeIterator(l.text[start:end])
	g, ok := it.Next()
	if !ok {
		return start
	}
	return start + g.End
}

func (l richTextLayout) breakByGrapheme(ctx vxfw.DrawContext, start, end, availableWidth int) int {
	breakAt := start
	width := 0
	it := uucode.NewGraphemeIterator(l.text[start:end])
	for g, ok := it.Next(); ok; g, ok = it.Next() {
		grapheme := l.text[start+g.Start : start+g.End]
		graphemeWidth := 0
		for _, char := range characters(ctx, grapheme) {
			graphemeWidth += char.Width
		}
		if width+graphemeWidth > availableWidth {
			break
		}
		width += graphemeWidth
		breakAt = start + g.End
	}
	return breakAt
}

func (t *RichText) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	layout := richTextLayoutFromSegments(t.Content)
	if t.Softwrap {
		return t.drawSoftwrap(ctx, layout)
	}

	size := t.findContainerSize(layout, ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := newHardwrapScanner(layout)
	var row uint16
	for scanner.Scan() {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}

		chars := scanner.Line(ctx)
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

func (t *RichText) drawSoftwrap(ctx vxfw.DrawContext, layout richTextLayout) (vxfw.Surface, error) {
	size := t.findContainerSize(layout, ctx)
	s := vxfw.NewSurface(size.Width, size.Height, t)

	scanner := newSoftwrapScanner(layout, ctx.Max.Width)
	var row uint16
	for scanner.Scan(ctx) {
		var col uint16
		if row > ctx.Max.Height {
			return s, nil
		}

		chars := scanner.Text(ctx)
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

func (t *RichText) findContainerSize(layout richTextLayout, ctx vxfw.DrawContext) vxfw.Size {
	var size vxfw.Size
	if t.Softwrap {
		scanner := newSoftwrapScanner(layout, ctx.Max.Width)
		for scanner.Scan(ctx) {
			if size.Height > ctx.Max.Height {
				return size
			}
			size.Height += 1
			chars := scanner.Text(ctx)
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
		if size.Width < ctx.Min.Width {
			size.Width = ctx.Min.Width
		}
		if size.Height < ctx.Min.Height {
			size.Height = ctx.Min.Height
		}
		return size
	}

	scanner := newHardwrapScanner(layout)
	for scanner.Scan() {
		if size.Height > ctx.Max.Height {
			return size
		}
		size.Height += 1
		chars := scanner.Line(ctx)
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
	if size.Width < ctx.Min.Width {
		size.Width = ctx.Min.Width
	}
	if size.Height < ctx.Min.Height {
		size.Height = ctx.Min.Height
	}

	return size
}

type lineRange struct {
	start int
	end   int
}

type SoftwrapScanner struct {
	layout richTextLayout
	rest   int
	token  lineRange
	width  uint16
}

func newSoftwrapScanner(layout richTextLayout, width uint16) SoftwrapScanner {
	return SoftwrapScanner{
		layout: layout,
		width:  width,
	}
}

func trimRightFuncInRange(s string, start, end int, f func(rune) bool) int {
	for end > start {
		r, size := utf8.DecodeLastRuneInString(s[start:end])
		if r == utf8.RuneError && size == 0 {
			break
		}
		if !f(r) {
			break
		}
		end -= size
	}
	return end
}

func trimTrailingLineBreakInRange(s string, start, end int) int {
	if end <= start {
		return end
	}
	r, size := utf8.DecodeLastRuneInString(s[start:end])
	if r == utf8.RuneError && size == 0 {
		return end
	}
	if uucode.LineBreak(r) != uucode.LineBreakBK &&
		uucode.LineBreak(r) != uucode.LineBreakCR &&
		uucode.LineBreak(r) != uucode.LineBreakLF &&
		uucode.LineBreak(r) != uucode.LineBreakNL {
		return end
	}
	end -= size
	if r == '\n' && end > start {
		prev, prevSize := utf8.DecodeLastRuneInString(s[start:end])
		if prev == '\r' {
			end -= prevSize
		}
	}
	return end
}

func (s *SoftwrapScanner) Scan(ctx vxfw.DrawContext) bool {
	if s.rest >= len(s.layout.text) || s.width == 0 {
		return false
	}

	lineStart := s.rest
	tokenEnd := lineStart
	width := 0
	maxWidth := int(s.width)

	it := uucode.NewLineIterator(s.layout.text[lineStart:])
	for segment, ok := it.Next(); ok; segment, ok = it.Next() {
		segStart := lineStart + segment.Start
		segEnd := lineStart + segment.End
		wordEnd := trimRightFuncInRange(s.layout.text, segStart, segEnd, unicode.IsSpace)
		wordWidth := s.layout.width(ctx, segStart, wordEnd)
		spaceWidth := s.layout.width(ctx, wordEnd, segEnd)

		// This word is longer than the line. We have to break on
		// graphemes.
		if wordWidth > maxWidth {
			breakAt := s.layout.breakByGrapheme(ctx, segStart, wordEnd, maxWidth-width)
			if breakAt == segStart {
				if tokenEnd > lineStart {
					s.token = lineRange{start: lineStart, end: tokenEnd}
					s.rest = segStart
					return true
				}
				breakAt = s.layout.firstGraphemeEnd(segStart, wordEnd)
			}
			s.token = lineRange{start: lineStart, end: breakAt}
			s.rest = breakAt
			return true
		}

		// Check if this segment fits. If it doesn't we are done.
		if width+wordWidth > maxWidth {
			s.token = lineRange{start: lineStart, end: tokenEnd}
			s.rest = segStart
			return true
		}

		if segment.Break == uucode.LineMustBreak {
			s.token = lineRange{
				start: lineStart,
				end:   trimTrailingLineBreakInRange(s.layout.text, lineStart, segEnd),
			}
			s.rest = segEnd
			return true
		}

		tokenEnd = wordEnd
		width += wordWidth

		// If the space doesn't fit, we return now and drop it.
		if width+spaceWidth > maxWidth {
			s.token = lineRange{start: lineStart, end: tokenEnd}
			s.rest = segEnd
			return true
		}

		tokenEnd = segEnd
		width += spaceWidth
	}

	return false
}

func (s *SoftwrapScanner) Text(ctx vxfw.DrawContext) []vaxis.Cell {
	return s.layout.cells(ctx, s.token.start, s.token.end)
}

type HardwrapScanner struct {
	layout richTextLayout
	rest   int
	line   lineRange
}

func newHardwrapScanner(layout richTextLayout) HardwrapScanner {
	return HardwrapScanner{layout: layout}
}

func (h *HardwrapScanner) Scan() bool {
	if h.rest >= len(h.layout.text) {
		return false
	}

	lineStart := h.rest
	it := uucode.NewLineIterator(h.layout.text[lineStart:])
	for segment, ok := it.Next(); ok; segment, ok = it.Next() {
		if segment.Break != uucode.LineMustBreak {
			continue
		}
		segEnd := lineStart + segment.End
		h.line = lineRange{
			start: lineStart,
			end:   trimTrailingLineBreakInRange(h.layout.text, lineStart, segEnd),
		}
		h.rest = segEnd
		return true
	}

	return false
}

func (h *HardwrapScanner) Line(ctx vxfw.DrawContext) []vaxis.Cell {
	return h.layout.cells(ctx, h.line.start, h.line.end)
}

// Verify we meet the Widget interface
var _ vxfw.Widget = &RichText{}
