package vaxis

import (
	"testing"

	"go.rockorager.dev/vaxis/ansi"
)

func TestParseMouseEventHorizontalWheelButtons(t *testing.T) {
	tests := []struct {
		name   string
		button MouseButton
		code   uint32
	}{
		{name: "left", button: MouseWheelLeft, code: 66},
		{name: "right", button: MouseWheelRight, code: 67},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mouse, ok := parseMouseEvent(ansi.CSI{
				Intermediate:    [ansi.MaxIntermediate]rune{'<'},
				NumIntermediate: 1,
				Parameters:      [ansi.InlineCSIParams]uint32{tt.code, 1, 1},
				NumParameters:   3,
				Final:           'M',
			}, Resize{}, false)
			if !ok {
				t.Fatal("mouse parse failed")
			}
			if mouse.Button != tt.button {
				t.Fatalf("parsed button = %d, want %d", mouse.Button, tt.button)
			}
		})
	}
}
