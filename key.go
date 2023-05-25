package rtk

import (
	"fmt"
	"strings"
)

type Key struct {
	Codepoint rune
	Modifiers ModifierMask
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
	case k.Codepoint < 0x00:
		return "<invalid>"
	case k.Codepoint < 0x20:
		ch := fmt.Sprintf("%c", k.Codepoint+0x40)
		return fmt.Sprintf("<c-%s>", strings.ToLower(ch))
	default:
		return fmt.Sprintf("%c", k.Codepoint)
	}
}

const (
	extended rune = 1 << 30
)

const (
	KeyUp rune = extended + 1 + iota
	KeyRight
	KeyDown
	KeyLeft
	KeyInsert
	KeyDelete
	KeyBackspace
	KeyPgDown
	KeyPgUp
	KeyHome
	KeyEnd
	KeyF00
	KeyF01
	KeyF02
	KeyF03
	KeyF04
	KeyF05
	KeyF06
	KeyF07
	KeyF08
	KeyF09
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24
	KeyF25
	KeyF26
	KeyF27
	KeyF28
	KeyF29
	KeyF30
	KeyF31
	KeyF32
	KeyF33
	KeyF34
	KeyF35
	KeyF36
	KeyF37
	KeyF38
	KeyF39
	KeyF40
	KeyF41
	KeyF42
	KeyF43
	KeyF44
	KeyF45
	KeyF46
	KeyF47
	KeyF48
	KeyF49
	KeyF50
	KeyF51
	KeyF52
	KeyF53
	KeyF54
	KeyF55
	KeyF56
	KeyF57
	KeyF58
	KeyF59
	KeyF60
	KeyF61
	KeyF62
	KeyF63
	KeyF64
	KeyF65
	KeyF66
	KeyF67
	KeyF68
	KeyF69
	KeyF70
	KeyF71
	KeyF72
	KeyF73
	KeyF74
	KeyF75
	KeyF76
	KeyF77
	KeyF78
	KeyF79
	KeyF80
	KeyF81
	KeyF82
	KeyF83
	KeyF84
	KeyF85
	KeyF86
	KeyF87
	KeyF88
	KeyF89
	KeyF90
	KeyF91
	KeyF92
	KeyF93
	KeyF94
	KeyF95
	KeyF96
	KeyF97
	KeyF98
	KeyF99   // Ok that's enough
	KeyEnter // kent
	KeyClear // kclr
	KeyDownLeft
	KeyDownRight
	KeyUpLeft
	KeyUpRight
	KeyCenter
	KeyBegin
	KeyCancel
	KeyClose
	KeyCommand
	KeyCopy
	KeyExit
	KeyPrint
	KeyRefresh
	KeySeparator
	// notcurses says these are only avaialbe in kitty kbp
	KeyCapsLock
	KeyScrollLock
	KeyNumlock
	KeyPrintScreen
	KeyPause
	KeyMenu
	// Media keys, also generally only kitty kbp
	KeyMediaPlay
	KeyMediaPause
	KeyMediaPPause // wtf is this?
	KeyMediaRev
	KeyMediaStop
	KeyMediaFF
	KeyMediaRewind
	KeyMediaNext
	KeyMediaPrev
	KeyMediaRecord
	KeyMediaVolDown
	KeyMediaVolUp
	KeyMediaMute
	// Modifiers, when pressed by themselves
	KeyLeftShift
	KeyLeftControl
	KeyLeftAlt
	KeyLeftSuper
	KeyLeftHyper
	KeyLeftMeta
	KeyRightShift
	KeyRightControl
	KeyRightAlt
	KeyRightSuper
	KeyRightHyper
	KeyRightMeta
	KeyL3Shift
	KeyL5Shift

	// Aliases
	KeyReturn = KeyEnter
	KeyTab    = 0x09
	KeyEsc    = 0x1B
	KeySpace  = 0x20
)
