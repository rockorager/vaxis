package term

import (
	"strings"
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestMouseSGRFormatAloneDoesNotReport(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		EventType: vaxis.EventPress,
	})

	if got != "" {
		t.Fatalf("mouse report = %q, want empty", got)
	}
}

func TestMouseX10ReportsOnlyBasicPresses(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{9}, '?'))

	if got, want := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       0,
		Row:       0,
		EventType: vaxis.EventPress,
	}), "\x1B[M !!"; got != want {
		t.Fatalf("x10 press report = %q, want %q", got, want)
	}

	if got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		EventType: vaxis.EventRelease,
	}); got != "" {
		t.Fatalf("x10 release report = %q, want empty", got)
	}

	if got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseWheelUp,
		EventType: vaxis.EventPress,
	}); got != "" {
		t.Fatalf("x10 wheel report = %q, want empty", got)
	}
}

func TestMouseX10DropsLargeCoordinates(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{9}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       223,
		Row:       0,
		EventType: vaxis.EventPress,
	})

	if got != "" {
		t.Fatalf("x10 large-coordinate report = %q, want empty", got)
	}
}

func TestMouseNormalLegacyReleaseUsesButton3(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseRightButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1B[M##$"; got != want {
		t.Fatalf("legacy release report = %q, want %q", got, want)
	}
}

func TestMouseEventModeReportingMatchesGhostty(t *testing.T) {
	t.Run("normal ignores motion", func(t *testing.T) {
		vt := New()
		vt.update(testCSI('h', []uint32{1000}, '?'))
		vt.update(testCSI('h', []uint32{1006}, '?'))

		got := vt.handleMouse(vaxis.Mouse{
			Button:    vaxis.MouseLeftButton,
			Col:       1,
			Row:       2,
			EventType: vaxis.EventMotion,
		})
		if got != "" {
			t.Fatalf("normal-mode motion report = %q, want empty", got)
		}
	})

	t.Run("button mode requires button for motion", func(t *testing.T) {
		vt := New()
		vt.update(testCSI('h', []uint32{1002}, '?'))
		vt.update(testCSI('h', []uint32{1006}, '?'))

		got := vt.handleMouse(vaxis.Mouse{
			Button:    vaxis.MouseNoButton,
			Col:       1,
			Row:       2,
			EventType: vaxis.EventMotion,
		})
		if got != "" {
			t.Fatalf("button-mode no-button motion report = %q, want empty", got)
		}

		got = vt.handleMouse(vaxis.Mouse{
			Button:    vaxis.MouseLeftButton,
			Col:       1,
			Row:       2,
			EventType: vaxis.EventMotion,
		})
		if want := "\x1B[<32;2;3M"; got != want {
			t.Fatalf("button-mode drag report = %q, want %q", got, want)
		}
	})
}

func TestMouseSGRWheelButtonMappings(t *testing.T) {
	tests := []struct {
		name   string
		button vaxis.MouseButton
		want   string
	}{
		{name: "wheel up", button: vaxis.MouseWheelUp, want: "\x1B[<64;1;1M"},
		{name: "wheel down", button: vaxis.MouseWheelDown, want: "\x1B[<65;1;1M"},
		{name: "wheel left", button: vaxis.MouseWheelLeft, want: "\x1B[<66;1;1M"},
		{name: "wheel right", button: vaxis.MouseWheelRight, want: "\x1B[<67;1;1M"},
		{name: "button 8", button: vaxis.MouseButton8, want: "\x1B[<128;1;1M"},
		{name: "button 9", button: vaxis.MouseButton9, want: "\x1B[<129;1;1M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.update(testCSI('h', []uint32{1003}, '?'))
			vt.update(testCSI('h', []uint32{1006}, '?'))

			got := vt.handleMouse(vaxis.Mouse{
				Button:    tt.button,
				Col:       0,
				Row:       0,
				EventType: vaxis.EventPress,
			})

			if got != tt.want {
				t.Fatalf("sgr mouse report = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMouseUnsupportedButtonsIgnored(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	for _, button := range []vaxis.MouseButton{vaxis.MouseButton10, vaxis.MouseButton11, vaxis.MouseButton(200)} {
		got := vt.handleMouse(vaxis.Mouse{
			Button:    button,
			Col:       1,
			Row:       1,
			EventType: vaxis.EventPress,
		})
		if got != "" {
			t.Fatalf("unsupported button %d report = %q, want empty", button, got)
		}
	}
}

func TestMouseSGRReleaseKeepsButtonIdentity(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseRightButton,
		Col:       4,
		Row:       5,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1B[<2;5;6m"; got != want {
		t.Fatalf("sgr release report = %q, want %q", got, want)
	}
}

func TestMouseSGRMotionNoButtonAnyMode(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseNoButton,
		Col:       1,
		Row:       2,
		EventType: vaxis.EventMotion,
	})

	if want := "\x1B[<35;2;3M"; got != want {
		t.Fatalf("sgr motion report = %q, want %q", got, want)
	}
}

func TestMouseUTF8EncodesLargeCoordinates(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1005}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       223,
		Row:       224,
		EventType: vaxis.EventPress,
	})

	if want := "\x1B[M \u0100\u0101"; got != want {
		t.Fatalf("utf8 mouse report = %q, want %q", got, want)
	}
}

func TestMouseURXVTFormat(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1015}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModAlt | vaxis.ModCtrl,
	})

	if want := "\x1B[56;3;4M"; got != want {
		t.Fatalf("urxvt mouse report = %q, want %q", got, want)
	}
}

func TestMouseURXVTReleaseUsesLegacyButton3(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1015}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseRightButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1B[35;3;4M"; got != want {
		t.Fatalf("urxvt release report = %q, want %q", got, want)
	}
}

func TestMouseURXVTModifiersWithShiftCapture(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1015}, '?'))
	vt.update(testCSI('s', []uint32{1}, '>'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModShift | vaxis.ModAlt | vaxis.ModCtrl,
	})

	if want := "\x1B[60;3;4M"; got != want {
		t.Fatalf("urxvt modified press report = %q, want %q", got, want)
	}
}

func TestMouseSGRPixelsFormat(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1016}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		XPixel:    50,
		YPixel:    75,
		EventType: vaxis.EventPress,
	})

	if want := "\x1B[<0;50;75M"; got != want {
		t.Fatalf("sgr-pixels mouse report = %q, want %q", got, want)
	}
}

func TestMouseSGRPixelsReleaseKeepsButtonIdentity(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1016}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseRightButton,
		XPixel:    10,
		YPixel:    20,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1B[<2;10;20m"; got != want {
		t.Fatalf("sgr-pixels release report = %q, want %q", got, want)
	}
}

func TestMouseFormatResetClearsActiveFormat(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))
	vt.update(testCSI('h', []uint32{1015}, '?'))
	vt.update(testCSI('l', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
	})

	if want := "\x1B[M #$"; got != want {
		t.Fatalf("mouse report after format reset = %q, want %q", got, want)
	}
	if !vt.mode.mouseURXVT {
		t.Fatal("resetting SGR mouse format cleared report state for URXVT format")
	}
}

func TestMouseEventModeResetClearsActiveTracking(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('l', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseNoButton,
		Col:       1,
		Row:       2,
		EventType: vaxis.EventMotion,
	})

	if got != "" {
		t.Fatalf("mouse report after reset = %q, want empty", got)
	}
	if !vt.mode.mouseMotion {
		t.Fatal("resetting normal mouse mode cleared report state for any-motion mode")
	}
}

func TestMouseEventModeEnableOverridesActiveTracking(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('h', []uint32{9}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseNoButton,
		Col:       1,
		Row:       2,
		EventType: vaxis.EventMotion,
	})

	if got != "" {
		t.Fatalf("x10 motion report = %q, want empty", got)
	}
	if !vt.mode.mouseMotion {
		t.Fatal("enabling x10 mouse mode cleared report state for any-motion mode")
	}
}

func TestMouseModesSaveRestoreActiveTrackingAndFormat(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))
	vt.update(testCSI('s', []uint32{1003, 1006}, '?'))
	vt.update(testCSI('l', []uint32{1003}, '?'))
	vt.update(testCSI('l', []uint32{1006}, '?'))

	if got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseNoButton,
		Col:       1,
		Row:       2,
		EventType: vaxis.EventMotion,
	}); got != "" {
		t.Fatalf("mouse report with saved modes disabled = %q, want empty", got)
	}
	<-vt.events
	<-vt.events

	vt.update(testCSI('r', []uint32{1003, 1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseNoButton,
		Col:       1,
		Row:       2,
		EventType: vaxis.EventMotion,
	})
	if want := "\x1B[<35;2;3M"; got != want {
		t.Fatalf("restored mouse report = %q, want %q", got, want)
	}
}

func TestMouseEventModeUpdatesMouseShape(t *testing.T) {
	vt := New()

	vt.update(testCSI('h', []uint32{1000}, '?'))

	if got, want := vt.mouseShape, vaxis.MouseShapeDefault; got != want {
		t.Fatalf("mouse shape after mouse mode set = %q, want %q", got, want)
	}
	ev := <-vt.events
	shape, ok := ev.(EventMouseShape)
	if !ok {
		t.Fatalf("event = %T, want EventMouseShape", ev)
	}
	if got, want := shape.Shape, vaxis.MouseShapeDefault; got != want {
		t.Fatalf("event shape = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1000}, '?'))

	if got, want := vt.mouseShape, vaxis.MouseShapeTextInput; got != want {
		t.Fatalf("mouse shape after mouse mode reset = %q, want %q", got, want)
	}
	ev = <-vt.events
	shape, ok = ev.(EventMouseShape)
	if !ok {
		t.Fatalf("event = %T, want EventMouseShape", ev)
	}
	if got, want := shape.Shape, vaxis.MouseShapeTextInput; got != want {
		t.Fatalf("event shape = %q, want %q", got, want)
	}
}

func TestMouseEventModeDoesNotRepeatMouseShapeEvent(t *testing.T) {
	vt := New()

	vt.update(testCSI('h', []uint32{1000}, '?'))
	<-vt.events
	vt.update(testCSI('h', []uint32{1003}, '?'))

	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected duplicate mouse shape event: %T", ev)
	default:
	}
	if got, want := vt.mouseShape, vaxis.MouseShapeDefault; got != want {
		t.Fatalf("mouse shape = %q, want %q", got, want)
	}
}

func TestMouseModifiersInNonX10Modes(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModAlt | vaxis.ModCtrl,
	})

	if want := "\x1B[<24;3;4M"; got != want {
		t.Fatalf("sgr modified press report = %q, want %q", got, want)
	}
}

func TestMouseShiftEscapesMouseReportingByDefault(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModShift | vaxis.ModAlt | vaxis.ModCtrl,
	})

	if got != "" {
		t.Fatalf("shift mouse report = %q, want empty", got)
	}
}

func TestXTSHIFTESCAPETogglesShiftMouseCapture(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1006}, '?'))

	vt.update(testCSI('s', []uint32{1}, '>'))
	got := vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModShift | vaxis.ModAlt | vaxis.ModCtrl,
	})
	if want := "\x1B[<28;3;4M"; got != want {
		t.Fatalf("captured shift mouse report = %q, want %q", got, want)
	}

	vt.update(testCSI('s', []uint32{0}, '>'))
	got = vt.handleMouse(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Col:       2,
		Row:       3,
		EventType: vaxis.EventPress,
		Modifiers: vaxis.ModShift,
	})
	if got != "" {
		t.Fatalf("disabled shift mouse report = %q, want empty", got)
	}
}

func TestXTSHIFTESCAPEIgnoresInvalidParams(t *testing.T) {
	vt := New()

	vt.update(testCSI('s', []uint32{1}, '>'))
	if !vt.mode.mouseShiftCapture {
		t.Fatal("XTSHIFTESCAPE did not enable shift capture")
	}

	vt.update(testCSI('s', []uint32{2}, '>'))
	vt.update(testCSI('s', []uint32{1, 1}, '>'))
	if !vt.mode.mouseShiftCapture {
		t.Fatal("invalid XTSHIFTESCAPE changed shift capture")
	}
}

func TestPromptClickEventsEmitsSGRLeftPress(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;click_events=1")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       4,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1b[<0;5;1M"; got != want {
		t.Fatalf("prompt click event = %q, want %q", got, want)
	}
}

func TestPromptClickLineRight(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       4,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1b[C\x1b[C"; got != want {
		t.Fatalf("prompt click move right = %q, want %q", got, want)
	}
}

func TestPromptClickLineLeft(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cursor.col = 6

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       2,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1b[D\x1b[D\x1b[D\x1b[D"; got != want {
		t.Fatalf("prompt click move left = %q, want %q", got, want)
	}
}

func TestPromptClickLineSkipsNonInputCells(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "h")
	vt.osc("133;C")
	printText(vt, "X")
	vt.osc("133;B")
	printText(vt, "llo")
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       5,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1b[C\x1b[C"; got != want {
		t.Fatalf("prompt click skipped move = %q, want %q", got, want)
	}
}

func TestPromptClickLineUsesApplicationCursorKeys(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.update(testCSI('h', []uint32{1}, '?'))
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       3,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1bOC"; got != want {
		t.Fatalf("application cursor prompt click = %q, want %q", got, want)
	}
}

func TestPromptClickLineRightAcrossSoftWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "abcdefghij")
	vt.cursor.row = 0
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       1,
		Col:       1,
		EventType: vaxis.EventRelease,
	})

	if want := strings.Repeat("\x1b[C", 9); got != want {
		t.Fatalf("soft-wrap prompt click right = %q, want %q", got, want)
	}
}

func TestPromptClickLineLeftAcrossSoftWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "abcdefghij")
	vt.cursor.row = 1
	vt.cursor.col = 1

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       2,
		EventType: vaxis.EventRelease,
	})

	if want := strings.Repeat("\x1b[D", 9); got != want {
		t.Fatalf("soft-wrap prompt click left = %q, want %q", got, want)
	}
}

func TestPromptClickLineStopsAtHardLineBreak(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cr()
	vt.lf()
	printText(vt, "world")
	vt.cursor.row = 0
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       1,
		Col:       0,
		EventType: vaxis.EventRelease,
	})

	if want := strings.Repeat("\x1b[C", 5); got != want {
		t.Fatalf("hard-break prompt click right = %q, want %q", got, want)
	}
}

func TestPromptClickLineStopsAtNewPrompt(t *testing.T) {
	vt := New()
	vt.resize(20, 4)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cr()
	vt.lf()
	vt.osc("133;P;k=c")
	vt.osc("133;B")
	printText(vt, "world")
	vt.cr()
	vt.lf()
	vt.osc("133;A")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "again")
	vt.cursor.row = 1
	vt.cursor.col = 0

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       2,
		Col:       2,
		EventType: vaxis.EventRelease,
	})

	if want := strings.Repeat("\x1b[C", 5); got != want {
		t.Fatalf("new-prompt prompt click right = %q, want %q", got, want)
	}
}

func TestPromptClickRightOfInputMovesToEnd(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cursor.col = 2

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       15,
		EventType: vaxis.EventRelease,
	})

	if want := strings.Repeat("\x1b[C", 5); got != want {
		t.Fatalf("right-of-input prompt click = %q, want %q", got, want)
	}
}

func TestPromptClickRightOfInputAtEndDoesNotMove(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       15,
		EventType: vaxis.EventRelease,
	})

	if got != "" {
		t.Fatalf("right-of-input at end prompt click = %q, want empty", got)
	}
}

func TestPromptClickRightOfInputFromLastCharMovesOnce(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A;cl=line")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")
	vt.cursor.col = 6

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       15,
		EventType: vaxis.EventRelease,
	})

	if want := "\x1b[C"; got != want {
		t.Fatalf("right-of-input from last char = %q, want %q", got, want)
	}
}

func TestPromptClickIgnoredWhenDisabledOrNotRelease(t *testing.T) {
	vt := New()
	vt.resize(20, 3)
	vt.osc("133;A")
	printText(vt, "> ")
	vt.osc("133;B")
	printText(vt, "hello")

	got := vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       4,
		EventType: vaxis.EventRelease,
	})
	if got != "" {
		t.Fatalf("disabled prompt click = %q, want empty", got)
	}

	vt.osc("133;A;cl=line")
	got = vt.handlePromptClick(vaxis.Mouse{
		Button:    vaxis.MouseLeftButton,
		Row:       0,
		Col:       4,
		EventType: vaxis.EventPress,
	})
	if got != "" {
		t.Fatalf("prompt click press = %q, want empty", got)
	}
}
