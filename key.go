package rtk

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
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
	buf := &bytes.Buffer{}
	if k.Modifiers != 0 {
		buf.WriteRune('<')
		if k.Modifiers&ModNumLock != 0 {
			buf.WriteString("num-")
		}
		if k.Modifiers&ModCapsLock != 0 {
			buf.WriteString("caps-")
		}
		if k.Modifiers&ModMeta != 0 {
			buf.WriteString("meta-")
		}
		if k.Modifiers&ModHyper != 0 {
			buf.WriteString("hyper-")
		}
		if k.Modifiers&ModSuper != 0 {
			buf.WriteString("super-")
		}
		if k.Modifiers&ModCtrl != 0 {
			buf.WriteString("c-")
		}
		if k.Modifiers&ModAlt != 0 {
			buf.WriteString("a-")
		}
		if k.Modifiers&ModShift != 0 {
			buf.WriteString("s-")
		}
	}
	switch {
	case k.Codepoint == KeyTab:
		// Handle further down
	case k.Codepoint == KeySpace:
		// Handle further down
	case k.Codepoint == KeyEsc:
		// Handle further down
	case k.Codepoint == KeyBackspace:
		// handle further down
	case k.Codepoint < 0x00:
		return "<invalid>"
	case k.Codepoint < 0x20:
		ch := fmt.Sprintf("%c", k.Codepoint+0x40)
		return fmt.Sprintf("<c-%s>", strings.ToLower(ch))
	case k.Codepoint <= unicode.MaxRune:
		buf.WriteRune(k.Codepoint)
		if k.Modifiers != 0 {
			buf.WriteRune('>')
		}
		return buf.String()
	}

	if k.Modifiers == 0 {
		buf.WriteRune('<')
	}
	// By this point we only have extended keys
	switch k.Codepoint {
	case KeyUp:
		buf.WriteString("up")
	case KeyRight:
		buf.WriteString("right")
	case KeyDown:
		buf.WriteString("down")
	case KeyLeft:
		buf.WriteString("left")
	case KeyInsert:
		buf.WriteString("insert")
	case KeyDelete:
		buf.WriteString("delete")
	case KeyBackspace:
		buf.WriteString("bs")
	case KeyPgDown:
		buf.WriteString("pgdown")
	case KeyPgUp:
		buf.WriteString("pgup")
	case KeyHome:
		buf.WriteString("home")
	case KeyEnd:
		buf.WriteString("end")
	case KeyF00:
		buf.WriteString("f0")
	case KeyF01:
		buf.WriteString("f1")
	case KeyF02:
		buf.WriteString("f2")
	case KeyF03:
		buf.WriteString("f3")
	case KeyF04:
		buf.WriteString("f4")
	case KeyF05:
		buf.WriteString("f5")
	case KeyF06:
		buf.WriteString("f6")
	case KeyF07:
		buf.WriteString("f7")
	case KeyF08:
		buf.WriteString("f8")
	case KeyF09:
		buf.WriteString("f9")
	case KeyF10:
		buf.WriteString("f10")
	case KeyF11:
		buf.WriteString("f11")
	case KeyF12:
		buf.WriteString("f12")
	case KeyF13:
		buf.WriteString("f13")
	case KeyF14:
		buf.WriteString("f14")
	case KeyF15:
		buf.WriteString("f15")
	case KeyF16:
		buf.WriteString("f16")
	case KeyF17:
		buf.WriteString("f17")
	case KeyF18:
		buf.WriteString("f18")
	case KeyF19:
		buf.WriteString("f19")
	case KeyF20:
		buf.WriteString("f20")
	case KeyF21:
		buf.WriteString("f21")
	case KeyF22:
		buf.WriteString("f22")
	case KeyF23:
		buf.WriteString("f23")
	case KeyF24:
		buf.WriteString("f24")
	case KeyF25:
		buf.WriteString("f25")
	case KeyF26:
		buf.WriteString("f26")
	case KeyF27:
		buf.WriteString("f27")
	case KeyF28:
		buf.WriteString("f28")
	case KeyF29:
		buf.WriteString("f29")
	case KeyF30:
		buf.WriteString("f30")
	case KeyF31:
		buf.WriteString("f31")
	case KeyF32:
		buf.WriteString("f32")
	case KeyF33:
		buf.WriteString("f33")
	case KeyF34:
		buf.WriteString("f34")
	case KeyF35:
		buf.WriteString("f35")
	case KeyF36:
		buf.WriteString("f36")
	case KeyF37:
		buf.WriteString("f37")
	case KeyF38:
		buf.WriteString("f38")
	case KeyF39:
		buf.WriteString("f39")
	case KeyF40:
		buf.WriteString("f40")
	case KeyF41:
		buf.WriteString("f41")
	case KeyF42:
		buf.WriteString("f42")
	case KeyF43:
		buf.WriteString("f43")
	case KeyF44:
		buf.WriteString("f44")
	case KeyF45:
		buf.WriteString("f45")
	case KeyF46:
		buf.WriteString("f46")
	case KeyF47:
		buf.WriteString("f47")
	case KeyF48:
		buf.WriteString("f48")
	case KeyF49:
		buf.WriteString("f49")
	case KeyF50:
		buf.WriteString("f50")
	case KeyF51:
		buf.WriteString("f51")
	case KeyF52:
		buf.WriteString("f52")
	case KeyF53:
		buf.WriteString("f53")
	case KeyF54:
		buf.WriteString("f54")
	case KeyF55:
		buf.WriteString("f55")
	case KeyF56:
		buf.WriteString("f56")
	case KeyF57:
		buf.WriteString("f57")
	case KeyF58:
		buf.WriteString("f58")
	case KeyF59:
		buf.WriteString("f59")
	case KeyF60:
		buf.WriteString("f60")
	case KeyF61:
		buf.WriteString("f61")
	case KeyF62:
		buf.WriteString("f62")
	case KeyF63:
		buf.WriteString("f63")
	case KeyEnter:
		buf.WriteString("enter")
	case KeyClear:
		buf.WriteString("clear")
	case KeyDownLeft:
		buf.WriteString("down-left")
	case KeyDownRight:
		buf.WriteString("down-right")
	case KeyUpLeft:
		buf.WriteString("up-left")
	case KeyUpRight:
		buf.WriteString("up-right")
	case KeyCenter:
		buf.WriteString("center")
	case KeyBegin:
		buf.WriteString("begin")
	case KeyCancel:
		buf.WriteString("cancel")
	case KeyClose:
		buf.WriteString("close")
	case KeyCommand:
		buf.WriteString("cmd")
	case KeyCopy:
		buf.WriteString("copy")
	case KeyExit:
		buf.WriteString("exit")
	case KeyPrint:
		buf.WriteString("print")
	case KeyRefresh:
		buf.WriteString("refresh")
		// notcurses says these are only avaialbe in kitty kbp:
	case KeyCapsLock:
		buf.WriteString("caps-lock")
	case KeyScrollLock:
		buf.WriteString("scroll-lock")
	case KeyNumlock:
		buf.WriteString("num-lock")
	case KeyPrintScreen:
		buf.WriteString("prtscr")
	case KeyPause:
		buf.WriteString("pause")
	case KeyMenu:
		buf.WriteString("menu")
		// Media keys, also generally only kitty kbp:
	case KeyMediaPlay:
		buf.WriteString("media-play")
	case KeyMediaPause:
		buf.WriteString("media-pause")
	case KeyMediaPPause:
		buf.WriteString("mediea-ppause")
	case KeyMediaRev:
		buf.WriteString("media-rev")
	case KeyMediaStop:
		buf.WriteString("media-stop")
	case KeyMediaFF:
		buf.WriteString("media-ff")
	case KeyMediaRewind:
		buf.WriteString("media-rw")
	case KeyMediaNext:
		buf.WriteString("media-next")
	case KeyMediaPrev:
		buf.WriteString("media-prev")
	case KeyMediaRecord:
		buf.WriteString("media-rec")
	case KeyMediaVolDown:
		buf.WriteString("vol-down")
	case KeyMediaVolUp:
		buf.WriteString("vol-up")
	case KeyMediaMute:
		buf.WriteString("mute")
	// Modifiers, when pressed by themselves
	case KeyLeftShift:
		buf.WriteString("left-shift")
	case KeyLeftControl:
		buf.WriteString("left-ctrl")
	case KeyLeftAlt:
		buf.WriteString("left-alt")
	case KeyLeftSuper:
		buf.WriteString("left-super")
	case KeyLeftHyper:
		buf.WriteString("left-hyper")
	case KeyLeftMeta:
		buf.WriteString("left-meta")
	case KeyRightShift:
		buf.WriteString("right-shift")
	case KeyRightControl:
		buf.WriteString("right-ctrl")
	case KeyRightAlt:
		buf.WriteString("right-alt")
	case KeyRightSuper:
		buf.WriteString("right-super")
	case KeyRightHyper:
		buf.WriteString("right-hyper")
	case KeyRightMeta:
		buf.WriteString("right-meta")
	case KeyL3Shift:
		buf.WriteString("l3-shift")
	case KeyL5Shift:
		buf.WriteString("l5-shift")
	// Aliases
	case KeyTab:
		buf.WriteString("tab")
	case KeyEsc:
		buf.WriteString("esc")
	case KeySpace:
		buf.WriteString("space")
	}
	buf.WriteRune('>')
	return buf.String()
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
	KeyF63 // F63 is max defined in terminfo
	KeyEnter
	KeyClear
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

var keyMap = map[string]Key{}
