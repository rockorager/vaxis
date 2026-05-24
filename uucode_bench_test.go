package vaxis_test

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/rockorager/go-uucode"
)

var benchmarkUucodeResult []string
var benchmarkUucodeWidthResult int
var benchmarkCharactersResult []vaxis.Character

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

	b.Run("graphemes with string width", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			width := 0
			g := uucode.NewGraphemeIterator(testString)
			for cluster, ok := g.Next(); ok; cluster, ok = g.Next() {
				grapheme := testString[cluster.Start:cluster.End]
				result = append(result, grapheme)
				width += uucode.StringWidth(grapheme)
			}
			benchmarkUucodeResult = result
			benchmarkUucodeWidthResult = width
		}
	})

	b.Run("graphemes with width iterator", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			width := 0
			g := uucode.NewGraphemeWidthIterator(testString)
			for cluster, ok := g.Next(); ok; cluster, ok = g.Next() {
				result = append(result, testString[cluster.Start:cluster.End])
				width += cluster.Width
			}
			benchmarkUucodeResult = result
			benchmarkUucodeWidthResult = width
		}
	})
}

func BenchmarkCharacters(b *testing.B) {
	const testString = "ASCII A\u0300 👩🏽‍🚀 🇨🇭 क्‍ष 한글 😀 _ end\twith tab"

	b.ReportAllocs()
	for i := 0; i < b.N; i += 1 {
		benchmarkCharactersResult = vaxis.Characters(testString)
	}
}
