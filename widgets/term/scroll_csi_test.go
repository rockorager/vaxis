package term

import "testing"

func TestScrollUpDownZeroParameterDoesNothing(t *testing.T) {
	tests := []struct {
		name  string
		final rune
	}{
		{name: "scroll up", final: 'S'},
		{name: "scroll down", final: 'T'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 3)
			setScreenLine(vt.primaryScreen, 0, "AAA")
			setScreenLine(vt.primaryScreen, 1, "BBB")
			setScreenLine(vt.primaryScreen, 2, "CCC")
			vt.lastCol = true

			vt.update(testCSI(tt.final, []uint32{0}))

			if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
			if !vt.lastCol {
				t.Fatal("zero-count scroll reset pending wrap")
			}
		})
	}
}

func TestScrollUpDownDefaultParameterScrollsOnce(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")

	vt.update(testCSI('S', nil))
	if got, want := vt.String(), "BBB\nCCC\n   "; got != want {
		t.Fatalf("screen after CSI S = %q want %q", got, want)
	}

	vt.update(testCSI('T', nil))
	if got, want := vt.String(), "   \nBBB\nCCC"; got != want {
		t.Fatalf("screen after CSI T = %q want %q", got, want)
	}
}

func TestScrollUpDownIgnoreMultipleParameters(t *testing.T) {
	tests := []struct {
		name   string
		final  rune
		params []uint32
	}{
		{name: "scroll up", final: 'S', params: []uint32{1, 1}},
		{name: "scroll down", final: 'T', params: []uint32{1, 1}},
		{name: "scroll down mouse form", final: 'T', params: []uint32{1, 2, 3, 4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 3)
			setScreenLine(vt.primaryScreen, 0, "AAA")
			setScreenLine(vt.primaryScreen, 1, "BBB")
			setScreenLine(vt.primaryScreen, 2, "CCC")

			vt.update(testCSI(tt.final, tt.params))

			if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
		})
	}
}
