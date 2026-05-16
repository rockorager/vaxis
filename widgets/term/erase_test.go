package term

import "testing"

func TestEraseDisplayIgnoresInvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "multiple", params: []uint32{2, 1}},
		{name: "unknown", params: []uint32{4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 2)
			setScreenLine(vt.primaryScreen, 0, "ABC")
			setScreenLine(vt.primaryScreen, 1, "DEF")

			vt.update(testCSI('J', tt.params))

			if got, want := vt.String(), "ABC\nDEF"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
		})
	}
}

func TestEraseLineIgnoresInvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "multiple", params: []uint32{2, 1}},
		{name: "unknown", params: []uint32{3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 1)
			setScreenLine(vt.primaryScreen, 0, "ABC")

			vt.update(testCSI('K', tt.params))

			if got, want := vt.String(), "ABC"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
		})
	}
}

func TestEraseDisplayScrollCompleteClearsScreenAndScrollback(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	appendViewportLines(vt, "111", "222")
	setScreenLine(vt.primaryScreen, 0, "ABC")
	setScreenLine(vt.primaryScreen, 1, "DEF")
	vt.scrollOffset = 1

	vt.update(testCSI('J', []uint32{22}))

	if got, want := vt.String(), "   \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback len = %d, want 0", got)
	}
	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset = %d, want 0", got)
	}
}
