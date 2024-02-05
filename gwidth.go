package vaxis

import (
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

func gwidth(s string, unicode bool) int {
	if unicode {
		return uniseg.StringWidth(s)
	}
	total := 0
	for _, r := range s {
		if r == 0xFE0F {
			// runewidth incorrectly thinks a VS16 selector is 1
			// wide
			continue
		}
		total += runewidth.RuneWidth(r)
	}
	return total
}
