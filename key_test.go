package rtk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name     string
		key      Key
	}{
		{
			name:     "j",
			key:      Key{Codepoint: 'j'},
		},
		{
			name:     "Ctrl+@",
			key:      Key{Codepoint: 0x00},
		},
		{
			name:     "Ctrl+a",
			key:      Key{Codepoint: 0x01},
		},
		{
			name:     "Alt+a",
			key:      Key{Codepoint: 'a', Modifiers: ModAlt},
		},
		{
			name:     "F1",
			key:      Key{Codepoint: KeyF01},
		},
		{
			name:     "Shift+F1",
			key:      Key{Codepoint: KeyF01, Modifiers: ModShift},
		},
		{
			name:     "Shift+Tab",
			key:      Key{Codepoint: KeyTab, Modifiers: ModShift},
		},
		{
			name:     "Escape",
			key:      Key{Codepoint: KeyEsc},
		},
		{
			name:     "space",
			key:      Key{Codepoint: KeySpace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.key.String()
			assert.Equal(t, test.name, actual)
		})
	}
}
