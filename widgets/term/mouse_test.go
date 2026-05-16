package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestMouseSGRFormatAloneDoesNotReport(t *testing.T) {
	vt := New()
	vt.mode.mouseSGR = true

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

func TestMouseSGRReleaseKeepsButtonIdentity(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.mode.mouseSGR = true

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
	vt.mode.mouseSGR = true

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

func TestMouseEventModeResetClearsActiveTracking(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.update(testCSI('h', []uint32{1003}, '?'))
	vt.update(testCSI('l', []uint32{1000}, '?'))
	vt.mode.mouseSGR = true

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
	vt.mode.mouseSGR = true

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

func TestMouseModifiersInNonX10Modes(t *testing.T) {
	vt := New()
	vt.update(testCSI('h', []uint32{1000}, '?'))
	vt.mode.mouseSGR = true

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
	vt.mode.mouseSGR = true

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
	vt.mode.mouseSGR = true

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
