package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func BenchmarkTerminalActions(b *testing.B) {
	benchmarks := []struct {
		name string
		seqs []ansi.Sequence
	}{
		{
			name: "plain",
			seqs: repeatedTerminalSequences(ansi.Print{Grapheme: "x", Width: 1}, 4096),
		},
		{
			name: "control",
			seqs: repeatedTerminalSequences([]ansi.Sequence{
				ansi.Print{Grapheme: "x", Width: 1},
				ansi.C0(0x08),
				ansi.Print{Grapheme: "y", Width: 1},
				ansi.C0(0x0D),
				ansi.C0(0x0A),
			}, 1024),
		},
		{
			name: "csi",
			seqs: repeatedTerminalSequences([]ansi.Sequence{
				ansi.Print{Grapheme: "x", Width: 1},
				benchCSI('D', []uint32{1}),
				benchCSI('P', []uint32{1}),
				benchCSI('X', []uint32{4}),
				benchCSI('H', []uint32{1, 1}),
			}, 1024),
		},
		{
			name: "mixed",
			seqs: repeatedTerminalSequences([]ansi.Sequence{
				ansi.Print{Grapheme: "x", Width: 1},
				ansi.C0(0x0A),
				benchCSI('m', []uint32{1}),
				ansi.Print{Grapheme: "y", Width: 1},
				benchCSI('m', []uint32{0}),
				benchCSI('H', []uint32{1, 1}),
				testESC('0', ')'),
				ansi.C0(0x0E),
				ansi.Print{Grapheme: "q", Width: 1},
				ansi.C0(0x0F),
			}, 1024),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			vt := New()
			vt.resize(80, 24)
			vt.primaryScreen.state.scrollbackLimit = 0
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.seqs)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, seq := range bm.seqs {
					applySequence(vt, seq)
				}
			}
		})
	}
}

func benchCSI(final rune, params []uint32, intermediates ...rune) ansi.CSI {
	var seq ansi.CSI
	seq.Final = final
	seq.NumParameters = len(params)
	copy(seq.Parameters[:], params)
	seq.NumIntermediate = len(intermediates)
	copy(seq.Intermediate[:], intermediates)
	return seq
}

func repeatedTerminalSequences(seq ansi.Sequence, n int) []ansi.Sequence {
	seqs, ok := seq.([]ansi.Sequence)
	if !ok {
		seqs = []ansi.Sequence{seq}
	}
	out := make([]ansi.Sequence, 0, len(seqs)*n)
	for i := 0; i < n; i++ {
		out = append(out, seqs...)
	}
	return out
}
