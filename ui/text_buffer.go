package ui

import (
	"unicode"
	"unicode/utf8"
)

type TextCursor struct {
	Line   int
	Column int
}

type TextBuffer struct {
	chars              []Character
	anchor             int
	cursor             int
	preferredColumn    int
	hasPreferredColumn bool
}

func NewTextBuffer(text string) TextBuffer {
	return TextBuffer{chars: vaxisCharacters(text)}
}

func (b *TextBuffer) SetText(text string) {
	b.chars = vaxisCharacters(text)
	b.anchor = clampInt(b.anchor, 0, len(b.chars))
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
	b.setCursorOffset(offset, false)
	b.clearPreferredColumn()
}

func (b TextBuffer) Selection() TextSelection {
	return TextSelection{Base: b.positionForOffset(b.anchor), Extent: b.positionForOffset(b.CursorOffset())}
}

func (b *TextBuffer) SetSelection(selection TextSelection) bool {
	base, ok := b.offsetForPosition(selection.Base)
	if !ok {
		return false
	}
	extent, ok := b.offsetForPosition(selection.Extent)
	if !ok {
		return false
	}
	b.anchor = base
	b.cursor = extent
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) CollapseSelection(pos TextPosition) bool {
	offset, ok := b.offsetForPosition(pos)
	if !ok {
		return false
	}
	b.setCursorOffset(offset, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendSelection(pos TextPosition) bool {
	offset, ok := b.offsetForPosition(pos)
	if !ok {
		return false
	}
	b.setCursorOffset(offset, true)
	b.clearPreferredColumn()
	return true
}

func (b TextBuffer) HasSelection() bool {
	start, end := b.selectionOffsets()
	return start != end
}

func (b TextBuffer) SelectedText() string {
	start, end := b.selectionOffsets()
	if start == end {
		return ""
	}
	return charactersString(b.chars[start:end])
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
	b.setCursorOffset(b.offsetForCursor(cursor), false)
	b.clearPreferredColumn()
}

func (b *TextBuffer) Insert(text string) bool {
	insert := vaxisCharacters(text)
	if len(insert) == 0 {
		return false
	}
	start, end := b.selectionOffsets()
	next := make([]Character, 0, len(b.chars)+len(insert))
	next = append(next, b.chars[:start]...)
	next = append(next, insert...)
	next = append(next, b.chars[end:]...)
	b.chars = next
	b.setCursorOffset(start+len(insert), false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteBackward() bool {
	if b.deleteSelection() {
		return true
	}
	cursor := b.CursorOffset()
	if cursor == 0 {
		return false
	}
	b.chars = append(b.chars[:cursor-1], b.chars[cursor:]...)
	b.setCursorOffset(cursor-1, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteForward() bool {
	if b.deleteSelection() {
		return true
	}
	cursor := b.CursorOffset()
	if cursor >= len(b.chars) {
		return false
	}
	b.chars = append(b.chars[:cursor], b.chars[cursor+1:]...)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveLeft() bool {
	if b.HasSelection() {
		start, _ := b.selectionOffsets()
		b.setCursorOffset(start, false)
		b.clearPreferredColumn()
		return true
	}
	if b.CursorOffset() == 0 {
		return false
	}
	b.setCursorOffset(b.cursor-1, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveRight() bool {
	if b.HasSelection() {
		_, end := b.selectionOffsets()
		b.setCursorOffset(end, false)
		b.clearPreferredColumn()
		return true
	}
	if b.CursorOffset() >= len(b.chars) {
		return false
	}
	b.setCursorOffset(b.cursor+1, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendLeft() bool {
	if b.CursorOffset() == 0 {
		return false
	}
	b.setCursorOffset(b.cursor-1, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendRight() bool {
	if b.CursorOffset() >= len(b.chars) {
		return false
	}
	b.setCursorOffset(b.cursor+1, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveWordLeft() bool {
	if b.HasSelection() {
		start, _ := b.selectionOffsets()
		b.setCursorOffset(start, false)
		b.clearPreferredColumn()
		return true
	}
	next := b.previousWordBoundary(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveWordRight() bool {
	if b.HasSelection() {
		_, end := b.selectionOffsets()
		b.setCursorOffset(end, false)
		b.clearPreferredColumn()
		return true
	}
	next := b.nextWordBoundary(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendWordLeft() bool {
	next := b.previousWordBoundary(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendWordRight() bool {
	next := b.nextWordBoundary(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteWordBackward() bool {
	if b.deleteSelection() {
		return true
	}
	cursor := b.CursorOffset()
	next := b.previousWordBoundary(cursor)
	if next == cursor {
		return false
	}
	b.chars = append(b.chars[:next], b.chars[cursor:]...)
	b.setCursorOffset(next, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) DeleteWordForward() bool {
	if b.deleteSelection() {
		return true
	}
	cursor := b.CursorOffset()
	next := b.nextWordBoundary(cursor)
	if next == cursor {
		return false
	}
	b.chars = append(b.chars[:cursor], b.chars[next:]...)
	b.setCursorOffset(cursor, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveHome() bool {
	next := b.lineStart(b.CursorOffset())
	if next == b.cursor {
		if b.HasSelection() {
			b.setCursorOffset(next, false)
			b.clearPreferredColumn()
			return true
		}
		return false
	}
	b.setCursorOffset(next, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveEnd() bool {
	next := b.lineEnd(b.CursorOffset())
	if next == b.cursor {
		if b.HasSelection() {
			b.setCursorOffset(next, false)
			b.clearPreferredColumn()
			return true
		}
		return false
	}
	b.setCursorOffset(next, false)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendHome() bool {
	next := b.lineStart(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) ExtendEnd() bool {
	next := b.lineEnd(b.CursorOffset())
	if next == b.cursor {
		return false
	}
	b.setCursorOffset(next, true)
	b.clearPreferredColumn()
	return true
}

func (b *TextBuffer) MoveLineUp() bool {
	cursor := b.Cursor()
	if cursor.Line == 0 {
		return false
	}
	column := b.verticalColumn(cursor.Column)
	b.setCursorOffset(b.offsetForCursor(TextCursor{Line: cursor.Line - 1, Column: column}), false)
	return true
}

func (b *TextBuffer) MoveLineDown() bool {
	cursor := b.Cursor()
	if cursor.Line >= b.lineCount()-1 {
		return false
	}
	column := b.verticalColumn(cursor.Column)
	b.setCursorOffset(b.offsetForCursor(TextCursor{Line: cursor.Line + 1, Column: column}), false)
	return true
}

func (b *TextBuffer) ExtendLineUp() bool {
	return b.moveLine(-1, true)
}

func (b *TextBuffer) ExtendLineDown() bool {
	return b.moveLine(1, true)
}

func (b *TextBuffer) SelectAll() bool {
	if len(b.chars) == 0 && b.anchor == 0 && b.cursor == 0 {
		return false
	}
	b.anchor = 0
	b.cursor = len(b.chars)
	b.clearPreferredColumn()
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
	b.setCursorOffset(offset, false)
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
	return b.moveVisual(layout, -1, false)
}

func (b *TextBuffer) MoveVisualDown(layout TextLayout) bool {
	return b.moveVisual(layout, 1, false)
}

func (b *TextBuffer) ExtendVisualUp(layout TextLayout) bool {
	return b.moveVisual(layout, -1, true)
}

func (b *TextBuffer) ExtendVisualDown(layout TextLayout) bool {
	return b.moveVisual(layout, 1, true)
}

func (b *TextBuffer) moveVisual(layout TextLayout, delta int, extend bool) bool {
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
	b.setCursorOffset(offset, extend)
	return true
}

func (b *TextBuffer) moveLine(delta int, extend bool) bool {
	cursor := b.Cursor()
	if delta < 0 && cursor.Line == 0 {
		return false
	}
	if delta > 0 && cursor.Line >= b.lineCount()-1 {
		return false
	}
	column := b.verticalColumn(cursor.Column)
	b.setCursorOffset(b.offsetForCursor(TextCursor{Line: cursor.Line + delta, Column: column}), extend)
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

func (b TextBuffer) previousWordBoundary(offset int) int {
	offset = clampInt(offset, 0, len(b.chars))
	for offset > 0 && textBufferKind(b.chars[offset-1]) == textBufferSpace {
		offset--
	}
	if offset == 0 {
		return 0
	}
	kind := textBufferKind(b.chars[offset-1])
	for offset > 0 && textBufferKind(b.chars[offset-1]) == kind {
		offset--
	}
	return offset
}

func (b TextBuffer) nextWordBoundary(offset int) int {
	offset = clampInt(offset, 0, len(b.chars))
	for offset < len(b.chars) && textBufferKind(b.chars[offset]) == textBufferSpace {
		offset++
	}
	if offset >= len(b.chars) {
		return len(b.chars)
	}
	kind := textBufferKind(b.chars[offset])
	for offset < len(b.chars) && textBufferKind(b.chars[offset]) == kind {
		offset++
	}
	return offset
}

type textBufferCharKind int

const (
	textBufferSpace textBufferCharKind = iota
	textBufferWord
	textBufferPunctuation
)

func textBufferKind(ch Character) textBufferCharKind {
	if ch.Grapheme == "" {
		return textBufferSpace
	}
	r, _ := utf8.DecodeRuneInString(ch.Grapheme)
	if unicode.IsSpace(r) {
		return textBufferSpace
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return textBufferWord
	}
	return textBufferPunctuation
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

func (b *TextBuffer) setCursorOffset(offset int, extend bool) {
	b.cursor = clampInt(offset, 0, len(b.chars))
	if !extend {
		b.anchor = b.cursor
	}
}

func (b TextBuffer) selectionOffsets() (int, int) {
	anchor := clampInt(b.anchor, 0, len(b.chars))
	cursor := b.CursorOffset()
	if anchor <= cursor {
		return anchor, cursor
	}
	return cursor, anchor
}

func (b *TextBuffer) deleteSelection() bool {
	start, end := b.selectionOffsets()
	if start == end {
		return false
	}
	b.chars = append(b.chars[:start], b.chars[end:]...)
	b.setCursorOffset(start, false)
	b.clearPreferredColumn()
	return true
}

func (b TextBuffer) positionForOffset(offset int) TextPosition {
	pos := TextPosition{}
	for _, ch := range b.chars[:clampInt(offset, 0, len(b.chars))] {
		pos.ByteOffset += len(ch.Grapheme)
		pos.RuneOffset += utf8.RuneCountInString(ch.Grapheme)
		pos.GraphemeOffset++
	}
	return pos
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
