package term

import (
	"fmt"

	"go.rockorager.dev/vaxis/ansi"
)

const kittyKeyboardStackLen = 8

type kittyKeyboardStack struct {
	flags [kittyKeyboardStackLen]uint8
	idx   uint8
}

func (s *kittyKeyboardStack) current() uint8 {
	return s.flags[s.idx]
}

func (s *kittyKeyboardStack) push(flags uint8) {
	s.idx = (s.idx + 1) % kittyKeyboardStackLen
	s.flags[s.idx] = flags
}

func (s *kittyKeyboardStack) pop(n uint32) {
	if n >= kittyKeyboardStackLen {
		*s = kittyKeyboardStack{}
		return
	}
	for i := uint32(0); i < n; i++ {
		s.flags[s.idx] = 0
		s.idx = (s.idx + kittyKeyboardStackLen - 1) % kittyKeyboardStackLen
	}
}

func (s *kittyKeyboardStack) set(flags uint8) {
	s.flags[s.idx] = flags
}

func (s *kittyKeyboardStack) setOr(flags uint8) {
	s.flags[s.idx] |= flags
}

func (s *kittyKeyboardStack) setNot(flags uint8) {
	s.flags[s.idx] &^= flags
}

func (vt *Model) activeKittyKeyboard() *kittyKeyboardStack {
	if vt.mode.smcup {
		return &vt.altKittyKeyboard
	}
	return &vt.primaryKittyKeyboard
}

func (vt *Model) kittyKeyboardQuery() {
	if !vt.EnableKittyKeyboard {
		return
	}
	vt.enqueueReplyString(fmt.Sprintf("\x1b[?%du", vt.activeKittyKeyboard().current()))
}

func (vt *Model) kittyKeyboardPush(seq ansi.CSI) {
	if !vt.EnableKittyKeyboard {
		return
	}
	flags, ok := kittyKeyboardFlags(seq, seq.NumParameters == 1)
	if !ok {
		return
	}
	vt.activeKittyKeyboard().push(flags)
}

func (vt *Model) kittyKeyboardPop(seq ansi.CSI) {
	if !vt.EnableKittyKeyboard {
		return
	}
	n := uint32(1)
	if seq.NumParameters == 1 {
		n = seq.Parameters[0]
	}
	vt.activeKittyKeyboard().pop(n)
}

func (vt *Model) kittyKeyboardSet(seq ansi.CSI) {
	if !vt.EnableKittyKeyboard {
		return
	}
	flags, ok := kittyKeyboardFlags(seq, seq.NumParameters >= 1)
	if !ok {
		return
	}

	action := uint32(1)
	if seq.NumParameters >= 2 {
		action = seq.Parameters[1]
	}

	switch action {
	case 1:
		vt.activeKittyKeyboard().set(flags)
	case 2:
		vt.activeKittyKeyboard().setOr(flags)
	case 3:
		vt.activeKittyKeyboard().setNot(flags)
	}
}

func kittyKeyboardFlags(seq ansi.CSI, hasParam bool) (uint8, bool) {
	if !hasParam {
		return 0, true
	}
	flags := seq.Parameters[0]
	if flags > 31 {
		return 0, false
	}
	return uint8(flags), true
}
