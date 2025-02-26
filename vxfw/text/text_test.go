package text

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"github.com/stretchr/testify/assert"
)

func TestText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		width    uint16
	}{
		{
			name:     "no wrap, perfect width",
			input:    "foo",
			expected: []string{"foo"},
			width:    3,
		},
		{
			name:     "no wrap, large",
			input:    "foo",
			expected: []string{"foo"},
			width:    4,
		},
		{
			name:  "simple",
			input: "foo bar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "hard break",
			input: "foo\nbar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "lots of space",
			input: "foo         bar",
			expected: []string{
				"foo",
				"bar",
			},
			width: 3,
		},
		{
			name:  "hard break and leading space",
			input: " foo\n bar",
			expected: []string{
				" foo",
				" bar",
			},
			width: 4,
		},
		{
			name:  "long word",
			input: "longwordwithnobreaks",
			expected: []string{
				"long",
				"word",
				"with",
				"nobr",
				"eaks",
			},
			width: 4,
		},
		{
			name:  "no soft wrap needed",
			input: "each line\nfits",
			expected: []string{
				"each line",
				"fits",
			},
			width: 10,
		},
	}

	ctx := vxfw.DrawContext{
		Characters: vaxis.Characters,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scanner := NewSoftwrapScanner(test.input, test.width)
			lines := []string{}
			for scanner.Scan(ctx) {
				lines = append(lines, scanner.Text())
			}
			assert.Equal(t, test.expected, lines)
		})
	}
}
