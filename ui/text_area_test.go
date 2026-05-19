package ui_test

import (
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type textAreaHarness struct{ value string }

func (h *textAreaHarness) Build(ui.BuildContext) ui.Widget {
	return ui.TextArea{
		Value:     h.value,
		SoftWrap:  true,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	}
}

func TestTextAreaEditsControlledMultilineValue(t *testing.T) {
	h := &textAreaHarness{}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 12, Height: 4})

	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})
	p := ui.NewPainter(ui.Size{Width: 12, Height: 4})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 1 || cursor.Row != 1 {
		t.Fatalf("cursor after enter = %#v ok=%v, want 1,1", cursor, ok)
	}

	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})

	if h.value != "a\nb" {
		t.Fatalf("value = %q, want a\\nb", h.value)
	}
	p = ui.NewPainter(ui.Size{Width: 12, Height: 4})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "a" {
		t.Fatalf("first line = %q, want a", got)
	}
	if got := p.Cell(1, 1).Grapheme; got != "b" {
		t.Fatalf("second line = %q, want b", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 2,1", cursor, ok)
	}
}

func TestTextAreaPlaceholderUsesPlaceholderStyleWhenUnfocused(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.TextField.Placeholder = ui.Style{Foreground: vaxis.ColorGray, Background: vaxis.ColorBlack}
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "x"},
		ui.TextArea{Placeholder: "notes", MinWidth: 8, MinHeight: 2},
	), ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(6, 0).Grapheme; got != "n" {
		t.Fatalf("placeholder = %q, want n", got)
	}
	if got := p.Cell(6, 0).Style; got != theme.TextField.Placeholder {
		t.Fatalf("placeholder style = %#v, want %#v", got, theme.TextField.Placeholder)
	}
}

func TestTextAreaMovesVisuallyThroughWrappedLines(t *testing.T) {
	h := &textAreaHarness{value: "abcdef"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 5, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight})
	app.Pump(ui.Size{Width: 5, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 5, Height: 4})

	p := ui.NewPainter(ui.Size{Width: 5, Height: 4})
	app.Paint(p)
	if got := p.Cell(1, 1).Grapheme; got != "d" {
		t.Fatalf("wrapped second visual line = %q, want d", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 2,1", cursor, ok)
	}
}

func TestTextAreaKeepsCursorVisibleAtSoftWrapBoundary(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "printable", text: "abc"},
		{name: "space", text: "ab "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &textAreaHarness{}
			app := ui.NewApp(h)
			app.Pump(ui.Size{Width: 5, Height: 4})

			app.Send(vaxis.Key{Text: tt.text, Keycode: []rune(tt.text)[0]})
			app.UpdateRoot(h)
			app.Pump(ui.Size{Width: 5, Height: 4})

			p := ui.NewPainter(ui.Size{Width: 5, Height: 4})
			app.Paint(p)
			if cursor, ok := p.Cursor(); !ok || cursor.Col != 1 || cursor.Row != 1 {
				t.Fatalf("cursor after %q = %#v ok=%v, want 1,1", tt.text, cursor, ok)
			}
		})
	}
}

func TestTextAreaExtendsAndPaintsSelection(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "abcd", SoftWrap: true})
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 3})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("selected second cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("unselected third cell background = %#v, should not be selection background", got)
	}
	if got := p.Cell(1, 0).Style; got == ui.DefaultTheme().TextField.Cursor {
		t.Fatalf("selected cell style = %#v, should not use cursor style", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("cursor after extending selection = %#v ok=%v, want 3,0", cursor, ok)
	}
}

func TestTextAreaCopiesSelection(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.TextArea{Value: "abcd", SoftWrap: true}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now)
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift}, now)
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab" {
		t.Fatalf("copies = %#v, want ab", backend.copies)
	}
}

func TestTextAreaIgnoresKeyRelease(t *testing.T) {
	h := &textAreaHarness{}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Key{Text: "x", Keycode: 'x', EventType: vaxis.EventRelease})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 10, Height: 3})
	if h.value != "" {
		t.Fatalf("value after key release = %q, want empty", h.value)
	}
}

func TestTextAreaWordMovementAndDeletionKeys(t *testing.T) {
	h := &textAreaHarness{value: "alpha beta, gamma"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 30, Height: 3})

	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 30, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 30, Height: 3})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 6 || cursor.Row != 0 {
		t.Fatalf("cursor after ctrl-right = %#v ok=%v, want 6,0", cursor, ok)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModCtrl | vaxis.ModShift})
	app.Pump(ui.Size{Width: 30, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyBackspace, Modifiers: vaxis.ModCtrl})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 30, Height: 3})
	if h.value != "alpha, gamma" {
		t.Fatalf("value after deleting selected word = %q, want alpha, gamma", h.value)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 30, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyBackspace, Modifiers: vaxis.ModCtrl})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 30, Height: 3})
	if h.value != "alpha, " {
		t.Fatalf("value after ctrl-backspace = %q, want alpha comma space", h.value)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyHome})
	app.Pump(ui.Size{Width: 30, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDelete, Modifiers: vaxis.ModCtrl})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 30, Height: 3})
	if h.value != ", " {
		t.Fatalf("value after ctrl-delete = %q, want comma space", h.value)
	}
}

func TestTextAreaMouseDragSelectsText(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "abcd", SoftWrap: true})
	app.Pump(ui.Size{Width: 10, Height: 3})

	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 3})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 3})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("selected second cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("unselected third cell background = %#v, should not be selection background", got)
	}
}

func TestTextAreaMouseReverseDragCopiesSelection(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.TextArea{Value: "abcd", SoftWrap: true}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab" {
		t.Fatalf("copies = %#v, want ab", backend.copies)
	}
}

func TestTextAreaPaintsSelectedEmptyLine(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "a\n\nb", SoftWrap: true})
	app.Pump(ui.Size{Width: 10, Height: 5})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 10, Height: 5})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 5})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 5})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 5})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 1).Background; got != want {
		t.Fatalf("selected empty line background = %#v, want %#v", got, want)
	}
}

func TestTextAreaSelectAllDoesNotPaintExtraCellAfterNonEmptyLineBeforeBlankLine(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "abc\n\nabc", SoftWrap: true})
	app.Pump(ui.Size{Width: 12, Height: 5})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 12, Height: 5})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 12, Height: 5})

	p := ui.NewPainter(ui.Size{Width: 12, Height: 5})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 1).Background; got != want {
		t.Fatalf("selected empty line background = %#v, want %#v", got, want)
	}
	if got := p.Cell(4, 0).Background; got == want {
		t.Fatalf("extra cell after first line background = %#v, should not be selection background", got)
	}
}

func TestTextAreaScrollsVerticallyToCursor(t *testing.T) {
	h := &textAreaHarness{value: "a\nb\nc"}
	app := ui.NewApp(ui.TextArea{
		Value:     h.value,
		MinHeight: 2,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 10, Height: 2})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "b" {
		t.Fatalf("top visible line = %q, want b", got)
	}
	if got := p.Cell(1, 1).Grapheme; got != "c" {
		t.Fatalf("bottom visible line = %q, want c", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 1 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 1,1", cursor, ok)
	}
}

func TestTextAreaSelectionPaintsVisibleRowsWhenStartIsOffscreen(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "a\nb\nc", MinHeight: 2})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 10, Height: 2})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 2})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Grapheme; got != "b" {
		t.Fatalf("top visible line = %q, want b", got)
	}
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("top visible selection background = %#v, want %#v", got, want)
	}
	if got := p.Cell(1, 1).Grapheme; got != "c" {
		t.Fatalf("bottom visible line = %q, want c", got)
	}
	if got := p.Cell(1, 1).Background; got != want {
		t.Fatalf("bottom visible selection background = %#v, want %#v", got, want)
	}
}

func TestTextAreaScrollsHorizontallyWithoutSoftWrap(t *testing.T) {
	h := &textAreaHarness{value: "abcdef"}
	app := ui.NewApp(ui.TextArea{
		Value:     h.value,
		MinWidth:  5,
		SoftWrap:  false,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	})
	app.Pump(ui.Size{Width: 5, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 5, Height: 3})

	p := ui.NewPainter(ui.Size{Width: 5, Height: 3})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "e" {
		t.Fatalf("first visible scrolled cell = %q, want e", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "f" {
		t.Fatalf("second visible scrolled cell = %q, want f", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("cursor = %#v ok=%v, want 3,0", cursor, ok)
	}
}

func TestTextAreaSelectionClipsHorizontally(t *testing.T) {
	app := ui.NewApp(ui.TextArea{Value: "abcdef", MinWidth: 5, SoftWrap: false})
	app.Pump(ui.Size{Width: 5, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 5, Height: 3})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 5, Height: 3})

	p := ui.NewPainter(ui.Size{Width: 5, Height: 3})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Grapheme; got != "e" {
		t.Fatalf("first visible selected cell = %q, want e", got)
	}
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("first visible selected background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Grapheme; got != "f" {
		t.Fatalf("second visible selected cell = %q, want f", got)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("second visible selected background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("cell after horizontally clipped selection background = %#v, should not be selected", got)
	}
}
