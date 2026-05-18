package ui

import "unicode/utf8"

type TextCursor struct {
	Line   int
	Column int
}

type TextBuffer struct {
	chars              []Character
	cursor             int
	preferredColumn    int
	hasPreferredColumn bool
}

func NewTextBuffer(text string) TextBuffer {
	return TextBuffer{chars: vaxisCharacters(text)}
}

func (b *TextBuffer) SetText(text string) {
	b.chars = vaxisCharacters(text)
	b.cursor = clampInt(b.cursor, 0, len(b.chars))
	b.clearPreferredColumn()
}

func (b TextBuffer) Text() string {
	return charactersString(b.chars)
}

func (b TextBuffer) Len() int {
	return len(b.chars)
}

func (b TextBuffer) CursorOffset() int {
	return clampInt(b.cursor, 0, len(b.chars))
}

func (b *TextBuffer) SetCursorOffset(offset int) {
	b.cursor = clampInt(offset, 0, len(b.chars))
	b.clearPreferredColumn()
}

func (b TextBuffer) Cursor() TextCursor {
	line, col := 0, 0
	for _, ch := range b.chars[:b.CursorOffset()] {
		if ch.Grapheme == "\n" {
			line++
			col = 0
			continue
		}
		col++
	}
	return TextCursor{Line: line, Column: col}
}

func (b *TextBuffer) SetCursor(cursor TextCursor) {
	b.cursor = b.offsetForCursor(cursor)
	b.clearPreferredColumn()
}

func (b *TextBuffer) Insert(text string) bool {
	insert := vaxisCharacters(text)
	if len(insert) == 0 {
		return false
	}
	cursor := b.CursorOffset()
	next := make([]Character, 0, len(b.chars)+len(insert))
	next = append(next, b.chars[:cursor]...)
	next = append(next, insert...)
	next = append(next, b.chars[cursor:]...)
	b.chars = next
	b.cursor = cursor + len(insert)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteBackward() bool {
	cursor := b.CursorOffset()
	if cursor == 0 {
		return false
	}
	b.chars = append(b.chars[:cursor-1], b.chars[cursor:]...)
	b.cursor = cursor - 1
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteForward() bool {
	cursor := b.CursorOffset()
	if cursor >= len(b.chars) {
		return false
	}
	b.chars = append(b.chars[:cursor], b.chars[cursor+1:]...)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveLeft() bool {
	if b.CursorOffset() == 0 {
		return false
	}
	b.cursor--
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveRight() bool {
	if b.CursorOffset() >= len(b.chars) {
		return false
	}
	b.cursor++
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveHome() bool {
	next := b.lineStart(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.cursor = next
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveEnd() bool {
	next := b.lineEnd(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.cursor = next
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveLineUp() bool {
	cursor := b.Cursor()
	if cursor.Line == 0 {
		return false
	}
	column := b.verticalColumn(cursor.Column)
	b.cursor = b.offsetForCursor(TextCursor{Line: cursor.Line - 1, Column: column})
	return true
}

func (b *TextBuffer) MoveLineDown() bool {
	cursor := b.Cursor()
	if cursor.Line >= b.lineCount()-1 {
		return false
	}
	column := b.verticalColumn(cursor.Column)
	b.cursor = b.offsetForCursor(TextCursor{Line: cursor.Line + 1, Column: column})
	return true
}

func (b TextBuffer) Position() TextPosition {
	pos := TextPosition{}
	for _, ch := range b.chars[:b.CursorOffset()] {
		pos.ByteOffset += len(ch.Grapheme)
		pos.RuneOffset += utf8.RuneCountInString(ch.Grapheme)
		pos.GraphemeOffset++
	}
	return pos
}

func (b *TextBuffer) SetPosition(pos TextPosition) bool {
	offset, ok := b.offsetForPosition(pos)
	if !ok {
		return false
	}
	b.cursor = offset
	b.clearPreferredColumn()
	return true
}

func (b TextBuffer) Layout(c Constraints, opts TextLayoutOptions) TextLayout {
	return LayoutText([]TextSpan{{Text: b.Text()}}, c, opts)
}

func (b TextBuffer) CursorCell(layout TextLayout) (row, col int, ok bool) {
	return layout.CellForPosition(b.Position())
}

func (b *TextBuffer) MoveToCell(layout TextLayout, row, col int) bool {
	pos, ok := layout.PositionForCell(row, col)
	if !ok {
		return false
	}
	return b.SetPosition(pos)
}

func (b *TextBuffer) MoveVisualUp(layout TextLayout) bool {
	return b.moveVisual(layout, -1)
}

func (b *TextBuffer) MoveVisualDown(layout TextLayout) bool {
	return b.moveVisual(layout, 1)
}

func (b *TextBuffer) moveVisual(layout TextLayout, delta int) bool {
	row, col, ok := b.CursorCell(layout)
	if !ok {
		return false
	}
	if !b.hasPreferredColumn {
		b.preferredColumn = col
		b.hasPreferredColumn = true
	}
	nextRow := row + delta
	if nextRow < 0 || nextRow >= len(layout.Lines) {
		return false
	}
	pos, ok := layout.PositionForCell(nextRow, b.preferredColumn)
	if !ok {
		return false
	}
	offset, ok := b.offsetForPosition(pos)
	if !ok {
		return false
	}
	b.cursor = offset
	return true
}

func (b TextBuffer) offsetForCursor(cursor TextCursor) int {
	if cursor.Line <= 0 {
		return min(max(0, cursor.Column), b.lineLengthAtOffset(0))
	}
	line, col := 0, 0
	for i, ch := range b.chars {
		if line == cursor.Line {
			if ch.Grapheme == "\n" || col >= cursor.Column {
				return i
			}
			col++
			continue
		}
		if ch.Grapheme == "\n" {
			line++
			col = 0
		}
	}
	if line < cursor.Line {
		return len(b.chars)
	}
	return len(b.chars)
}

func (b TextBuffer) lineLengthAtOffset(offset int) int {
	end := b.lineEnd(offset)
	start := b.lineStart(offset)
	return end - start
}

func (b TextBuffer) lineStart(offset int) int {
	offset = clampInt(offset, 0, len(b.chars))
	for offset > 0 && b.chars[offset-1].Grapheme != "\n" {
		offset--
	}
	return offset
}

func (b TextBuffer) lineEnd(offset int) int {
	offset = clampInt(offset, 0, len(b.chars))
	for offset < len(b.chars) && b.chars[offset].Grapheme != "\n" {
		offset++
	}
	return offset
}

func (b TextBuffer) lineCount() int {
	lines := 1
	for _, ch := range b.chars {
		if ch.Grapheme == "\n" {
			lines++
		}
	}
	return lines
}

func (b *TextBuffer) verticalColumn(current int) int {
	if !b.hasPreferredColumn {
		b.preferredColumn = current
		b.hasPreferredColumn = true
	}
	return b.preferredColumn
}

func (b *TextBuffer) clearPreferredColumn() {
	b.hasPreferredColumn = false
}

func (b TextBuffer) offsetForPosition(pos TextPosition) (int, bool) {
	if pos.Span != 0 {
		return 0, false
	}
	byteOffset := 0
	for i, ch := range b.chars {
		if byteOffset == pos.ByteOffset {
			return i, true
		}
		byteOffset += len(ch.Grapheme)
		if byteOffset > pos.ByteOffset {
			return 0, false
		}
	}
	if byteOffset == pos.ByteOffset {
		return len(b.chars), true
	}
	return 0, false
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
