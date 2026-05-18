package vaxis_test

import (
	"strings"
	"testing"

	"github.com/rockorager/go-uucode"
)

var benchmarkUucodeResult []string

func BenchmarkUucode(b *testing.B) {
	const testString = "😀🔮🌏📝test string"

	b.Run("rune reader", func(b *testing.B) {
		// Just so we can see the penalty.
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			r := strings.NewReader(testString)
			for {
				ch, _, err := r.ReadRune()
				if err != nil {
					break
				}
				result = append(result, string(ch))
			}
			benchmarkUucodeResult = result
		}
	})

	b.Run("graphemes", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			g := uucode.NewGraphemeIterator(testString)
			for cluster, ok := g.Next(); ok; cluster, ok = g.Next() {
				result = append(result, testString[cluster.Start:cluster.End])
			}
			benchmarkUucodeResult = result
		}
	})
}
