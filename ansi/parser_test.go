package ansi

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func escSeq(final rune, intermediates string) ESC {
	seq := ESC{Final: final, NumIntermediate: len([]rune(intermediates))}
	copy(seq.Intermediate[:], []rune(intermediates))
	return seq
}

func TestParserInputModeEmitsBareEscapeAfterTimeout(t *testing.T) {
	r, w := io.Pipe()
	parse := NewParser(r, ParserModeInput)
	defer r.Close()
	defer w.Close()

	if _, err := w.Write([]byte{0x1B}); err != nil {
		t.Fatal(err)
	}

	select {
	case seq := <-parse.Next():
		if got, ok := seq.(C0); !ok || got != C0(0x1B) {
			t.Fatalf("sequence = %#v, want ESC C0", seq)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for input ESC timeout")
	}
}

func TestParserOutputModeDoesNotEmitBareEscapeAfterTimeout(t *testing.T) {
	r, w := io.Pipe()
	parse := NewParser(r, ParserModeOutput)
	defer r.Close()
	defer w.Close()

	if _, err := w.Write([]byte{0x1B}); err != nil {
		t.Fatal(err)
	}

	select {
	case seq := <-parse.Next():
		t.Fatalf("unexpected sequence before output EOF: %#v", seq)
	case <-time.After(30 * time.Millisecond):
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case seq := <-parse.Next():
		if _, ok := seq.(EOF); !ok {
			t.Fatalf("sequence after output close = %#v, want EOF", seq)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for output parser EOF")
	}
}

func csiSeq(final rune, intermediates string, params []int, colonAfter ...int) CSI {
	seq := CSI{
		Final:           final,
		NumIntermediate: len([]rune(intermediates)),
		NumParameters:   len(params),
	}
	copy(seq.Intermediate[:], []rune(intermediates))
	if len(params) <= InlineCSIParams {
		for i, param := range params {
			seq.Parameters[i] = uint32(param)
		}
	} else {
		seq.ExtraParameters = make([]uint32, len(params))
		for i, param := range params {
			seq.ExtraParameters[i] = uint32(param)
		}
	}
	for _, idx := range colonAfter {
		seq.ColonSeparators |= 1 << uint(idx)
	}
	return seq
}

func dcsSeq(final rune, intermediates string, params []int, data string) DCS {
	seq := DCS{
		Final:           final,
		NumIntermediate: len([]rune(intermediates)),
		NumParameters:   len(params),
		Data:            []rune(data),
	}
	copy(seq.Intermediate[:], []rune(intermediates))
	if len(params) <= InlineCSIParams {
		for i, param := range params {
			seq.Parameters[i] = uint32(param)
		}
	} else {
		seq.ExtraParameters = make([]uint32, len(params))
		for i, param := range params {
			seq.ExtraParameters[i] = uint32(param)
		}
	}
	return seq
}

func TestUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "UTF-8",
			input: "🔥",
			expected: []Sequence{
				Print{"🔥", 2},
			},
		},
		{
			name:  "UTF-8",
			input: "👩‍🚀",
			expected: []Sequence{
				Print{"👩‍🚀", 2},
			},
		},
		{
			name:  "ASCII with combining mark",
			input: "a\u0301b",
			expected: []Sequence{
				Print{"a\u0301", 1},
				Print{"b", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestC1Controls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "CSI",
			input: "a\u009b2J",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('J', "", []int{2}),
			},
		},
		{
			name:  "CSI raw byte",
			input: "a\x9b2J",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('J', "", []int{2}),
			},
		},
		{
			name:  "OSC",
			input: "a\u009d2;hi\u009c",
			expected: []Sequence{
				Print{"a", 1},
				OSC{Payload: []rune("2;hi")},
			},
		},
		{
			name:  "OSC raw byte",
			input: "a\x9d2;hi\x9c",
			expected: []Sequence{
				Print{"a", 1},
				OSC{Payload: []rune("2;hi")},
			},
		},
		{
			name:  "OSC invalid UTF-8",
			input: "a\x1b]2;abc\xc0\x1b\\",
			expected: []Sequence{
				Print{"a", 1},
				OSC{Payload: []rune{'2', ';', 'a', 'b', 'c', 0xC0}, InvalidUTF8: true},
			},
		},
		{
			name:  "ST clears string state",
			input: "\u009d2;hi\u009c\x1b\\",
			expected: []Sequence{
				OSC{Payload: []rune("2;hi")},
				escSeq('\\', ""),
			},
		},
		{
			name:  "DCS",
			input: "a\u0090+qdata\u009c",
			expected: []Sequence{
				Print{"a", 1},
				dcsSeq('q', "+", nil, "data"),
			},
		},
		{
			name:  "APC",
			input: "a\u009fdata\u009c",
			expected: []Sequence{
				Print{"a", 1},
				APC{Data: "data"},
			},
		},
		{
			name:  "APC empty ST consumed",
			input: "a\x1b_\x1b\\b",
			expected: []Sequence{
				Print{"a", 1},
				APC{},
				Print{"b", 1},
			},
		},
		{
			name:  "NEL",
			input: "a\u0085",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('E', ""),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r, ParserModeOutput)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestSOSPMControlStringsAreIgnored(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "C1 PM",
			input: "\x9eignored\x9ca\x19",
		},
		{
			name:  "C1 SOS",
			input: "\x98ignored\x9ca\x19",
		},
		{
			name:  "7-bit PM",
			input: "\x1b^ignored\x1b\\a\x19",
		},
		{
			name:  "7-bit SOS",
			input: "\x1bXignored\x1b\\a\x19",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parse := NewParser(strings.NewReader(test.input), ParserModeOutput)
			expected := []Sequence{
				Print{"a", 1},
				C0(0x19),
			}

			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(expected), i, "wrong amount of sequences")
					break
				}
				if i < len(expected) {
					assert.Equal(t, expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestCSIWithManySGRParameters(t *testing.T) {
	parse := NewParser(strings.NewReader("\x1b[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136;0m"), ParserModeOutput)

	seq := <-parse.Next()
	csi, ok := seq.(CSI)
	if !ok {
		t.Fatalf("sequence = %T, want CSI", seq)
	}
	if got, want := csi.Final, rune('m'); got != want {
		t.Fatalf("final = %q, want %q", got, want)
	}
	if got, want := csi.NumParameters, 18; got != want {
		t.Fatalf("parameter count = %d, want %d", got, want)
	}
	want := []uint32{4, 3, 38, 2, 51, 51, 51, 48, 2, 170, 170, 170, 58, 2, 255, 97, 136, 0}
	if got := csi.Params(); !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %v, want %v", got, want)
	}
	if !csi.ColonAfter(0) {
		t.Fatal("colon separator after first SGR parameter was not preserved")
	}
}

func TestCSISGRColonParameterVariants(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		params     []int
		colonAfter []int
	}{
		{
			name:       "foreground mode",
			input:      "\x1b[38:2m",
			params:     []int{38, 2},
			colonAfter: []int{0},
		},
		{
			name:       "indexed foreground and background",
			input:      "\x1b[38:5:1;48:5:0m",
			params:     []int{38, 5, 1, 48, 5, 0},
			colonAfter: []int{0, 1, 3, 4},
		},
		{
			name:       "direct background",
			input:      "\x1b[48:2:240:143:104m",
			params:     []int{48, 2, 240, 143, 104},
			colonAfter: []int{0, 1, 2, 3},
		},
		{
			name:       "underline style",
			input:      "\x1b[4:3m",
			params:     []int{4, 3},
			colonAfter: []int{0},
		},
		{
			name:       "blank color space",
			input:      "\x1b[58:2::240:143:104m",
			params:     []int{58, 2, 0, 240, 143, 104},
			colonAfter: []int{0, 1, 2, 3, 4},
		},
		{
			name:       "mixed blank and semicolon",
			input:      "\x1b[;4:3;38;2;175;175;215;58:2::190:80:70m",
			params:     []int{0, 4, 3, 38, 2, 175, 175, 215, 58, 2, 0, 190, 80, 70},
			colonAfter: []int{1, 8, 9, 10, 11, 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parse := NewParser(strings.NewReader(tt.input), ParserModeOutput)

			seq := <-parse.Next()
			csi, ok := seq.(CSI)
			if !ok {
				t.Fatalf("sequence = %T, want CSI", seq)
			}
			if got, want := csi.Final, rune('m'); got != want {
				t.Fatalf("final = %q, want %q", got, want)
			}
			if got, want := csi.NumParameters, len(tt.params); got != want {
				t.Fatalf("parameter count = %d, want %d", got, want)
			}
			wantParams := make([]uint32, len(tt.params))
			for i, param := range tt.params {
				wantParams[i] = uint32(param)
			}
			if got := csi.Params(); !reflect.DeepEqual(got, wantParams) {
				t.Fatalf("params = %v, want %v", got, wantParams)
			}
			for _, idx := range tt.colonAfter {
				if !csi.ColonAfter(idx) {
					t.Fatalf("colon separator after param %d was not preserved", idx)
				}
			}
		})
	}
}

func TestCSIColonParametersWithNonSGRFinalAreIgnored(t *testing.T) {
	parse := NewParser(strings.NewReader("\x1b[38:2h"), ParserModeOutput)

	for {
		seq := <-parse.Next()
		switch seq.(type) {
		case CSI:
			t.Fatalf("unexpected CSI sequence with colon parameters and non-SGR final: %#v", seq)
		case EOF:
			return
		}
	}
}

func TestCSIParameterSaturatesWhenTooLong(t *testing.T) {
	parse := NewParser(strings.NewReader("\x1b[999999999999999999999999999999999999999999999999999999999999999C"), ParserModeOutput)

	seq := <-parse.Next()
	csi, ok := seq.(CSI)
	if !ok {
		t.Fatalf("sequence = %#v, want CSI", seq)
	}
	if got, want := csi.Param(0), int(^uint32(0)); got != want {
		t.Fatalf("CSI param = %d, want saturated %d", got, want)
	}
}

func TestCSIWithMaxParametersDispatches(t *testing.T) {
	input := "\x1b[" + strings.Repeat("1;", MaxCSIParams-1) + "2H"
	parse := NewParser(strings.NewReader(input), ParserModeOutput)

	seq := <-parse.Next()
	csi, ok := seq.(CSI)
	if !ok {
		t.Fatalf("sequence = %T, want CSI", seq)
	}
	if got, want := csi.NumParameters, MaxCSIParams; got != want {
		t.Fatalf("parameter count = %d, want %d", got, want)
	}
	if got, want := csi.Param(MaxCSIParams-1), 2; got != want {
		t.Fatalf("last parameter = %d, want %d", got, want)
	}
}

func TestCSIWithTooManyParametersIsIgnored(t *testing.T) {
	input := "\x1b[" + strings.Repeat("1;", MaxCSIParams) + "2H"
	parse := NewParser(strings.NewReader(input), ParserModeOutput)

	for {
		seq := <-parse.Next()
		switch seq.(type) {
		case CSI:
			t.Fatalf("unexpected CSI sequence after parameter overflow: %#v", seq)
		case EOF:
			return
		}
	}
}

func TestCSIGhosttyParserShapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CSI
	}{
		{
			name:     "DECRQM",
			input:    "\x1b[?2026$p",
			expected: csiSeq('p', "?$", []int{2026}),
		},
		{
			name:     "cursor style",
			input:    "\x1b[3 q",
			expected: csiSeq('q', " ", []int{3}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parse := NewParser(strings.NewReader(test.input), ParserModeOutput)

			seq := <-parse.Next()
			csi, ok := seq.(CSI)
			if !ok {
				t.Fatalf("sequence = %#v, want CSI", seq)
			}
			assert.Equal(t, test.expected, csi)
		})
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name     string
		inRange  []rune
		input    rune
		expected bool
	}{
		{
			name:     "endpoint min",
			inRange:  []rune{0x00, 0x20},
			input:    0x00,
			expected: true,
		},
		{
			name:     "endpoint max",
			inRange:  []rune{0x00, 0x20},
			input:    0x20,
			expected: true,
		},
		{
			name:     "within",
			inRange:  []rune{0x00, 0x20},
			input:    0x19,
			expected: true,
		},
		{
			name:     "outside",
			inRange:  []rune{0x00, 0x20},
			input:    0x21,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := in(test.input, test.inRange[0], test.inRange[1])
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestAnywhere(t *testing.T) {
	tests := []struct {
		expected stateFn
		name     string
		input    rune
	}{
		{
			name:     "0x18",
			input:    0x18,
			expected: ground,
		},
		{
			name:     "0x1A",
			input:    0x1A,
			expected: ground,
		},
		{
			name:     "0x1B",
			input:    0x1B,
			expected: escape,
		},
		{
			name:     "eof",
			input:    eof,
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := bytes.NewBuffer(nil)
			parse := NewParser(r)
			called := false
			parse.exit = func() {
				called = true
			}
			actual := anywhere(test.input, parse)
			act := reflect.ValueOf(actual).Pointer()
			exp := reflect.ValueOf(test.expected).Pointer()
			assert.Equal(t, exp, act, "wrong return function")
			if test.expected != nil {
				assert.True(t, called, "exit function not called")
			}
		})
	}
}

func TestCSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "CSI Entry + C0",
			input: "a\x1b[\x00",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
			},
		},
		{
			name:  "CSI Entry + escape",
			input: "a\x1b[\x1b",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Entry + ignore",
			input: "a\x1b[\x7F",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Entry + dispatch",
			input: "a\x1b[c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", nil),
			},
		},
		{
			name:  "CSI Param with collect first",
			input: "a\x1b[<c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "<", nil),
			},
		},
		{
			name:  "CSI Param with colorspace",
			input: "a\x1b[38:2::0:0:0m",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('m', "", []int{38, 2, 0, 0, 0, 0}, 0, 1, 2, 3, 4),
			},
		},
		{
			name:  "CSI Param with colorspace fg and bg",
			input: "a\x1b[38:2::0:0:0;48:2::0:0:0m",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('m', "", []int{38, 2, 0, 0, 0, 0, 48, 2, 0, 0, 0, 0}, 0, 1, 2, 3, 4, 6, 7, 8, 9, 10),
			},
		},
		{
			name:  "CSI Param SGR with semicolons",
			input: "a\x1b[38;2;0;0;0;48;2;0;0;0m",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('m', "", []int{38, 2, 0, 0, 0, 48, 2, 0, 0, 0}),
			},
		},
		{
			name:  "CSI Param",
			input: "a\x1b[0c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", []int{0}),
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0\x00",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0\x7F\x00",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
			},
		},
		{
			name:  "CSI Param with long param",
			input: "a\x1b[9999c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", []int{9999}),
			},
		},
		{
			name:  "CSI Param with multiple",
			input: "a\x1b[0;0c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", []int{0, 0}),
			},
		},
		{
			name:  "CSI Param with multiple blank",
			input: "a\x1b[;c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", []int{0, 0}),
			},
		},
		{
			name:  "CSI Param with multiple filled or blank",
			input: "a\x1b[;1c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "", []int{0, 1}),
			},
		},
		{
			name:  "CSI Param + csiIgnore",
			input: "a\x1b[;1\x3Cc",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Param + escape",
			input: "a\x1b[;1\x1b",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Intermediate",
			input: "a\x1b[\x20\x20c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "  ", nil),
			},
		},
		{
			name:  "CSI Intermediate + escape",
			input: "a\x1b[\x20\x20\x1b",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Intermediate + c0",
			input: "a\x1b[\x20\x20\x00c",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
				csiSeq('c', "  ", nil),
			},
		},
		{
			name:  "CSI Intermediate + 7f ignore",
			input: "a\x1b[\x20\x20\x7Fc",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "  ", nil),
			},
		},
		{
			name:  "CSI Intermediate + eof",
			input: "a\x1b[\x20\x20",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Intermediate + param",
			input: "a\x1b[0\x20\x20c",
			expected: []Sequence{
				Print{"a", 1},
				csiSeq('c', "  ", []int{0}),
			},
		},
		{
			name:  "CSI Intermediate + param + ignore",
			input: "a\x1b[0\x20\x20\x30c",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Ignore + eof",
			input: "a\x1b[0\x20\x20\x30\x3A",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Ignore + esc",
			input: "a\x1b[0\x20\x20\x30\x1B",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "CSI Ignore + c0",
			input: "a\x1b[0\x20\x20\x30\x00c",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
			},
		},
		{
			name:  "CSI Ignore + 7F ignore",
			input: "a\x1b[0\x20\x20\x30\x7Fc",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				t.Logf("%T", seq)
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestDCS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "DCS Entry + C0",
			input: "a\x1bP\x00",
			expected: []Sequence{
				Print{"a", 1},
			},
		},
		{
			name:  "DCS Entry + end",
			input: "a\x1bPq",
			expected: []Sequence{
				Print{"a", 1},
				dcsSeq('q', "", nil, ""),
			},
		},
		{
			name:  "DCS Entry + data + end",
			input: "a\x1bPq#0;2;0;\x1b\\",
			expected: []Sequence{
				Print{"a", 1},
				dcsSeq('q', "", nil, "#0;2;0;"),
			},
		},
		{
			name:  "DCS empty ST consumed",
			input: "a\x1bPq\x1b\\b",
			expected: []Sequence{
				Print{"a", 1},
				dcsSeq('q', "", nil, ""),
				Print{"b", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestDCSWithTooManyParametersIsIgnored(t *testing.T) {
	input := "\x1bP6" + strings.Repeat(";", MaxCSIParams) + "7pignored\x1b\\"
	parser := NewParser(strings.NewReader(input))

	for {
		seq := <-parser.Next()
		switch seq.(type) {
		case DCS:
			t.Fatalf("unexpected DCS sequence after parameter overflow: %#v", seq)
		case EOF:
			return
		}
	}
}

func TestDCSGhosttyParserShapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DCS
	}{
		{
			name:     "XTGETTCAP",
			input:    "\x1bP+q\x1b\\",
			expected: dcsSeq('q', "+", nil, ""),
		},
		{
			name:     "params",
			input:    "\x1bP1000p\x1b\\",
			expected: dcsSeq('p', "", []int{1000}, ""),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parse := NewParser(strings.NewReader(test.input), ParserModeOutput)

			seq := <-parse.Next()
			dcs, ok := seq.(DCS)
			if !ok {
				t.Fatalf("sequence = %#v, want DCS", seq)
			}
			assert.Equal(t, test.expected, dcs)
		})
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "ESC W",
			input: "a\x1bDc",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('D', ""),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC W",
			input: "a\x1bWc",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('W', ""),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC W with a C0",
			input: "a\x1b\x00Wc",
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
				escSeq('W', ""),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC Backspace",
			input: "a\x1b\x7f",
			expected: []Sequence{
				Print{"a", 1},
				escSeq(0x7F, ""),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "fewer sequences than expected")
					break
				}
				assert.Equal(t, test.expected[i], seq)
				i += 1
				assert.LessOrEqual(t, i, len(test.expected), "more sequences than expected")
			}
		})
	}
}

func TestEscapeIntermediate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "ESC SP F",
			input: "a\x1b Fc",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('F', " "),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC # 3",
			input: "a\x1b#3c",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('3', "#"),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC ( B",
			input: "a\x1b(Bc",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('B', "("),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC ( B with C0",
			input: "a\x1b(\tBc",
			expected: []Sequence{
				Print{"a", 1},
				C0('\t'),
				escSeq('B', "("),
				Print{"c", 1},
			},
		},
		{
			name:  "ESC ( B with ignore",
			input: "a\x1b(\x7FBc",
			expected: []Sequence{
				Print{"a", 1},
				escSeq('B', "("),
				Print{"c", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestGround(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "printables",
			input: "abc",
			expected: []Sequence{
				Print{"a", 1},
				Print{"b", 1},
				Print{"c", 1},
			},
		},
		{
			name:  "printable with c0",
			input: string([]rune{'a', 0x00, 'c'}),
			expected: []Sequence{
				Print{"a", 1},
				C0(0x00),
				Print{"c", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					break
				}
				assert.Equal(t, test.expected[i], seq)
				i += 1
			}
		})
	}
}

func TestOSC(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "OSC entry",
			input: "a\x1b\x5D",
			expected: []Sequence{
				Print{"a", 1},
				OSC{},
			},
		},
		{
			name:  "OSC end ST",
			input: "a\x1B\x5D\x1B\x5C",
			expected: []Sequence{
				Print{"a", 1},
				OSC{},
			},
		},
		{
			name:  "OSC end CAN",
			input: "a\x1B\x5D\x1B\x18",
			expected: []Sequence{
				Print{"a", 1},
				OSC{},
				C0(0x18),
			},
		},
		{
			name:  "OSC end SUB",
			input: "a\x1B\x5D\x1B\x1A",
			expected: []Sequence{
				Print{"a", 1},
				OSC{},
				C0(0x1A),
			},
		},
		{
			name:  "OSC 8 ;; http://example.com",
			input: "a\x1B\x5D8;;http://example.com\x1b\x5CLink\x1b\x5D8;;\x1b\x5C",
			expected: []Sequence{
				Print{"a", 1},
				OSC{
					Payload: []rune{
						'8',
						';',
						';',
						'h',
						't',
						't',
						'p',
						':',
						'/',
						'/',
						'e',
						'x',
						'a',
						'm',
						'p',
						'l',
						'e',
						'.',
						'c',
						'o',
						'm',
					},
				},
				Print{"L", 1},
				Print{"i", 1},
				Print{"n", 1},
				Print{"k", 1},
				OSC{
					Payload: []rune{
						'8',
						';',
						';',
					},
				},
			},
		},
		{
			name:  "OSC bell terminated",
			input: "a\x1B\x5D\ab",
			expected: []Sequence{
				Print{"a", 1},
				OSC{},
				Print{"b", 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := <-parse.Next()
				if _, ok := seq.(EOF); ok {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}
