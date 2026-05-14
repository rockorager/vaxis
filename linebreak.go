package vaxis

import (
	"unicode/utf8"

	"github.com/rockorager/go-uucode"
)

func hasTrailingLineBreakInString(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)
	switch uucode.LineBreak(r) {
	case uucode.LineBreakBK, uucode.LineBreakCR, uucode.LineBreakLF, uucode.LineBreakNL:
		return true
	default:
		return false
	}
}
