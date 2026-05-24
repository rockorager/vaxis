// vtwidth is a utility to measure the width of a string as it will be rendered
// in the terminal
package main

import (
	"fmt"
	"os"

	"go.rockorager.dev/vaxis"
)

type failure struct {
	input    string
	actual   int
	expected int
}

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer vx.Close()
	// Our test corpus
	cases := []string{
		"😀",
		"\u26A0\uFE0F", // VS16 selector
		"👩‍🚀",          // ZWJ
		"👋🏿",           // skin tone selector
		"🏳️‍🌈",         // VS16 and ZWJ
		"⚠️",
	}
	failures := []failure{}
	_, col := vx.CursorPosition()
	for _, c := range cases {
		w := vx.RenderedWidth(c)

		// out := "|" + strings.Repeat("-", w) + "|"
		fmt.Print(c)
		// fmt.Println("|" + c + "|")
		_, next := vx.CursorPosition()
		if w != (next - col) {
			failures = append(failures, failure{
				input:    c,
				actual:   next - col,
				expected: w,
			})
		}
		fmt.Println("")
	}
	vx.Close()
	for _, f := range failures {
		fmt.Printf("Test fail: %q: actual=%d, expected=%d\n", f.input, f.actual, f.expected)
	}
	if len(failures) > 0 {
		os.Exit(1)
	}
}
