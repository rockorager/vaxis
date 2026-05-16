package ansi

import (
	"strings"
	"testing"
)

var parserBenchInputs = map[string]string{
	"plain": strings.Repeat("the quick brown fox jumps over the lazy dog\n", 256),
	"control": strings.Repeat(
		"\x1b[38;2;12;34;56m\x1b[0m\x1b[10;20H\x1b[2K\x1b[?25l\x1b[?25h",
		512,
	),
	"csi": strings.Repeat(
		"\x1b[38;2;12;34;56mcolor\x1b[0m\x1b[10;20Hpos\x1b[2K",
		512,
	),
	"mixed": strings.Repeat(
		"prompt \x1b[1;32muser@host\x1b[0m \x1b[?25l\x1b[2Koutput\r\n",
		512,
	),
}

func BenchmarkParser(b *testing.B) {
	for name, input := range parserBenchInputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parser := NewParser(strings.NewReader(input), ParserModeOutput)
				for seq := range parser.Next() {
					if _, ok := seq.(EOF); ok {
						break
					}
				}
			}
		})
	}
}
