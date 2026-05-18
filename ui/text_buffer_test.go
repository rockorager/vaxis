package ui

import "testing"

func TestTextBufferInsertDeleteAndCursor(t *testing.T) {
	var b TextBuffer
	if !b.Insert("a\nb") {
		t.Fatal("insert returned false")
	}
	if got := b.Text(); got != "a\nb" {
		t.Fatalf("text = %q, want a\\nb", got)
	}
	if got := b.Cursor(); got != (TextCursor{Line: 1, Column: 1}) {
		t.Fatalf("cursor = %#v, want line 1 col 1", got)
	}
	if !b.DeleteBackward() {
		t.Fatal("delete backward returned false")
	}
	if got := b.Text(); got != "a\n" {
		t.Fatalf("after backspace text = %q, want a\\n", got)
	}
	if got := b.Cursor(); got != (TextCursor{Line: 1, Column: 0}) {
		t.Fatalf("after backspace cursor = %#v, want line 1 col 0", got)
	}
	if !b.DeleteBackward() {
		t.Fatal("delete newline returned false")
	}
	if got := b.Text(); got != "a" {
		t.Fatalf("after deleting newline text = %q, want a", got)
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("after deleting newline cursor = %#v, want line 0 col 1", got)
	}
}

func TestTextBufferForwardDeleteAndSetCursor(t *testing.T) {
	b := NewTextBuffer("ab\ncd")
	b.SetCursor(TextCursor{Line: 0, Column: 1})
	if !b.DeleteForward() {
		t.Fatal("delete forward returned false")
	}
	if got := b.Text(); got != "a\ncd" {
		t.Fatalf("text after delete = %q, want a\\ncd", got)
	}
	if !b.DeleteForward() {
		t.Fatal("delete newline returned false")
	}
	if got := b.Text(); got != "acd" {
		t.Fatalf("text after deleting newline = %q, want acd", got)
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor after delete = %#v, want line 0 col 1", got)
	}
}

func TestTextBufferLineMovementPreservesColumn(t *testing.T) {
	b := NewTextBuffer("a\nabcd\nxy")
	b.SetCursor(TextCursor{Line: 1, Column: 3})
	if !b.MoveLineDown() {
		t.Fatal("move down returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 2, Column: 2}) {
		t.Fatalf("cursor after down = %#v, want line 2 col 2", got)
	}
	if !b.MoveLineUp() {
		t.Fatal("move up returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 1, Column: 3}) {
		t.Fatalf("cursor after up = %#v, want restored line 1 col 3", got)
	}
	b.MoveHome()
	if got := b.Cursor(); got != (TextCursor{Line: 1, Column: 0}) {
		t.Fatalf("cursor after home = %#v, want line 1 col 0", got)
	}
	b.MoveEnd()
	if got := b.Cursor(); got != (TextCursor{Line: 1, Column: 4}) {
		t.Fatalf("cursor after end = %#v, want line 1 col 4", got)
	}
}

func TestTextBufferLeftRightMovement(t *testing.T) {
	b := NewTextBuffer("ab")
	if b.MoveLeft() {
		t.Fatal("move left at start returned true")
	}
	if !b.MoveRight() {
		t.Fatal("first move right returned false")
	}
	if !b.MoveRight() {
		t.Fatal("second move right returned false")
	}
	if b.MoveRight() {
		t.Fatal("move right at end returned true")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 2}) {
		t.Fatalf("cursor = %#v, want line 0 col 2", got)
	}
	if !b.MoveLeft() {
		t.Fatal("move left from end returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor after left = %#v, want line 0 col 1", got)
	}
}

func TestTextBufferPositionUsesGraphemeByteAndRuneOffsets(t *testing.T) {
	b := NewTextBuffer("a界b")
	b.SetCursor(TextCursor{Line: 0, Column: 2})
	pos := b.Position()
	if pos.ByteOffset != len("a界") || pos.RuneOffset != 2 || pos.GraphemeOffset != 2 {
		t.Fatalf("position = %#v, want after two graphemes", pos)
	}
	if !b.SetPosition(TextPosition{ByteOffset: len("a")}) {
		t.Fatal("set position returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor from position = %#v, want line 0 col 1", got)
	}
	if b.SetPosition(TextPosition{ByteOffset: len("a") + 1}) {
		t.Fatal("set position inside grapheme unexpectedly succeeded")
	}
}

func TestTextBufferLayoutMapsCursorCell(t *testing.T) {
	b := NewTextBuffer("ab\n界c")
	b.SetCursor(TextCursor{Line: 1, Column: 1})
	layout := b.Layout(Constraints{MaxWidth: 10, MaxHeight: 10}, TextLayoutOptions{})
	row, col, ok := b.CursorCell(layout)
	if !ok || row != 1 || col != 2 {
		t.Fatalf("cursor cell = %d,%d ok=%v, want 1,2 true", row, col, ok)
	}
	if !b.MoveToCell(layout, 0, 1) {
		t.Fatal("move to cell returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor after move to cell = %#v, want line 0 col 1", got)
	}
}

func TestTextBufferVisualMovementUsesWrappedLayout(t *testing.T) {
	b := NewTextBuffer("abcdef")
	b.SetCursor(TextCursor{Line: 0, Column: 1})
	layout := b.Layout(Constraints{MaxWidth: 3, MaxHeight: 10}, TextLayoutOptions{SoftWrap: true})
	if !b.MoveVisualDown(layout) {
		t.Fatal("visual down returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 4}) {
		t.Fatalf("cursor after visual down = %#v, want col 4", got)
	}
	if !b.MoveVisualUp(layout) {
		t.Fatal("visual up returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor after visual up = %#v, want col 1", got)
	}
}

func TestTextBufferSelectionReplacesAndDeletes(t *testing.T) {
	b := NewTextBuffer("abcd")
	b.SetCursor(TextCursor{Line: 0, Column: 1})
	if !b.ExtendRight() {
		t.Fatal("first extend right returned false")
	}
	if !b.ExtendRight() {
		t.Fatal("second extend right returned false")
	}
	if got := b.SelectedText(); got != "bc" {
		t.Fatalf("selected text = %q, want bc", got)
	}
	if !b.Insert("X") {
		t.Fatal("insert returned false")
	}
	if got := b.Text(); got != "aXd" {
		t.Fatalf("text after replacing selection = %q, want aXd", got)
	}
	if b.HasSelection() {
		t.Fatal("selection should collapse after insert")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 2}) {
		t.Fatalf("cursor after replacing selection = %#v, want line 0 col 2", got)
	}

	b.SetCursor(TextCursor{Line: 0, Column: 0})
	if !b.ExtendRight() {
		t.Fatal("first extend right returned false")
	}
	if !b.ExtendRight() {
		t.Fatal("second extend right returned false")
	}
	if !b.DeleteForward() {
		t.Fatal("delete forward returned false")
	}
	if got := b.Text(); got != "d" {
		t.Fatalf("text after deleting selection = %q, want d", got)
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 0}) {
		t.Fatalf("cursor after deleting selection = %#v, want line 0 col 0", got)
	}
}

func TestTextBufferPlainMovementCollapsesSelection(t *testing.T) {
	b := NewTextBuffer("abcd")
	b.SetCursor(TextCursor{Line: 0, Column: 1})
	if !b.ExtendRight() {
		t.Fatal("first extend right returned false")
	}
	if !b.ExtendRight() {
		t.Fatal("second extend right returned false")
	}
	if !b.MoveLeft() {
		t.Fatal("move left returned false")
	}
	if b.HasSelection() {
		t.Fatal("selection should collapse after plain movement")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 1}) {
		t.Fatalf("cursor after collapsing left = %#v, want line 0 col 1", got)
	}
}

func TestTextBufferWordMovementSkipsWhitespaceAndGroupsPunctuation(t *testing.T) {
	b := NewTextBuffer("one  two, three")
	if !b.MoveWordRight() {
		t.Fatal("first word right returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 3}) {
		t.Fatalf("cursor after first word = %#v, want col 3", got)
	}
	if !b.MoveWordRight() {
		t.Fatal("second word right returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 8}) {
		t.Fatalf("cursor after second word = %#v, want col 8", got)
	}
	if !b.MoveWordRight() {
		t.Fatal("punctuation word right returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 9}) {
		t.Fatalf("cursor after punctuation = %#v, want col 9", got)
	}
	if !b.MoveWordLeft() {
		t.Fatal("word left returned false")
	}
	if got := b.Cursor(); got != (TextCursor{Line: 0, Column: 8}) {
		t.Fatalf("cursor after word left = %#v, want col 8", got)
	}
}

func TestTextBufferWordSelectionAndDeletion(t *testing.T) {
	b := NewTextBuffer("alpha beta, gamma")
	b.SetCursor(TextCursor{Line: 0, Column: 6})
	if !b.ExtendWordRight() {
		t.Fatal("extend word right returned false")
	}
	if got := b.SelectedText(); got != "beta" {
		t.Fatalf("selected text = %q, want beta", got)
	}
	if !b.DeleteWordForward() {
		t.Fatal("delete selection returned false")
	}
	if got := b.Text(); got != "alpha , gamma" {
		t.Fatalf("text after deleting selection = %q, want alpha , gamma", got)
	}

	b.SetCursor(TextCursor{Line: 0, Column: len("alpha , gamma")})
	if !b.DeleteWordBackward() {
		t.Fatal("delete word backward returned false")
	}
	if got := b.Text(); got != "alpha , " {
		t.Fatalf("text after deleting word backward = %q, want alpha comma space", got)
	}
	if !b.DeleteWordBackward() {
		t.Fatal("delete trailing punctuation returned false")
	}
	if got := b.Text(); got != "alpha " {
		t.Fatalf("text after deleting punctuation = %q, want alpha space", got)
	}
}
