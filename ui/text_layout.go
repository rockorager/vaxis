package ui

import "unicode/utf8"

type TextLayoutOptions struct {
	SoftWrap bool
	Overflow TextOverflow
	MaxLines int
	Align    TextAlign
}

type TextLayout struct {
	Lines []TextLine
	Size  Size
}

type TextSelectionRange struct {
	Row   int
	Col   int
	Width int
}

type TextCursorCellOptions struct {
	SoftWrap bool
	Width    int
}

type TextLine struct {
	Runs   []TextSpan
	Width  int
	Offset int
	Cells  []TextCell
	Start  TextPosition
	End    TextPosition
}

type TextCell struct {
	Text      string
	Width     int
	Style     Style
	Position  TextPosition
	Synthetic bool
}

func (c TextCell) End() TextPosition {
	return advanceTextPosition(c.Position, c.Text)
}

type TextPosition struct {
	Span           int
	ByteOffset     int
	RuneOffset     int
	GraphemeOffset int
}

type textAtom struct {
	char      Character
	style     Style
	position  TextPosition
	end       TextPosition
	synthetic bool
}

type (
	textLayoutOptions = TextLayoutOptions
)

func LayoutText(spans []TextSpan, c Constraints, opts TextLayoutOptions) TextLayout {
	maxWidth := c.MaxWidth
	if !opts.SoftWrap || maxWidth == Unbounded {
		maxWidth = Unbounded
	}
	pos := TextPosition{}
	var lines []TextLine
	newLine := func(start TextPosition) TextLine {
		return TextLine{Start: start, End: start}
	}
	line := newLine(pos)
	word := []textAtom{}
	wordWidth := 0
	endedWithNewline := false
	flushLine := func(nextStart TextPosition) {
		appendWord(&line, word)
		word = nil
		wordWidth = 0
		lines = append(lines, line)
		line = newLine(nextStart)
	}
	commitLine := func(nextStart TextPosition) {
		lines = append(lines, line)
		line = newLine(nextStart)
	}
	flushWord := func() {
		appendWord(&line, word)
		word = nil
		wordWidth = 0
	}
	for spanIndex, span := range spans {
		pos.Span = spanIndex
		pos.ByteOffset = 0
		pos.RuneOffset = 0
		pos.GraphemeOffset = 0
		for _, ch := range vaxisCharacters(span.Text) {
			start := pos
			end := advanceTextPosition(pos, ch.Grapheme)
			pos = end
			if ch.Grapheme == "\n" {
				flushLine(pos)
				endedWithNewline = true
				continue
			}
			endedWithNewline = false
			atom := textAtom{char: ch, style: span.Style, position: start, end: end}
			if !opts.SoftWrap || maxWidth == Unbounded {
				appendAtom(&line, atom)
				continue
			}
			if isSpace(ch) {
				flushWord()
				if line.Width+ch.Width > maxWidth && line.Width > 0 {
					flushLine(atom.end)
				} else if line.Width+ch.Width <= maxWidth {
					appendAtom(&line, atom)
				}
				continue
			}
			if wordWidth+ch.Width > maxWidth && wordWidth > 0 {
				if line.Width > 0 {
					flushLine(word[0].position)
				}
				appendWord(&line, word)
				word = nil
				wordWidth = 0
				flushLine(atom.position)
			}
			word = append(word, atom)
			wordWidth += ch.Width
			if line.Width+wordWidth > maxWidth && line.Width > 0 {
				commitLine(word[0].position)
			}
		}
	}
	appendWord(&line, word)
	if len(lines) == 0 || line.Width > 0 || len(line.Runs) > 0 || endedWithNewline {
		lines = append(lines, line)
	}
	if opts.MaxLines > 0 && len(lines) > opts.MaxLines {
		lines = lines[:opts.MaxLines]
		if opts.Overflow == TextOverflowEllipsis && len(lines) > 0 {
			applyEllipsis(&lines[len(lines)-1], c.MaxWidth)
		}
	} else if opts.Overflow == TextOverflowEllipsis && opts.MaxLines == 1 && c.HasBoundedWidth() && len(lines) == 1 && lines[0].Width > c.MaxWidth {
		applyEllipsis(&lines[0], c.MaxWidth)
	}
	width := 0
	for _, line := range lines {
		width = max(width, line.Width)
	}
	size := c.Constrain(Size{Width: width, Height: len(lines)})
	for i := range lines {
		lines[i].Offset = textAlignOffset(size.Width, lines[i].Width, opts.Align)
	}
	return TextLayout{Lines: lines, Size: size}
}

func layoutText(spans []TextSpan, c Constraints, opts textLayoutOptions) TextLayout {
	return LayoutText(spans, c, opts)
}

func (l TextLayout) PositionForCell(row, col int) (TextPosition, bool) {
	if row < 0 || row >= len(l.Lines) {
		return TextPosition{}, false
	}
	line := l.Lines[row]
	local := col - line.Offset
	if local < 0 {
		return line.Start, true
	}
	x := 0
	for _, cell := range line.Cells {
		if local >= x && local < x+cell.Width {
			return cell.Position, true
		}
		x += cell.Width
	}
	if local >= line.Width {
		return line.End, true
	}
	return TextPosition{}, false
}

func (l TextLayout) CellForPosition(pos TextPosition) (row, col int, ok bool) {
	for y, line := range l.Lines {
		x := 0
		for _, cell := range line.Cells {
			if sameTextPosition(cell.Position, pos) {
				return y, line.Offset + x, true
			}
			x += cell.Width
		}
		if sameTextPosition(line.End, pos) {
			return y, line.Offset + line.Width, true
		}
	}
	return 0, 0, false
}

func (l TextLayout) CursorCell(pos TextPosition, opts TextCursorCellOptions) (row, col int, ok bool) {
	row, col, ok = l.CellForPosition(pos)
	if !ok {
		return 0, 0, false
	}
	if opts.SoftWrap && opts.Width > 0 && col >= opts.Width {
		return row + 1, 0, true
	}
	return row, col, true
}

func (l TextLayout) SelectionRanges(selection TextSelection) []TextSelectionRange {
	if selection.IsCollapsed() {
		return nil
	}
	var ranges []TextSelectionRange
	for row, line := range l.Lines {
		start := -1
		width := 0
		x := line.Offset
		for _, cell := range line.Cells {
			if selection.IntersectsCell(cell) {
				if start < 0 {
					start = x
				}
				width += cell.Width
			} else if start >= 0 {
				ranges = append(ranges, TextSelectionRange{Row: row, Col: start, Width: width})
				start = -1
				width = 0
			}
			x += cell.Width
		}
		if start >= 0 {
			ranges = append(ranges, TextSelectionRange{Row: row, Col: start, Width: width})
		}
		if len(line.Cells) == 0 && selection.ContainsLineBreak(line) {
			ranges = append(ranges, TextSelectionRange{Row: row, Col: line.Offset, Width: 1})
		}
	}
	return ranges
}

func advanceTextPosition(pos TextPosition, text string) TextPosition {
	pos.ByteOffset += len(text)
	pos.RuneOffset += utf8.RuneCountInString(text)
	pos.GraphemeOffset++
	return pos
}

func sameTextPosition(a, b TextPosition) bool {
	return a.Span == b.Span && a.ByteOffset == b.ByteOffset
}

func compareTextPosition(a, b TextPosition) int {
	if a.Span < b.Span {
		return -1
	}
	if a.Span > b.Span {
		return 1
	}
	if a.ByteOffset < b.ByteOffset {
		return -1
	}
	if a.ByteOffset > b.ByteOffset {
		return 1
	}
	return 0
}

func appendWord(line *TextLine, word []textAtom) {
	for _, atom := range word {
		appendAtom(line, atom)
	}
}

func appendAtom(line *TextLine, atom textAtom) {
	if len(line.Runs) == 0 || line.Runs[len(line.Runs)-1].Style != atom.style {
		line.Runs = append(line.Runs, TextSpan{Style: atom.style})
	}
	line.Runs[len(line.Runs)-1].Text += atom.char.Grapheme
	line.Width += atom.char.Width
	line.Cells = append(line.Cells, TextCell{
		Text:      atom.char.Grapheme,
		Width:     atom.char.Width,
		Style:     atom.style,
		Position:  atom.position,
		Synthetic: atom.synthetic,
	})
	line.End = atom.end
}

func isSpace(ch Character) bool {
	return ch.Grapheme == " " || ch.Grapheme == "\t"
}

func applyEllipsis(line *TextLine, maxWidth int) {
	if maxWidth == Unbounded || maxWidth <= 0 {
		line.Runs = nil
		line.Width = 0
		line.Cells = nil
		return
	}
	atoms := lineAtoms(*line)
	width := 1
	keep := 0
	for keep < len(atoms) && width+atoms[keep].char.Width <= maxWidth {
		width += atoms[keep].char.Width
		keep++
	}
	if keep < len(atoms) {
		atoms = atoms[:keep]
	}
	syntheticPos := line.Start
	if len(atoms) > 0 {
		syntheticPos = atoms[len(atoms)-1].end
	}
	style := Style{}
	if len(atoms) > 0 {
		style = atoms[len(atoms)-1].style
	} else if len(line.Runs) > 0 {
		style = line.Runs[0].Style
	}
	atoms = append(atoms, textAtom{char: Character{Grapheme: "…", Width: 1}, style: style, position: syntheticPos, end: syntheticPos, synthetic: true})
	line.Runs = nil
	line.Width = 0
	line.Cells = nil
	appendWord(line, atoms)
}

func lineAtoms(line TextLine) []textAtom {
	var atoms []textAtom
	for _, cell := range line.Cells {
		atoms = append(atoms, textAtom{
			char:      Character{Grapheme: cell.Text, Width: cell.Width},
			style:     cell.Style,
			position:  cell.Position,
			end:       advanceTextPosition(cell.Position, cell.Text),
			synthetic: cell.Synthetic,
		})
	}
	return atoms
}

func paintLaidOutText(p *Painter, off Offset, layout TextLayout, opts textLayoutOptions) {
	for y, line := range layout.Lines {
		x := off.X + line.Offset
		for _, run := range line.Runs {
			p.DrawText(Offset{X: x, Y: off.Y + y}, run.Text, run.Style)
			x += textWidth(run.Text)
		}
	}
}

func paintTextBackground(p *Painter, off Offset, size Size, spans []TextSpan) {
	style, ok := textBackgroundStyle(spans)
	if !ok || size.Width <= 0 || size.Height <= 0 {
		return
	}
	p.Fill(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, Cell{Character: Character{Grapheme: " ", Width: 1}, Style: style})
}

func textBackgroundStyle(spans []TextSpan) (Style, bool) {
	for _, span := range spans {
		if span.Style.Background != 0 {
			return span.Style, true
		}
	}
	return Style{}, false
}

func textAlignOffset(width, lineWidth int, align TextAlign) int {
	delta := max(0, width-lineWidth)
	switch align {
	case TextAlignEnd, TextAlignRight:
		return delta
	case TextAlignCenter:
		return delta / 2
	default:
		return 0
	}
}
