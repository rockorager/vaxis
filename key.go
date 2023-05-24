package rtk

import (
	"fmt"
	"strings"
)

type Key struct {
	codepoint rune
	modifiers ModifierMask
}

type ModifierMask int

const (
	// Values equivalent to kitty keyboard protocol
	ModShift ModifierMask = 1 << iota
	ModAlt
	ModCtrl
	ModSuper
	ModHyper
	ModMeta
	ModCapsLock
	ModNumLock
)

// Modified keys will always have prefixes in this order:
//
//	<num-caps-meta-hyper-super-c-a-s-{key}>
func (k Key) String() string {
	switch {
	case k.codepoint < 0x00:
		return "<invalid>"
	case k.codepoint < 0x20:
		ch := fmt.Sprintf("%c", k.codepoint+0x40)
		return fmt.Sprintf("<c-%s>", strings.ToLower(ch))
	default:
		return fmt.Sprintf("%c", k.codepoint)
	}
}

const (
	synthetic rune = 1 << 30
)
