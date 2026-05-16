package vaxis

import "testing"

func benchmarkScreenResize(b *testing.B, cols int, rows int) {
	s := newScreen()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		s.resize(cols, rows)
	}
}

func BenchmarkScreenResize80x24(b *testing.B) {
	benchmarkScreenResize(b, 80, 24)
}

func BenchmarkScreenResize200x60(b *testing.B) {
	benchmarkScreenResize(b, 200, 60)
}
