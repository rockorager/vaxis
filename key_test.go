package vaxis

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name string
		key  Key
	}{
		{
			name: "j",
			key:  Key{Keycode: 'j'},
		},
		{
			name: "Ctrl+@",
			key:  Key{Keycode: 0x00},
		},
		{
			name: "Ctrl+a",
			key:  Key{Keycode: 0x01},
		},
		{
			name: "Alt+a",
			key:  Key{Keycode: 'a', Modifiers: ModAlt},
		},
		{
			name: "F1",
			key:  Key{Keycode: KeyF01},
		},
		{
			name: "Shift+F1",
			key:  Key{Keycode: KeyF01, Modifiers: ModShift},
		},
		{
			name: "Shift+Tab",
			key:  Key{Keycode: KeyTab, Modifiers: ModShift},
		},
		{
			name: "Escape",
			key:  Key{Keycode: KeyEsc},
		},
		{
			name: "space",
			key:  Key{Keycode: KeySpace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.key.String()
			assert.Equal(t, test.name, actual)
		})
	}
}

func TestKeyDecode(t *testing.T) {
	tests := []struct {
		name     string
		sequence ansi.Sequence
		expected Key
	}{
		{
			name:     "legacy: j",
			sequence: ansi.Print('j'),
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name:     "legacy: Up",
			sequence: ansi.SS3('A'),
			expected: Key{Keycode: KeyUp},
		},
		{
			name:     "legacy: Up, normal keys",
			sequence: ansi.CSI{Final: 'A'},
			expected: Key{Keycode: KeyUp},
		},
		{
			name:     "legacy: shift+j",
			sequence: ansi.Print('J'),
			expected: Key{
				Keycode:     'j',
				ShiftedCode: 'J',
				Modifiers:   ModShift,
				Text:        "J",
			},
		},
		{
			name: "kitty: j with event",
			sequence: ansi.CSI{
				Final: 'u',
				Parameters: [][]int{
					{106},
					{1, 1},
					{106},
				},
			},
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name: "kitty: j with minimal",
			sequence: ansi.CSI{
				Final: 'u',
				Parameters: [][]int{
					{106},
					{},
					{106},
				},
			},
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name: "kitty: ф",
			sequence: ansi.CSI{
				Final: 'u',
				Parameters: [][]int{
					{1092, 0, 102},
					{},
					{1092},
				},
			},
			expected: Key{
				Keycode:        'ф',
				BaseLayoutCode: 'f',
				Text:           "ф",
			},
		},
		{
			name: "kitty: multiple codepoints",
			sequence: ansi.CSI{
				Final: 'u',
				Parameters: [][]int{
					{106},
					{},
					{127482, 127480},
				},
			},
			expected: Key{
				Keycode: 'j',
				Text:    "🇺🇸",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			act := decodeKey(test.sequence)
			assert.Equal(t, test.expected, act)
		})
	}
}
