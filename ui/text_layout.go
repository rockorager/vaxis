package ui

import "unicode/utf8"

// TextLayoutOptions controls wrapping, overflow, line limits, and alignment.
type TextLayoutOptions struct {
	SoftWrap bool
	Overflow TextOverflow
	MaxLines int
	Align    TextAlign
}

// TextLayout is the measured line and cell representation of styled text.
type TextLayout struct {
	Lines []TextLine
	Size  Size
}

// TextSelectionRange describes a contiguous selected cell range on one row.
type TextSelectionRange struct {
	Row   int
	Col   int
	Width int
}

// TextCursorCellOptions controls cursor mapping at soft-wrap boundaries.
type TextCursorCellOptions struct {
	SoftWrap  bool
	WrapWidth int
}

// TextLine describes one laid-out display line.
type TextLine struct {
	Runs   []TextSpan
	Width  int
	Offset int
	Cells  []TextCell
	Start  TextPosition
	End    TextPosition
}

// TextCell describes one grapheme cell in a text layout.
type TextCell struct {
	Text      string
	Width     int
	Style     Style
	Position  TextPosition
	Synthetic bool
}

// End returns the text position immediately after this cell.
func (c TextCell) End() TextPosition {
	return advanceTextPosition(c.Position, c.Text)
}

// TextPosition identifies a location in styled text.
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

type textCharacter struct {
	char     Character
	position TextPosition
	end      TextPosition
}

type (
	textLayoutOptions = TextLayoutOptions
)

type textLayoutPaintOptions struct {
	Size           Size
	ScrollRow      int
	ScrollCol      int
	Selection      TextSelection
	SelectionStyle Style
}

func textLayoutSpanRect(layout TextLayout, spanIndex int) (Rect, bool) {
	if spanIndex < 0 {
		return Rect{}, false
	}
	minX, minY := 0, 0
	maxX, maxY := 0, 0
	found := false
	for y, line := range layout.Lines {
		x := line.Offset
		for _, cell := range line.Cells {
			if cell.Position.Span == spanIndex {
				if !found {
					minX, maxX = x, x+cell.Width
					minY, maxY = y, y+1
					found = true
				} else {
					minX = min(minX, x)
					maxX = max(maxX, x+cell.Width)
					minY = min(minY, y)
					maxY = max(maxY, y+1)
				}
			}
			x += cell.Width
		}
	}
	if !found {
		return Rect{}, false
	}
	return Rect{X: minX, Y: minY, Width: max(1, maxX-minX), Height: max(1, maxY-minY)}, true
}

// LayoutText lays out styled text spans into terminal display lines.
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

// PositionForCell maps a layout row and column to a text position.
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

// CellForPosition maps a text position to a layout row and column.
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

// CursorCell maps a text position to the cell where a cursor should be shown.
func (l TextLayout) CursorCell(pos TextPosition, opts TextCursorCellOptions) (row, col int, ok bool) {
	row, col, ok = l.CellForPosition(pos)
	if !ok {
		return 0, 0, false
	}
	if opts.SoftWrap && opts.WrapWidth > 0 && col >= opts.WrapWidth {
		return row + 1, 0, true
	}
	return row, col, true
}

// SelectionRanges maps a text selection to visible layout cell ranges.
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

func paintVisibleTextLayout(p *Painter, off Offset, layout TextLayout, opts textLayoutPaintOptions) {
	paintVisibleTextSelection(p, off, layout, opts)
	for row := opts.ScrollRow; row < len(layout.Lines) && row < opts.ScrollRow+opts.Size.Height; row++ {
		line := layout.Lines[row]
		y := off.Y + row - opts.ScrollRow
		x := line.Offset - opts.ScrollCol
		for _, cell := range line.Cells {
			style := cell.Style
			if opts.Selection.IntersectsCell(cell) {
				style = mergeStyle(style, opts.SelectionStyle)
			}
			p.DrawText(Offset{X: off.X + x, Y: y}, cell.Text, style)
			x += cell.Width
		}
	}
}

func paintVisibleTextSelection(p *Painter, off Offset, layout TextLayout, opts textLayoutPaintOptions) {
	for _, selection := range layout.SelectionRanges(opts.Selection) {
		if selection.Row < opts.ScrollRow || selection.Row >= opts.ScrollRow+opts.Size.Height {
			continue
		}
		col := selection.Col
		width := selection.Width
		if col < opts.ScrollCol {
			width -= opts.ScrollCol - col
			col = opts.ScrollCol
		}
		if col+width > opts.ScrollCol+opts.Size.Width {
			width = opts.ScrollCol + opts.Size.Width - col
		}
		if width <= 0 {
			continue
		}
		p.Fill(Rect{
			X:      off.X + col - opts.ScrollCol,
			Y:      off.Y + selection.Row - opts.ScrollRow,
			Width:  width,
			Height: 1,
		}, Cell{Character: Character{Grapheme: " ", Width: 1}, Style: opts.SelectionStyle})
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

func textLayoutPositionForPoint(layout TextLayout, pt Point) (TextPosition, bool) {
	if len(layout.Lines) == 0 {
		return TextPosition{}, true
	}
	row := clampInt(pt.Y, 0, len(layout.Lines)-1)
	if pt.Y < 0 {
		return layout.Lines[0].Start, true
	}
	if pt.Y >= len(layout.Lines) {
		return layout.Lines[len(layout.Lines)-1].End, true
	}
	pos, ok := layout.PositionForCell(row, pt.X)
	if ok {
		return pos, true
	}
	if pt.X < layout.Lines[row].Offset {
		return layout.Lines[row].Start, true
	}
	return layout.Lines[row].End, true
}

func textSelectionForSpans(spans []TextSpan) TextSelection {
	return TextSelection{Extent: textEndPositionForSpans(spans)}
}

func textWordSelectionForSpans(spans []TextSpan, pos TextPosition) TextSelection {
	chars := textCharactersForSpans(spans)
	if len(chars) == 0 {
		return TextSelection{Base: pos, Extent: pos}
	}
	offset, ok := textCharacterIndexForPosition(chars, pos)
	if !ok {
		return TextSelection{Base: pos, Extent: pos}
	}
	kind := textBufferKind(chars[offset].char)
	start := offset
	for start > 0 && textBufferKind(chars[start-1].char) == kind {
		start--
	}
	end := offset + 1
	for end < len(chars) && textBufferKind(chars[end].char) == kind {
		end++
	}
	return TextSelection{Base: textPositionAtCharacterIndex(chars, start), Extent: textPositionAtCharacterIndex(chars, end)}
}

func textLineSelectionForSpans(spans []TextSpan, pos TextPosition) TextSelection {
	chars := textCharactersForSpans(spans)
	if len(chars) == 0 {
		return TextSelection{Base: pos, Extent: pos}
	}
	if sameTextPosition(pos, chars[len(chars)-1].end) && chars[len(chars)-1].char.Grapheme == "\n" {
		return TextSelection{Base: pos, Extent: pos}
	}
	offset, ok := textCharacterIndexForPosition(chars, pos)
	if !ok {
		return TextSelection{Base: pos, Extent: pos}
	}
	start := offset
	for start > 0 && chars[start-1].char.Grapheme != "\n" {
		start--
	}
	end := offset
	for end < len(chars) && chars[end].char.Grapheme != "\n" {
		end++
	}
	if end < len(chars) && chars[end].char.Grapheme == "\n" {
		end++
	}
	return TextSelection{Base: textPositionAtCharacterIndex(chars, start), Extent: textPositionAtCharacterIndex(chars, end)}
}

func textEndPositionForSpans(spans []TextSpan) TextPosition {
	end := TextPosition{}
	for spanIndex, span := range spans {
		end.Span = spanIndex
		end.ByteOffset = 0
		end.RuneOffset = 0
		end.GraphemeOffset = 0
		for _, ch := range vaxisCharacters(span.Text) {
			end = advanceTextPosition(end, ch.Grapheme)
		}
	}
	return end
}

func textCharactersForSpans(spans []TextSpan) []textCharacter {
	var chars []textCharacter
	for spanIndex, span := range spans {
		pos := TextPosition{Span: spanIndex}
		for _, ch := range vaxisCharacters(span.Text) {
			end := advanceTextPosition(pos, ch.Grapheme)
			chars = append(chars, textCharacter{char: ch, position: pos, end: end})
			pos = end
		}
	}
	return chars
}

func textCharacterIndexForPosition(chars []textCharacter, pos TextPosition) (int, bool) {
	for i, ch := range chars {
		if compareTextPosition(ch.position, pos) <= 0 && compareTextPosition(pos, ch.end) < 0 {
			return i, true
		}
	}
	if len(chars) > 0 && sameTextPosition(pos, chars[len(chars)-1].end) {
		return len(chars) - 1, true
	}
	return 0, false
}

func textPositionAtCharacterIndex(chars []textCharacter, index int) TextPosition {
	if index >= len(chars) {
		return chars[len(chars)-1].end
	}
	return chars[index].position
}

func selectedTextForSpans(spans []TextSpan, selection TextSelection) string {
	if selection.IsCollapsed() {
		return ""
	}
	selection = selection.Normalized()
	out := ""
	for spanIndex, span := range spans {
		pos := TextPosition{Span: spanIndex}
		for _, ch := range vaxisCharacters(span.Text) {
			end := advanceTextPosition(pos, ch.Grapheme)
			if compareTextPosition(selection.Base, pos) <= 0 && compareTextPosition(pos, selection.Extent) < 0 {
				out += ch.Grapheme
			}
			pos = end
		}
	}
	return out
}
