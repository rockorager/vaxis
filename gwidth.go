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
		if r >= 0xFE00 && r <= 0xFE0F {
			// Variation Selectors 1 - 16
			continue
		}
		if r >= 0xE0100 && r <= 0xE01EF {
			// Variation Selectors 17-256
			continue
		}
		total += runewidth.RuneWidth(r)
	}
	return total
}
