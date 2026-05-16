package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestSelectionViewportSourceRows(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.primaryScreen.setCell(0, 0, cellString("a"))
	vt.primaryScreen.setCell(1, 0, cellString("b"))
	vt.scrollUp(1)
	vt.scrollViewport(1)

	if got, want := mustViewportSourceRow(t, vt, 0), 0; got != want {
		t.Fatalf("top source row = %d, want %d", got, want)
	}
	if got, want := mustViewportSourceRow(t, vt, 1), 1; got != want {
		t.Fatalf("bottom source row = %d, want %d", got, want)
	}
}

func TestSelectionRangeContainsForwardAndReverse(t *testing.T) {
	for _, sel := range []selectionRange{
		{start: selectionPoint{sourceRow: 1, col: 2}, end: selectionPoint{sourceRow: 3, col: 1}},
		{start: selectionPoint{sourceRow: 3, col: 1}, end: selectionPoint{sourceRow: 1, col: 2}},
	} {
		if sel.contains(1, 1) {
			t.Fatal("selection included cell before top-left")
		}
		if !sel.contains(1, 2) || !sel.contains(2, 0) || !sel.contains(2, 99) || !sel.contains(3, 1) {
			t.Fatalf("selection containment mismatch for %#v", sel)
		}
		if sel.contains(3, 2) {
			t.Fatal("selection included cell after bottom-right")
		}
	}
}

func TestSelectionRenderOverlay(t *testing.T) {
	vt := New()
	vt.resize(3, 1)
	setScreenLine(vt.primaryScreen, 0, "abc")
	sourceRow, ok := vt.viewportSourceRow(0)
	if !ok {
		t.Fatal("missing source row")
	}
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: sourceRow, col: 1},
		end:   selectionPoint{sourceRow: sourceRow, col: 2},
	})

	snapshot := vt.snapshotDraw(nil)
	if got := snapshot.cells[0].cell.Attribute & vaxis.AttrReverse; got != 0 {
		t.Fatalf("unselected cell reverse attribute = %v, want off", got)
	}
	if got := snapshot.cells[1].cell.Attribute & vaxis.AttrReverse; got == 0 {
		t.Fatal("selected cell did not render reverse")
	}
	if got := snapshot.cells[2].cell.Attribute & vaxis.AttrReverse; got == 0 {
		t.Fatal("selected end cell did not render reverse")
	}
}

func TestSelectionRenderComposesWithReverseVideo(t *testing.T) {
	vt := New()
	vt.resize(2, 1)
	setScreenLine(vt.primaryScreen, 0, "ab")
	vt.mode.decscnm = true
	sourceRow, ok := vt.viewportSourceRow(0)
	if !ok {
		t.Fatal("missing source row")
	}
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: sourceRow, col: 0},
		end:   selectionPoint{sourceRow: sourceRow, col: 1},
	})

	snapshot := vt.snapshotDraw(nil)
	if got := snapshot.cells[0].cell.Attribute & vaxis.AttrReverse; got != 0 {
		t.Fatalf("selected reverse-video cell reverse attribute = %v, want toggled off", got)
	}
}

func TestSelectionStringHardAndSoftWraps(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	setScreenLine(vt.primaryScreen, 0, "hello")
	setScreenLine(vt.primaryScreen, 1, "world")
	setScreenLine(vt.primaryScreen, 2, "again")
	vt.primaryScreen.row(0).wrapped = true
	vt.primaryScreen.row(1).wrapContinuation = true

	history := vt.primaryScreen.scrollbackLen()
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: history, col: 0},
		end:   selectionPoint{sourceRow: history + 2, col: 4},
	})

	if got, want := vt.Selection(), "helloworld\nagain"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionStringWideGraphemeOnce(t *testing.T) {
	vt := New()
	vt.resize(4, 1)
	setWideCell(vt.primaryScreen, 0, 0, "橋")
	vt.primaryScreen.setCell(0, 2, cellString("x"))

	sourceRow, ok := vt.viewportSourceRow(0)
	if !ok {
		t.Fatal("missing source row")
	}
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: sourceRow, col: 0},
		end:   selectionPoint{sourceRow: sourceRow, col: 2},
	})

	if got, want := vt.Selection(), "橋x"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseDragCreatesSelection(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	setScreenLine(vt.primaryScreen, 0, "abcde")

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 1})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 3})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 3})

	if got, want := vt.Selection(), "bcd"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseReverseDragCreatesSelection(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	setScreenLine(vt.primaryScreen, 0, "abcde")

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 3})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 1})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 1})

	if got, want := vt.Selection(), "bcd"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseDragAcrossVisibleScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 1})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 2, Col: 1})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 2, Col: 1})

	if got, want := vt.Selection(), "ABCD\n2EFGH\n3I"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseClickClearsExistingSelection(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	setScreenLine(vt.primaryScreen, 0, "abcde")
	sourceRow, ok := vt.viewportSourceRow(0)
	if !ok {
		t.Fatal("missing source row")
	}
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: sourceRow, col: 0},
		end:   selectionPoint{sourceRow: sourceRow, col: 2},
	})

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 4})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 4})

	if vt.HasSelection() {
		t.Fatal("click did not clear existing selection")
	}
}

func TestSelectionMouseShiftBypassesMouseReporting(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	setScreenLine(vt.primaryScreen, 0, "abcde")
	vt.mode.mouseEvent = mouseEventAny
	vt.mode.mouseFormat = mouseFormatSGR

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 0, Modifiers: vaxis.ModShift})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 2, Modifiers: vaxis.ModShift})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 2, Modifiers: vaxis.ModShift})

	if got, want := vt.Selection(), "abc"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseShiftCapturedByMouseReporting(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.mouseEvent = mouseEventAny
	vt.mode.mouseFormat = mouseFormatSGR
	vt.mode.mouseShiftCapture = true

	handled := vt.handleSelectionMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		EventType: vaxis.EventPress,
		Row:       0,
		Col:       0,
		Modifiers: vaxis.ModShift,
	})
	if handled {
		t.Fatal("selection consumed shift mouse despite mouseShiftCapture")
	}
	if got, want := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		EventType: vaxis.EventPress,
		Row:       0,
		Col:       0,
		Modifiers: vaxis.ModShift,
	}), "\x1b[<4;1;1M"; got != want {
		t.Fatalf("mouse report = %q, want %q", got, want)
	}
}

func TestSelectionMouseDoubleClickSelectsWord(t *testing.T) {
	vt := New()
	vt.resize(12, 1)
	setScreenLine(vt.primaryScreen, 0, "hello world")

	doubleClick(vt, 0, 7, 0)

	if got, want := vt.Selection(), "world"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseDoubleClickDragExtendsByWords(t *testing.T) {
	vt := New()
	vt.resize(20, 1)
	setScreenLine(vt.primaryScreen, 0, "one two three")

	doubleClickPress(vt, 0, 1, 0)
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 9})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 9})

	if got, want := vt.Selection(), "one two three"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseTripleClickSelectsSoftWrappedLine(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	setScreenLine(vt.primaryScreen, 0, "hello")
	setScreenLine(vt.primaryScreen, 1, "world")
	setScreenLine(vt.primaryScreen, 2, "next")
	vt.primaryScreen.row(0).wrapped = true
	vt.primaryScreen.row(1).wrapContinuation = true

	tripleClick(vt, 0, 1, 0)

	if got, want := vt.Selection(), "helloworld"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseTripleClickDragExtendsByLines(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	setScreenLine(vt.primaryScreen, 0, "one")
	setScreenLine(vt.primaryScreen, 1, "two")
	setScreenLine(vt.primaryScreen, 2, "three")

	tripleClickPress(vt, 0, 1, 0)
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 2, Col: 1})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 2, Col: 1})

	if got, want := vt.Selection(), "one\ntwo\nthree"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMouseRectangleDrag(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	setScreenLine(vt.primaryScreen, 0, "abcde")
	setScreenLine(vt.primaryScreen, 1, "fghij")
	setScreenLine(vt.primaryScreen, 2, "klmno")

	mods := vaxis.ModCtrl | vaxis.ModAlt
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 1, Modifiers: mods})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 2, Col: 3, Modifiers: mods})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 2, Col: 3, Modifiers: mods})

	if vt.selection == nil || !vt.selection.rectangle {
		t.Fatal("selection is not rectangular")
	}
	if got, want := vt.Selection(), "bcd\nghi\nlmn"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMousePixelThresholdIncludesEndpointCells(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.size = vaxis.Resize{Cols: 5, Rows: 1, XPixel: 50, YPixel: 10}
	setScreenLine(vt.primaryScreen, 0, "abcde")

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 1, XPixel: 10})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 3, XPixel: 39})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 3, XPixel: 39})

	if got, want := vt.Selection(), "bcd"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestSelectionMousePixelThresholdCanCreateEmptySelection(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.size = vaxis.Resize{Cols: 5, Rows: 1, XPixel: 50, YPixel: 10}
	setScreenLine(vt.primaryScreen, 0, "abcde")

	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 1, XPixel: 19})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: 2, XPixel: 20})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: 2, XPixel: 20})

	if vt.HasSelection() {
		t.Fatal("threshold-only drag created a selection")
	}
}

func TestSelectionClearsOnRISResizeAltScreenAndScrollbackClear(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	setSelectionForTest(t, vt, 0, 0, 0, 1)

	vt.ris()
	if vt.HasSelection() {
		t.Fatal("RIS did not clear selection")
	}

	vt.resize(5, 2)
	setSelectionForTest(t, vt, 0, 0, 0, 1)
	vt.resize(6, 2)
	if vt.HasSelection() {
		t.Fatal("resize did not clear selection")
	}

	setSelectionForTest(t, vt, 0, 0, 0, 1)
	vt.switchAltScreen(1047, true)
	if vt.HasSelection() {
		t.Fatal("alternate screen entry did not clear selection")
	}

	vt.switchAltScreen(1047, false)
	setSelectionForTest(t, vt, 0, 0, 0, 1)
	vt.ed(3, false)
	if vt.HasSelection() {
		t.Fatal("scrollback clear did not clear selection")
	}
}

func TestSelectionClearsOnKeyAndPasteInput(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	setSelectionForTest(t, vt, 0, 0, 0, 1)

	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter})
	if vt.HasSelection() {
		t.Fatal("key input did not clear selection")
	}

	setSelectionForTest(t, vt, 0, 0, 0, 1)
	vt.Update(vaxis.PasteStartEvent{})
	if vt.HasSelection() {
		t.Fatal("paste start did not clear selection")
	}

	setSelectionForTest(t, vt, 0, 0, 0, 1)
	vt.Update(vaxis.PasteEndEvent{})
	if vt.HasSelection() {
		t.Fatal("paste end did not clear selection")
	}
}

func TestSelectionViewportKeyDoesNotClearSelection(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	setSelectionForTest(t, vt, 2, 0, 2, 1)

	vt.Update(vaxis.Key{Keycode: vaxis.KeyPgUp, Modifiers: vaxis.ModShift})

	if !vt.HasSelection() {
		t.Fatal("viewport key cleared selection")
	}
}

func TestSelectionClearsOnAlternateScrollWheelTranslation(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.smcup = true
	vt.mode.altScroll = true
	setSelectionForTest(t, vt, 0, 0, 0, 1)

	report := vt.handleMouse(vaxis.Mouse{Button: vaxis.MouseWheelUp, EventType: vaxis.EventPress})

	if report == "" {
		t.Fatal("wheel did not translate to cursor keys")
	}
	if vt.HasSelection() {
		t.Fatal("alternate-scroll wheel translation did not clear selection")
	}
}

func doubleClick(vt *Model, row int, col int, mods vaxis.ModifierMask) {
	doubleClickPress(vt, row, col, mods)
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: row, Col: col, Modifiers: mods})
}

func doubleClickPress(vt *Model, row int, col int, mods vaxis.ModifierMask) {
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: row, Col: col, Modifiers: mods})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: row, Col: col, Modifiers: mods})
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: row, Col: col, Modifiers: mods})
}

func tripleClick(vt *Model, row int, col int, mods vaxis.ModifierMask) {
	tripleClickPress(vt, row, col, mods)
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: row, Col: col, Modifiers: mods})
}

func tripleClickPress(vt *Model, row int, col int, mods vaxis.ModifierMask) {
	doubleClick(vt, row, col, mods)
	vt.Update(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: row, Col: col, Modifiers: mods})
}

func mustViewportSourceRow(t *testing.T, vt *Model, viewportRow int) int {
	t.Helper()
	sourceRow, ok := vt.viewportSourceRow(viewportRow)
	if !ok {
		t.Fatalf("viewport row %d did not map to source row", viewportRow)
	}
	return sourceRow
}

func setSelectionForTest(t *testing.T, vt *Model, startRow int, startCol int, endRow int, endCol int) {
	t.Helper()
	startSource, ok := vt.viewportSourceRow(startRow)
	if !ok {
		t.Fatalf("viewport row %d did not map to source row", startRow)
	}
	endSource, ok := vt.viewportSourceRow(endRow)
	if !ok {
		t.Fatalf("viewport row %d did not map to source row", endRow)
	}
	vt.setSelectionLocked(&selectionRange{
		start: selectionPoint{sourceRow: startSource, col: startCol},
		end:   selectionPoint{sourceRow: endSource, col: endCol},
	})
}
