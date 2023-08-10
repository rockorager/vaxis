package vaxis_test

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func BenchmarkUniseg(b *testing.B) {
	const testString = "😀🔮🌏📝test string"

	b.Run("rune reader", func(b *testing.B) {
		// Just so we can see the penalty
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
		}
	})

	b.Run("graphemes", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			g := uniseg.NewGraphemes(testString)
			for g.Next() {
				result = append(result, g.Str())
			}
		}
	})
	b.Run("step", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := [][]byte{}
			in := []byte(testString)
			state := -1
			cluster := []byte{}
			for len(in) > 0 {
				cluster, in, _, state = uniseg.Step(in, state)
				result = append(result, cluster)
			}
		}
	})
	b.Run("stepstring", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			in := testString
			state := -1
			cluster := ""
			for len(in) > 0 {
				cluster, in, _, state = uniseg.StepString(in, state)
				result = append(result, cluster)
			}
		}
	})
	b.Run("firstfunction", func(b *testing.B) {
		for i := 0; i < b.N; i += 1 {
			result := []string{}
			in := testString
			state := -1
			cluster := ""
			for len(in) > 0 {
				cluster, in, _, state = uniseg.FirstGraphemeClusterInString(in, state)
				result = append(result, cluster)
			}
		}
	})
}
