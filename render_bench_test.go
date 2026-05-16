package vaxis

import (
	"bytes"
	"io"
	"testing"
)

func benchmarkRenderRefreshRGB(b *testing.B, cols int, rows int) {
	vx := &Vaxis{}
	vx.screenNext = newScreen()
	vx.screenLast = newScreen()
	vx.screenNext.resize(cols, rows)
	vx.screenLast.resize(cols, rows)
	vx.caps.rgb = true
	vx.caps.styledUnderlines = true
	vx.tw = &writer{
		buf:      bytes.NewBuffer(make([]byte, 0, 1<<20)),
		terminal: &terminalWriter{w: io.Discard},
		vx:       vx,
	}

	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			vx.screenNext.setCellDirect(col, row, Cell{
				Character: Character{Grapheme: "a", Width: 1},
				Style: Style{
					Foreground:     RGBColor(uint8(col), uint8(row), uint8(col+row)),
					Background:     RGBColor(uint8(row), uint8(col), uint8(row+1)),
					UnderlineColor: RGBColor(uint8(col+1), uint8(row+1), uint8(col)),
					UnderlineStyle: UnderlineSingle,
				},
			})
		}
	}

	vx.refresh = true
	b.ReportAllocs()
	b.SetBytes(int64(cols * rows))
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		vx.render()
		vx.tw.buf.Reset()
	}
}

func BenchmarkRenderRefreshRGB80x24(b *testing.B) {
	benchmarkRenderRefreshRGB(b, 80, 24)
}

func BenchmarkRenderRefreshRGB200x60(b *testing.B) {
	benchmarkRenderRefreshRGB(b, 200, 60)
}

func benchmarkRenderPartialRGB(b *testing.B, cols int, rows int, dirtyPct int) {
	vx := &Vaxis{}
	vx.screenNext = newScreen()
	vx.screenLast = newScreen()
	vx.screenNext.resize(cols, rows)
	vx.screenLast.resize(cols, rows)
	vx.caps.rgb = true
	vx.caps.styledUnderlines = true
	vx.tw = &writer{
		buf:      bytes.NewBuffer(make([]byte, 0, 1<<20)),
		terminal: &terminalWriter{w: io.Discard},
		vx:       vx,
	}

	base := Cell{
		Character: Character{Grapheme: "a", Width: 1},
		Style: Style{
			Foreground:     RGBColor(32, 64, 96),
			Background:     RGBColor(8, 16, 24),
			UnderlineColor: RGBColor(24, 48, 72),
			UnderlineStyle: UnderlineSingle,
		},
	}
	dirty := Cell{
		Character: Character{Grapheme: "b", Width: 1},
		Style: Style{
			Foreground:     RGBColor(196, 128, 16),
			Background:     RGBColor(24, 8, 120),
			UnderlineColor: RGBColor(220, 220, 40),
			UnderlineStyle: UnderlineDouble,
		},
	}

	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			vx.screenNext.setCellDirect(col, row, base)
			vx.screenLast.setCellDirect(col, row, base)
		}
	}

	totalCells := cols * rows
	dirtyCells := (totalCells * dirtyPct) / 100
	if dirtyCells < 1 {
		dirtyCells = 1
	}
	positions := make([]int, dirtyCells)
	step := totalCells / dirtyCells
	if step < 1 {
		step = 1
	}
	for i := 0; i < dirtyCells; i += 1 {
		positions[i] = (i * step) % totalCells
	}

	vx.refresh = false
	b.ReportAllocs()
	b.SetBytes(int64(dirtyCells))
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		cell := dirty
		if i%2 == 0 {
			cell = base
		}
		for _, pos := range positions {
			row := pos / cols
			col := pos % cols
			vx.screenNext.setCellDirect(col, row, cell)
		}
		vx.render()
		vx.tw.buf.Reset()
	}
}

func BenchmarkRenderPartialRGB80x24Dirty10Pct(b *testing.B) {
	benchmarkRenderPartialRGB(b, 80, 24, 10)
}

func BenchmarkRenderPartialRGB200x60Dirty10Pct(b *testing.B) {
	benchmarkRenderPartialRGB(b, 200, 60, 10)
}
