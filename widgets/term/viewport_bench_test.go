package term

import "testing"

var viewportBenchSink int

func BenchmarkViewport(b *testing.B) {
	b.Run("visible-active", func(b *testing.B) {
		vt := benchmarkViewportModel(1000)
		vt.scrollOffset = 0

		b.ReportAllocs()
		b.ResetTimer()
		sum := 0
		for i := 0; i < b.N; i++ {
			line := vt.visibleLine(i % vt.height())
			sum += len(line)
		}
		viewportBenchSink = sum
	})

	b.Run("visible-scrollback", func(b *testing.B) {
		vt := benchmarkViewportModel(1000)
		vt.scrollOffset = 500

		b.ReportAllocs()
		b.ResetTimer()
		sum := 0
		for i := 0; i < b.N; i++ {
			line := vt.visibleLine(i % vt.height())
			sum += len(line)
		}
		viewportBenchSink = sum
	})

	b.Run("scroll", func(b *testing.B) {
		vt := benchmarkViewportModel(1000)
		vt.scrollOffset = 500

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vt.scrollViewport(1)
			vt.scrollViewport(-1)
		}
	})
}

func benchmarkViewportModel(scrollback int) *Model {
	vt := New()
	vt.resize(160, 48)
	for i := 0; i < vt.height(); i += 1 {
		setScreenLine(vt.primaryScreen, i, "benchmark viewport line")
	}
	for i := 0; i < scrollback; i += 1 {
		vt.scrollUp(1)
		setScreenLine(vt.primaryScreen, vt.height()-1, "benchmark viewport line")
	}
	return vt
}
