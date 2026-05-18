package term

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
)

const mouseSelectionClickInterval = 500 * time.Millisecond

type selectionPoint struct {
	sourceRow int
	col       int
}

type selectionRange struct {
	start     selectionPoint
	end       selectionPoint
	rectangle bool
}

type mouseSelectionState struct {
	active       bool
	start        selectionPoint
	startXPixel  int
	clicks       int
	lastClickAt  time.Time
	lastClickRow int
	lastClickCol int
	hadSelection bool
	dragged      bool
	rectangle    bool
}

func (p selectionPoint) before(other selectionPoint) bool {
	if p.sourceRow != other.sourceRow {
		return p.sourceRow < other.sourceRow
	}
	return p.col < other.col
}

func (p selectionPoint) equal(other selectionPoint) bool {
	return p.sourceRow == other.sourceRow && p.col == other.col
}

func (r selectionRange) ordered() (selectionPoint, selectionPoint) {
	if r.end.before(r.start) {
		return r.end, r.start
	}
	return r.start, r.end
}

func (r selectionRange) contains(sourceRow int, col int) bool {
	start, end := r.ordered()
	if sourceRow < start.sourceRow || sourceRow > end.sourceRow {
		return false
	}
	if r.rectangle {
		left, right := start.col, end.col
		if right < left {
			left, right = right, left
		}
		return col >= left && col <= right
	}
	if start.sourceRow == end.sourceRow {
		return col >= start.col && col <= end.col
	}
	if sourceRow == start.sourceRow {
		return col >= start.col
	}
	if sourceRow == end.sourceRow {
		return col <= end.col
	}
	return true
}

func (r selectionRange) rowSpan(sourceRow int, width int) (int, int, bool) {
	start, end := r.ordered()
	if sourceRow < start.sourceRow || sourceRow > end.sourceRow {
		return 0, 0, false
	}
	if r.rectangle {
		left, right := start.col, end.col
		if right < left {
			left, right = right, left
		}
		if width > 0 {
			left = clampInt(left, 0, width-1)
			right = clampInt(right, 0, width-1)
		}
		return left, right, true
	}
	switch {
	case start.sourceRow == end.sourceRow:
		return start.col, end.col, true
	case sourceRow == start.sourceRow:
		right := end.col
		if width > 0 {
			right = width - 1
		}
		return start.col, right, true
	case sourceRow == end.sourceRow:
		return 0, end.col, true
	default:
		right := end.col
		if width > 0 {
			right = width - 1
		}
		return 0, right, true
	}
}

func (vt *Model) selectionContains(sourceRow int, col int) bool {
	if vt.selection == nil {
		return false
	}
	return vt.selection.contains(sourceRow, col)
}

func (vt *Model) setSelectionLocked(sel *selectionRange) {
	if sel == nil {
		if vt.selection != nil {
			vt.selection = nil
			vt.invalidate()
		}
		return
	}
	copy := *sel
	vt.selection = &copy
	vt.invalidate()
}

func (vt *Model) clearSelectionLocked() {
	if vt.selection == nil && !vt.selectionMouse.active {
		return
	}
	vt.selection = nil
	vt.selectionMouse.active = false
	vt.selectionMouse.dragged = false
	vt.selectionMouse.hadSelection = false
	vt.invalidate()
}

func (vt *Model) HasSelection() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	return vt.selection != nil
}

func (vt *Model) ClearSelection() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.clearSelectionLocked()
}

func (vt *Model) Selection() string {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	return vt.selectionStringLocked()
}

func (vt *Model) selectionStringLocked() string {
	if vt.selection == nil {
		return ""
	}
	start, end := vt.selection.ordered()
	var b strings.Builder
	for sourceRow := start.sourceRow; sourceRow <= end.sourceRow; sourceRow += 1 {
		line, meta, ok := vt.sourceRowLine(sourceRow)
		if !ok || len(line) == 0 {
			continue
		}
		left, right, ok := vt.selection.rowSpan(sourceRow, len(line))
		if !ok {
			continue
		}
		left = clampInt(left, 0, len(line)-1)
		right = clampInt(right, 0, len(line)-1)
		hardBreak := sourceRow != end.sourceRow && (!meta.wrapped || vt.selection.rectangle)
		writeSelectionCells(&b, line, left, right, hardBreak)
		if hardBreak {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeSelectionCells(b *strings.Builder, line []cell, left int, right int, trimRight bool) {
	if trimRight {
		for right >= left && selectionCellBlank(line[right]) {
			right -= 1
		}
	}
	for col := left; col <= right; col += 1 {
		cell := line[col]
		if cell.Width == 0 && cell.Grapheme == " " {
			continue
		}
		b.WriteString(cell.rune())
	}
}

func selectionCellBlank(cell cell) bool {
	return cell.Grapheme == "" || (cell.Grapheme == " " && cell.Width == 0)
}

func (vt *Model) handleSelectionMouse(msg vaxis.Mouse) bool {
	if msg.Button != vaxis.MouseLeftButton {
		if vt.mode.mouseEvent != mouseEventNone && vt.selection != nil && msg.EventType == vaxis.EventPress {
			vt.clearSelectionLocked()
		}
		return false
	}

	canSelect := vt.mode.mouseEvent == mouseEventNone
	if !canSelect && msg.Modifiers&vaxis.ModShift != 0 && !vt.mode.mouseShiftCapture {
		canSelect = true
	}
	if !canSelect {
		if msg.EventType == vaxis.EventPress && vt.selection != nil {
			vt.clearSelectionLocked()
		}
		return false
	}

	point, ok := vt.selectionPointFromMouse(msg)
	if !ok {
		return false
	}

	switch msg.EventType {
	case vaxis.EventPress:
		vt.startSelectionMouse(point, msg)
		return true
	case vaxis.EventMotion:
		if !vt.selectionMouse.active {
			return false
		}
		vt.updateSelectionMouse(point, msg.XPixel)
		return true
	case vaxis.EventRelease:
		if !vt.selectionMouse.active {
			return false
		}
		return vt.finishSelectionMouse(point, msg.XPixel)
	default:
		return false
	}
}

func (vt *Model) selectionPointFromMouse(msg vaxis.Mouse) (selectionPoint, bool) {
	if msg.Row < 0 || msg.Row >= vt.height() || msg.Col < 0 || msg.Col >= vt.width() {
		return selectionPoint{}, false
	}
	sourceRow, ok := vt.viewportSourceRow(msg.Row)
	if !ok {
		return selectionPoint{}, false
	}
	return selectionPoint{
		sourceRow: sourceRow,
		col:       msg.Col,
	}, true
}

func (vt *Model) startSelectionMouse(point selectionPoint, msg vaxis.Mouse) {
	now := time.Now()
	if vt.selectionMouse.clicks == 0 ||
		now.Sub(vt.selectionMouse.lastClickAt) > mouseSelectionClickInterval ||
		absInt(point.sourceRow-vt.selectionMouse.lastClickRow) > 0 ||
		absInt(point.col-vt.selectionMouse.lastClickCol) > 1 {
		vt.selectionMouse.clicks = 0
	}
	vt.selectionMouse.clicks += 1
	if vt.selectionMouse.clicks > 3 {
		vt.selectionMouse.clicks = 1
	}
	vt.selectionMouse.lastClickAt = now
	vt.selectionMouse.lastClickRow = point.sourceRow
	vt.selectionMouse.lastClickCol = point.col
	vt.selectionMouse.active = true
	vt.selectionMouse.start = point
	vt.selectionMouse.startXPixel = msg.XPixel
	vt.selectionMouse.hadSelection = vt.selection != nil
	vt.selectionMouse.dragged = false
	vt.selectionMouse.rectangle = selectionRectangleModifiers(msg.Modifiers)
	if vt.selection != nil {
		vt.selection = nil
		vt.invalidate()
	}
	switch vt.selectionMouse.clicks {
	case 2:
		if sel, ok := vt.selectWordAt(point); ok {
			vt.setSelectionLocked(&sel)
		}
	case 3:
		if sel, ok := vt.selectLineAt(point); ok {
			vt.setSelectionLocked(&sel)
		}
	}
}

func (vt *Model) updateSelectionMouse(point selectionPoint, xPixel int) {
	vt.selectionMouse.dragged = true
	switch vt.selectionMouse.clicks {
	case 2:
		vt.updateWordSelection(point)
		return
	case 3:
		vt.updateLineSelection(point)
		return
	}
	sel := selectionRange{
		start:     vt.selectionMouse.start,
		end:       point,
		rectangle: vt.selectionMouse.rectangle,
	}
	if adjusted, ok, empty := vt.pixelAdjustedSelection(sel, vt.selectionMouse.startXPixel, xPixel); empty {
		vt.setSelectionLocked(nil)
		return
	} else if ok {
		sel = adjusted
	}
	vt.setSelectionLocked(&sel)
}

func (vt *Model) finishSelectionMouse(point selectionPoint, xPixel int) bool {
	hadSelection := vt.selectionMouse.hadSelection
	dragged := vt.selectionMouse.dragged
	if dragged {
		vt.updateSelectionMouse(point, xPixel)
	}
	vt.selectionMouse.active = false
	vt.selectionMouse.dragged = false
	vt.selectionMouse.hadSelection = false
	vt.selectionMouse.rectangle = false
	if dragged {
		return true
	}
	return hadSelection || vt.selection != nil
}

func (vt *Model) pixelAdjustedSelection(sel selectionRange, startXPixel int, endXPixel int) (selectionRange, bool, bool) {
	if startXPixel == 0 || endXPixel == 0 || vt.size.XPixel <= 0 || vt.width() <= 0 {
		return selectionRange{}, false, false
	}
	cellWidth := vt.size.XPixel / vt.width()
	if cellWidth <= 0 {
		return selectionRange{}, false, false
	}
	threshold := cellWidth * 6 / 10
	if threshold <= 0 {
		threshold = 1
	}
	startFrac := startXPixel % cellWidth
	endFrac := endXPixel % cellWidth
	endBeforeStart := sel.end.before(sel.start)
	if sel.start.equal(sel.end) {
		endBeforeStart = endFrac < startFrac
	} else if sel.rectangle {
		switch {
		case sel.end.col < sel.start.col:
			endBeforeStart = true
		case sel.end.col > sel.start.col:
			endBeforeStart = false
		default:
			endBeforeStart = endFrac < startFrac
		}
	}
	includeStart := startFrac < threshold
	if endBeforeStart {
		includeStart = startFrac >= threshold
	}
	includeEnd := endFrac >= threshold
	if endBeforeStart {
		includeEnd = endFrac < threshold
	}

	start := sel.start
	if !includeStart {
		if endBeforeStart {
			start = vt.selectionPointLeft(start, sel.rectangle)
		} else {
			start = vt.selectionPointRight(start, sel.rectangle)
		}
	}
	end := sel.end
	if !includeEnd {
		if endBeforeStart {
			end = vt.selectionPointRight(end, sel.rectangle)
		} else {
			end = vt.selectionPointLeft(end, sel.rectangle)
		}
	}
	if (!includeStart && start.equal(sel.end)) || (!includeEnd && end.equal(sel.start)) {
		return selectionRange{}, true, true
	}
	return selectionRange{
		start:     start,
		end:       end,
		rectangle: sel.rectangle,
	}, true, false
}

func (vt *Model) selectionPointLeft(point selectionPoint, clamp bool) selectionPoint {
	if point.col > 0 {
		return selectionPoint{sourceRow: point.sourceRow, col: point.col - 1}
	}
	if clamp || point.sourceRow <= 0 {
		return point
	}
	line, _, ok := vt.sourceRowLine(point.sourceRow - 1)
	if !ok || len(line) == 0 {
		return point
	}
	return selectionPoint{sourceRow: point.sourceRow - 1, col: len(line) - 1}
}

func (vt *Model) selectionPointRight(point selectionPoint, clamp bool) selectionPoint {
	line, _, ok := vt.sourceRowLine(point.sourceRow)
	if !ok || len(line) == 0 {
		return point
	}
	if point.col < len(line)-1 {
		return selectionPoint{sourceRow: point.sourceRow, col: point.col + 1}
	}
	if clamp {
		return point
	}
	nextLine, _, ok := vt.sourceRowLine(point.sourceRow + 1)
	if !ok || len(nextLine) == 0 {
		return point
	}
	return selectionPoint{sourceRow: point.sourceRow + 1, col: 0}
}

func (vt *Model) updateWordSelection(point selectionPoint) {
	startWord, ok := vt.selectWordAt(vt.selectionMouse.start)
	if !ok {
		vt.setSelectionLocked(nil)
		return
	}
	currentWord, ok := vt.selectWordAt(point)
	if !ok {
		vt.setSelectionLocked(nil)
		return
	}
	if point.before(vt.selectionMouse.start) {
		vt.setSelectionLocked(&selectionRange{
			start: currentWord.start,
			end:   startWord.end,
		})
		return
	}
	vt.setSelectionLocked(&selectionRange{
		start: startWord.start,
		end:   currentWord.end,
	})
}

func (vt *Model) updateLineSelection(point selectionPoint) {
	startLine, ok := vt.selectLineAt(vt.selectionMouse.start)
	if !ok {
		vt.setSelectionLocked(nil)
		return
	}
	currentLine, ok := vt.selectLineAt(point)
	if !ok {
		vt.setSelectionLocked(nil)
		return
	}
	if point.before(vt.selectionMouse.start) {
		vt.setSelectionLocked(&selectionRange{
			start: currentLine.start,
			end:   startLine.end,
		})
		return
	}
	vt.setSelectionLocked(&selectionRange{
		start: startLine.start,
		end:   currentLine.end,
	})
}

func (vt *Model) selectWordAt(point selectionPoint) (selectionRange, bool) {
	cell, ok := vt.sourceCell(point)
	if !ok || !selectionCellHasText(cell) {
		return selectionRange{}, false
	}
	expectBoundary := selectionCellBoundary(cell)
	start := point
	for {
		prev, ok := vt.previousSelectionPoint(start)
		if !ok {
			break
		}
		prevCell, ok := vt.sourceCell(prev)
		if !ok || !selectionCellHasText(prevCell) || selectionCellBoundary(prevCell) != expectBoundary {
			break
		}
		start = prev
	}
	end := point
	for {
		next, ok := vt.nextSelectionPoint(end)
		if !ok {
			break
		}
		nextCell, ok := vt.sourceCell(next)
		if !ok || !selectionCellHasText(nextCell) || selectionCellBoundary(nextCell) != expectBoundary {
			break
		}
		end = next
	}
	return selectionRange{start: start, end: end}, true
}

func (vt *Model) selectLineAt(point selectionPoint) (selectionRange, bool) {
	line, _, ok := vt.sourceRowLine(point.sourceRow)
	if !ok || len(line) == 0 {
		return selectionRange{}, false
	}
	startRow := point.sourceRow
	for startRow > 0 {
		_, prevMeta, ok := vt.sourceRowLine(startRow - 1)
		if !ok || !prevMeta.wrapped {
			break
		}
		startRow -= 1
	}
	endRow := point.sourceRow
	for {
		_, meta, ok := vt.sourceRowLine(endRow)
		if !ok || !meta.wrapped {
			break
		}
		if _, _, ok := vt.sourceRowLine(endRow + 1); !ok {
			break
		}
		endRow += 1
	}

	startCol := 0
	if startLine, _, ok := vt.sourceRowLine(startRow); ok {
		for startCol < len(startLine)-1 && !selectionCellHasText(startLine[startCol]) {
			startCol += 1
		}
	}
	endLine, _, ok := vt.sourceRowLine(endRow)
	if !ok || len(endLine) == 0 {
		return selectionRange{}, false
	}
	endCol := len(endLine) - 1
	for endCol > 0 && selectionCellBlank(endLine[endCol]) {
		endCol -= 1
	}
	return selectionRange{
		start: selectionPoint{sourceRow: startRow, col: startCol},
		end:   selectionPoint{sourceRow: endRow, col: endCol},
	}, true
}

func (vt *Model) sourceCell(point selectionPoint) (cell, bool) {
	line, _, ok := vt.sourceRowLine(point.sourceRow)
	if !ok || point.col < 0 || point.col >= len(line) {
		return cell{}, false
	}
	return line[point.col], true
}

func (vt *Model) previousSelectionPoint(point selectionPoint) (selectionPoint, bool) {
	if point.col > 0 {
		return selectionPoint{sourceRow: point.sourceRow, col: point.col - 1}, true
	}
	if point.sourceRow <= 0 {
		return selectionPoint{}, false
	}
	line, meta, ok := vt.sourceRowLine(point.sourceRow - 1)
	if !ok || !meta.wrapped || len(line) == 0 {
		return selectionPoint{}, false
	}
	return selectionPoint{sourceRow: point.sourceRow - 1, col: len(line) - 1}, true
}

func (vt *Model) nextSelectionPoint(point selectionPoint) (selectionPoint, bool) {
	line, meta, ok := vt.sourceRowLine(point.sourceRow)
	if !ok || len(line) == 0 {
		return selectionPoint{}, false
	}
	if point.col < len(line)-1 {
		return selectionPoint{sourceRow: point.sourceRow, col: point.col + 1}, true
	}
	if !meta.wrapped {
		return selectionPoint{}, false
	}
	nextLine, _, ok := vt.sourceRowLine(point.sourceRow + 1)
	if !ok || len(nextLine) == 0 {
		return selectionPoint{}, false
	}
	return selectionPoint{sourceRow: point.sourceRow + 1, col: 0}, true
}

func selectionCellHasText(cell cell) bool {
	return cell.Grapheme != "" && (cell.Width != 0 || cell.Grapheme != " ")
}

func selectionCellBoundary(cell cell) bool {
	r, _ := utf8.DecodeRuneInString(cell.rune())
	return unicode.IsSpace(r)
}

func selectionRectangleModifiers(mods vaxis.ModifierMask) bool {
	return mods&vaxis.ModCtrl != 0 && mods&vaxis.ModAlt != 0
}

func clampInt(v int, minValue int, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
