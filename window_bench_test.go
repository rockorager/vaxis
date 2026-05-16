package vaxis

import "testing"

func benchmarkSetCellWindow(depth int) Window {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
		charCache:  make(map[string]int),
	}
	vx.screenNext.resize(200, 80)
	vx.screenLast.resize(200, 80)

	win := vx.Window()
	for i := 0; i < depth; i += 1 {
		win = win.New(1, 1, -1, -1)
	}
	return win
}

func benchmarkWindowSetCell(b *testing.B, depth int) {
	win := benchmarkSetCellWindow(depth)
	cell := Cell{Character: Character{Grapheme: "x", Width: 1}}
	width, height := win.Size()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.SetCell(i%width, (i/width)%height, cell)
	}
}

func BenchmarkWindowSetCellRoot(b *testing.B) {
	benchmarkWindowSetCell(b, 0)
}

func BenchmarkWindowSetCellDepth1(b *testing.B) {
	benchmarkWindowSetCell(b, 1)
}

func BenchmarkWindowSetCellDepth4(b *testing.B) {
	benchmarkWindowSetCell(b, 4)
}

func benchmarkWindowSetStyle(b *testing.B, depth int) {
	win := benchmarkSetCellWindow(depth)
	style := Style{Attribute: AttrBold}
	width, height := win.Size()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.SetStyle(i%width, (i/width)%height, style)
	}
}

func BenchmarkWindowSetStyleRoot(b *testing.B) {
	benchmarkWindowSetStyle(b, 0)
}

func BenchmarkWindowSetStyleDepth1(b *testing.B) {
	benchmarkWindowSetStyle(b, 1)
}

func BenchmarkWindowSetStyleDepth4(b *testing.B) {
	benchmarkWindowSetStyle(b, 4)
}

func BenchmarkWindowPrintTruncate(b *testing.B) {
	win := benchmarkSetCellWindow(0)
	seg := Segment{Text: "abcdefghijklmnopqrstuvwxyz0123456789"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.PrintTruncate(0, seg)
	}
}

func BenchmarkWindowPrint(b *testing.B) {
	win := benchmarkSetCellWindow(0)
	seg := Segment{Text: "abcdefghijklmnopqrstuvwxyz0123456789"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.Print(seg)
	}
}

func BenchmarkWindowPrintln(b *testing.B) {
	win := benchmarkSetCellWindow(0)
	seg := Segment{Text: "abcdefghijklmnopqrstuvwxyz0123456789"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.Println(0, seg)
	}
}

func BenchmarkWindowWrap(b *testing.B) {
	win := benchmarkSetCellWindow(0)
	seg := Segment{Text: "abcdefghijklmnopqrstuvwxyz 0123456789 abcdefghijklmnopqrstuvwxyz"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		win.Wrap(seg)
	}
}
