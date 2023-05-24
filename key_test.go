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
			key:      Key{codepoint: 'j'},
		},
		{
			name:     "<c-@>",
			key:      Key{codepoint: 0x00},
		},
		{
			name:     "<c-a>",
			key:      Key{codepoint: 0x01},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.key.String()
			assert.Equal(t, test.name, actual)
		})
	}
}
