package vaxis

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

type graphemeWidthMethod int

const (
	wcwidth graphemeWidthMethod = iota
	noZWJ                       // we do unicodeStd but not if there is a ZWJ
	unicodeStd
)

func gwidth(s string, method graphemeWidthMethod) int {
	switch method {
	case noZWJ:
		s = strings.ReplaceAll(s, "\u200D", "")
		return uniseg.StringWidth(s)
	case unicodeStd:
		return uniseg.StringWidth(s)
	default:
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
}
