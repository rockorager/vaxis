package vaxis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderedWidth(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		unicodeWidth int
		wcwidthWidth int
	}{
		{
			name:         "a",
			input:        "a",
			unicodeWidth: 1,
			wcwidthWidth: 1,
		},
		{
			name:         "emoji with ZWJ",
			input:        "👩‍🚀",
			unicodeWidth: 2,
			wcwidthWidth: 4,
		},
		{
			name:         "emoji with VS16 selector",
			input:        "\xE2\x9D\xA4\xEF\xB8\x8F",
			unicodeWidth: 2,
			// This is *technically* wrong but most ter
			wcwidthWidth: 1,
		},
		{
			name:         "emoji with skintone selector",
			input:        "👋🏿",
			unicodeWidth: 2,
			wcwidthWidth: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.unicodeWidth, gwidth(test.input, true))
			assert.Equal(t, test.wcwidthWidth, gwidth(test.input, false))
		})
	}
}
