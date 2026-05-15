package term

import (
	"testing"
)

func BenchmarkScreenBuffer(b *testing.B) {
	b.Run("erase-row-range", func(b *testing.B) {
		screen := newScreenBuffer(160, 48)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			screen.eraseRowRange(24, 0, 159, 0)
		}
	})

	b.Run("scroll-up", func(b *testing.B) {
		screen := newScreenBuffer(160, 48)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			screen.scrollUp(0, 47, 0, 159, 1, 0)
		}
	})

	b.Run("scroll-down", func(b *testing.B) {
		screen := newScreenBuffer(160, 48)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			screen.scrollDown(0, 47, 0, 159, 1, 0)
		}
	})
}
