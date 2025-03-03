package richtext

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/stretchr/testify/assert"
)

func TestHardwrapScanner(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no breaks",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:  "single hard break",
			input: "each line\nfits",
			expected: []string{
				"each line",
				"fits",
			},
		},
		{
			name:  "sequential hardbreak",
			input: "each line\n\nfits",
			expected: []string{
				"each line",
				"",
				"fits",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chars := vaxis.Characters(test.input)
			cells := make([]vaxis.Cell, 0, len(chars))
			for _, char := range chars {
				cell := vaxis.Cell{Character: char}
				cells = append(cells, cell)

			}
			scanner := NewHardwrapScanner(cells)
			lines := []string{}
			for scanner.Scan() {
				str := strings.Builder{}
				line := scanner.Line()
				for _, cell := range line {
					str.WriteString(cell.Grapheme)
				}
				lines = append(lines, str.String())
			}
			assert.Equal(t, test.expected, lines)
		})
	}
}

func TestFirstLineSegment(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		expectedBr bool
	}{
		{
			name:       "no breaks",
			input:      "foo",
			expected:   "foo",
			expectedBr: true,
		},
		{
			name:       "trailing break",
			input:      "foo\n",
			expected:   "foo\n",
			expectedBr: true,
		},
		{
			name:       "leading break",
			input:      "\nbar",
			expected:   "\n",
			expectedBr: true,
		},
		{
			name:       "middle break",
			input:      "foo\nbar",
			expected:   "foo\n",
			expectedBr: true,
		},
		{
			name:       "word break",
			input:      "foo bar",
			expected:   "foo ",
			expectedBr: false,
		},
		{
			name:       "word break with hard break",
			input:      "foo \nbar",
			expected:   "foo \n",
			expectedBr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chars := vaxis.Characters(test.input)
			cells := make([]vaxis.Cell, 0, len(chars))
			for _, char := range chars {
				cell := vaxis.Cell{Character: char}
				cells = append(cells, cell)
			}

			seg, br := firstLineSegment(cells)
			str := strings.Builder{}
			for _, cell := range seg {
				str.WriteString(cell.Grapheme)
			}
			assert.Equal(t, test.expected, str.String())
			assert.Equal(t, test.expectedBr, br)
		})
	}
}

func TestSoftWrapScanner(t *testing.T) {
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
			name:  "erock: 3 lines",
			input: "Line 1\nLine 2\nLine 3\n",
			expected: []string{
				"Line 1",
				"Line 2",
				"Line 3",
			},
			width: 6,
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chars := vaxis.Characters(test.input)
			cells := make([]vaxis.Cell, 0, len(chars))
			for _, char := range chars {
				cell := vaxis.Cell{Character: char}
				cells = append(cells, cell)
			}

			scanner := NewSoftwrapScanner(cells, test.width)
			lines := []string{}
			for scanner.Scan() {
				line := scanner.Text()
				str := strings.Builder{}
				for _, ch := range line {
					str.WriteString(ch.Grapheme)
				}
				lines = append(lines, str.String())
			}

			assert.Equal(t, test.expected, lines)
		})
	}
}
