package vaxis_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
	}{
		{
			name: "j",
			key:  vaxis.Key{Keycode: 'j'},
		},
		{
			name: "Ctrl+@",
			key:  vaxis.Key{Keycode: 0x00},
		},
		{
			name: "Ctrl+a",
			key:  vaxis.Key{Keycode: 0x01},
		},
		{
			name: "Alt+a",
			key:  vaxis.Key{Keycode: 'a', Modifiers: vaxis.ModAlt},
		},
		{
			name: "F1",
			key:  vaxis.Key{Keycode: vaxis.KeyF01},
		},
		{
			name: "Shift+F1",
			key:  vaxis.Key{Keycode: vaxis.KeyF01, Modifiers: vaxis.ModShift},
		},
		{
			name: "Shift+Tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift},
		},
		{
			name: "Escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc},
		},
		{
			name: "space",
			key:  vaxis.Key{Keycode: vaxis.KeySpace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.key.String()
			assert.Equal(t, test.name, actual)
		})
	}
}

func ExampleKey() {
	vx, _ := vaxis.New(vaxis.Options{})
	msg := vx.PollEvent()
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "Ctrl+c":
			vx.Close()
		case "Ctrl+l":
			vx.Refresh()
		case "j":
			// Down?
		default:
			// handle the key
		}
	}
}
