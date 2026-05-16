package term

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
	"github.com/mattn/go-sixel"
)

func applySequence(vt *Model, seq ansi.Sequence) {
	switch seq := seq.(type) {
	case ansi.Print:
		vt.print(seq)
	case ansi.C0:
		applyC0(vt, rune(seq))
	case ansi.ESC:
		applyESC(vt, seq)
	case ansi.CSI:
		applyCSI(vt, seq)
	case ansi.OSC:
		vt.osc(string(seq.Payload))
	case ansi.DCS:
		if seq.Final == 'q' && seq.NumIntermediate == 0 && seq.NumParameters == 0 {
			sixelAction{seq: seq}.apply(vt)
		}
	case ansi.APC:
		vt.postEvent(EventAPC{Payload: seq.Data})
	}
}

func applyC0(vt *Model, r rune) {
	switch r {
	case 0x07:
		vt.postEvent(EventBell{})
	case 0x08:
		vt.bs()
	case 0x09:
		vt.ht()
	case 0x0A, 0x0B, 0x0C:
		vt.lf()
	case 0x0D:
		vt.cr()
	case 0x0E:
		vt.charsets.selected = g1
	case 0x0F:
		vt.charsets.selected = g0
	}
}

func applyESC(vt *Model, seq ansi.ESC) {
	switch seq.NumIntermediate {
	case 0:
		switch seq.Final {
		case '7':
			vt.decsc()
		case '8':
			vt.decrc()
		case 'D':
			vt.ind()
		case 'E':
			vt.nel()
		case 'H':
			vt.hts()
		case 'M':
			vt.ri()
		case 'N':
			vt.charsets.saved = vt.charsets.selected
			vt.charsets.selected = g2
			vt.charsets.singleShift = true
		case 'O':
			vt.charsets.saved = vt.charsets.selected
			vt.charsets.selected = g3
			vt.charsets.singleShift = true
		case 'V':
			vt.setProtectedMode(protectedModeISO)
		case 'W':
			vt.setProtectedMode(protectedModeOff)
		case 'Z':
			vt.primaryDeviceAttributes()
		case '=':
			vt.mode.deckpam = true
			vt.mode.deckpnm = false
		case '>':
			vt.mode.deckpam = false
			vt.mode.deckpnm = true
		case 'c':
			vt.ris()
		case 'n':
			vt.charsets.selected = g2
		case 'o':
			vt.charsets.selected = g3
		}
	case 1:
		switch seq.Intermediate[0] {
		case '#':
			if seq.Final == '8' {
				vt.decaln()
			}
		case '(':
			vt.designateGCharset(g0, seq.Final)
		case ')':
			vt.designateGCharset(g1, seq.Final)
		case '*':
			vt.designateGCharset(g2, seq.Final)
		case '+':
			vt.designateGCharset(g3, seq.Final)
		}
	}
}

func (vt *Model) designateGCharset(designator charsetDesignator, final rune) {
	switch final {
	case 'A':
		vt.charsets.designations[designator] = british
	case '0':
		vt.charsets.designations[designator] = decSpecialAndLineDrawing
	case 'B':
		vt.charsets.designations[designator] = ascii
	}
}

func applyCSI(vt *Model, seq ansi.CSI) {
	switch seq.NumIntermediate {
	case 0:
		switch seq.Final {
		case '@':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.ich(defaultOne(ps(seq)))
		case 'A', 'k':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cuu(ps(seq))
		case 'B':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cud(ps(seq))
		case 'C':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cuf(ps(seq))
		case 'D', 'j':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cub(ps(seq))
		case 'E':
			if validCSIParamCount(seq, 1) {
				vt.cnl(ps(seq))
			}
		case 'F':
			if validCSIParamCount(seq, 1) {
				vt.cpl(ps(seq))
			}
		case 'G':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cha(ps(seq))
		case 'H', 'f':
			if validCSIParamCount(seq, 2) {
				vt.cup(seq)
			}
		case 'I':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cht(tabCount(seq))
		case 'J':
			if ps, ok := eraseDisplayParam(seq); ok {
				vt.ed(ps, false)
			}
		case 'K':
			if ps, ok := eraseLineParam(seq); ok {
				vt.el(ps, false)
			}
		case 'L':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.il(defaultOneIfMissing(seq))
		case 'M':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.dl(defaultOneIfMissing(seq))
		case 'P':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.dch(defaultOneIfMissing(seq))
		case 'S':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.scrollUp(defaultOneIfMissing(seq))
		case 'T':
			if validCSIParamCount(seq, 1) {
				vt.scrollDown(defaultOneIfMissing(seq))
			}
		case 'W':
			vt.ctc(seq, false)
		case 'X':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.ech(ps(seq))
		case 'Z':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.cbt(tabCount(seq))
		case '`':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.hpa(ps(seq))
		case 'a':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.hpr(ps(seq))
		case 'b':
			if seq.NumParameters <= 1 {
				vt.rep(ps(seq))
			}
		case 'c':
			vt.primaryDeviceAttributes()
		case 'd':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.vpa(ps(seq))
		case 'e':
			if !validCSIParamCount(seq, 1) {
				return
			}
			vt.vpr(ps(seq))
		case 'g':
			if seq.NumParameters == 1 {
				vt.tbc(ps(seq))
			}
		case 'h':
			vt.sm(seq)
		case 'l':
			vt.rm(seq)
		case 'm':
			vt.sgr(seq)
		case 'n':
			if ps, ok := deviceStatusParam(seq, false); ok {
				vt.deviceStatusReport(ps, false)
			}
		case 'r':
			vt.decstbm(seq)
		case 's':
			if seq.NumParameters == 0 && !vt.mode.declrmm {
				vt.decsc()
			} else {
				vt.decslrm(seq)
			}
		case 't':
			vt.xtwinops(seq)
		case 'u':
			vt.decrc()
		}
	case 1:
		switch seq.Intermediate[0] {
		case '>':
			switch seq.Final {
			case 'c':
				vt.enqueueReplyString("\x1b[>1;0;0c")
			case 'm':
				vt.modifyKeyFormat(seq)
			case 'n':
				vt.mode.modifyOtherKeys2 = false
			case 'q':
				vt.xtversion()
			case 's':
				vt.xtshiftescape(seq)
			case 'u':
				vt.kittyKeyboardPush(seq)
			}
		case '<':
			if seq.Final == 'u' {
				vt.kittyKeyboardPop(seq)
			}
		case '=':
			switch seq.Final {
			case 'c':
				vt.tertiaryDeviceAttributes()
			case 'u':
				vt.kittyKeyboardSet(seq)
			}
		case '?':
			switch seq.Final {
			case 'h':
				vt.decset(seq)
			case 'J':
				if ps, ok := eraseDisplayParam(seq); ok {
					vt.ed(ps, true)
				}
			case 'K':
				if ps, ok := eraseLineParam(seq); ok {
					vt.el(ps, true)
				}
			case 'n':
				if ps, ok := deviceStatusParam(seq, true); ok {
					vt.deviceStatusReport(ps, true)
				}
			case 'r':
				vt.restoreMode(seq)
			case 's':
				vt.saveMode(seq)
			case 'W':
				vt.ctc(seq, true)
			case 'l':
				vt.decrst(seq)
			case 'u':
				vt.kittyKeyboardQuery()
			}
		case ' ':
			if seq.Final == 'q' {
				vt.decscusr(seq)
			}
		case '"':
			if seq.Final == 'q' {
				vt.decsca(seq)
			}
		case '$':
			switch seq.Final {
			case 'p':
				vt.decrqm(ps(seq), true)
			case '}':
				vt.decsasd(seq)
			}
		}
	case 2:
		if seq.Intermediate[0] == '?' && seq.Intermediate[1] == '$' && seq.Final == 'p' {
			vt.decrqm(ps(seq), false)
		}
	}
}

func validCSIParamCount(seq ansi.CSI, max int) bool {
	return seq.NumParameters <= max
}

func tabCount(seq ansi.CSI) int {
	if seq.NumParameters == 0 {
		return 1
	}
	return seq.Param(0)
}

func defaultOneIfMissing(seq ansi.CSI) int {
	if seq.NumParameters == 0 {
		return 1
	}
	return seq.Param(0)
}

func eraseDisplayParam(seq ansi.CSI) (int, bool) {
	if seq.NumParameters == 0 {
		return 0, true
	}
	if seq.NumParameters != 1 {
		return 0, false
	}
	ps := seq.Param(0)
	switch ps {
	case 0, 1, 2, 3, 22:
		return ps, true
	default:
		return 0, false
	}
}

func eraseLineParam(seq ansi.CSI) (int, bool) {
	if seq.NumParameters == 0 {
		return 0, true
	}
	if seq.NumParameters != 1 {
		return 0, false
	}
	ps := seq.Param(0)
	switch ps {
	case 0, 1, 2:
		return ps, true
	default:
		return 0, false
	}
}

func deviceStatusParam(seq ansi.CSI, private bool) (int, bool) {
	if seq.NumParameters != 1 {
		return 0, false
	}
	ps := seq.Param(0)
	if private {
		return ps, ps == 996
	}
	switch ps {
	case 5, 6:
		return ps, true
	default:
		return 0, false
	}
}

func defaultOne(n int) int {
	if n == 0 {
		return 1
	}
	return n
}

func (vt *Model) primaryDeviceAttributes() {
	resp := strings.Builder{}
	resp.WriteString("\x1B[?")
	resp.WriteString("62;")
	resp.WriteString("4;")
	resp.WriteString("22")
	resp.WriteString("c")
	vt.enqueueReplyString(resp.String())
}

func (vt *Model) tertiaryDeviceAttributes() {
	vt.enqueueReplyString("\x1BP!|00000000\x1B\\")
}

func (vt *Model) deviceStatusReport(n int, private bool) {
	if private {
		switch n {
		case 996:
			if vt.theme != 0 {
				vt.enqueueReplyString(fmt.Sprintf("\x1B[?997;%dn", vt.theme))
			}
		}
		return
	}

	switch n {
	case 5:
		vt.enqueueReplyString("\x1B[0n")
	case 6:
		reportRow := vt.cursor.row
		reportCol := vt.cursor.col
		if vt.mode.decom {
			reportRow = max(0, reportRow-vt.margin.top)
			reportCol = max(0, reportCol-vt.margin.left)
		}
		resp := fmt.Sprintf("\x1B[%d;%dR", reportRow+1, reportCol+1)
		vt.enqueueReplyString(resp)
	}
}

func (vt *Model) xtwinops(seq ansi.CSI) {
	if seq.NumParameters != 1 {
		return
	}
	switch ps(seq) {
	case 14:
		if vt.size.XPixel > 0 && vt.size.YPixel > 0 {
			vt.enqueueReplyString(fmt.Sprintf("\x1B[4;%d;%dt", vt.size.YPixel, vt.size.XPixel))
		}
	case 16:
		if vt.size.Cols > 0 && vt.size.Rows > 0 && vt.size.XPixel > 0 && vt.size.YPixel > 0 {
			vt.enqueueReplyString(fmt.Sprintf("\x1B[6;%d;%dt", vt.size.YPixel/vt.size.Rows, vt.size.XPixel/vt.size.Cols))
		}
	case 18:
		vt.enqueueReplyString(fmt.Sprintf("\x1B[8;%d;%dt", vt.height(), vt.width()))
	case 21:
		vt.enqueueReplyString("\x1B]l" + vt.title + "\x1B\\")
	}
}

func (vt *Model) xtversion() {
	vt.enqueueReplyString("\x1BP>|vaxis\x1B\\")
}

func (vt *Model) decscusr(seq ansi.CSI) {
	if seq.NumParameters > 1 {
		return
	}
	style := ps(seq)
	if style < int(vaxis.CursorDefault) || style > int(vaxis.CursorBeam) {
		return
	}
	vt.cursor.style = vaxis.CursorStyle(style)
}

type sixelAction struct{ seq ansi.DCS }

func (a sixelAction) apply(vt *Model) {
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte{'\x1B', 'P'})
	for i, p := range a.seq.Params() {
		buf.WriteString(strconv.Itoa(int(p)))
		if i <= a.seq.NumParameters-1 {
			buf.WriteByte(';')
		}
	}
	buf.WriteByte('q')
	buf.WriteString(string(a.seq.Data))
	buf.Write([]byte{0x1B, '\\'})

	log.Info("SIXEL %d", buf.Len())
	dec := sixel.NewDecoder(buf)
	img := &Image{}
	img.origin.row = int(vt.cursor.row)
	img.origin.col = int(vt.cursor.col)
	err := dec.Decode(&img.img)
	if err != nil {
		log.Error("couldn't decode sixel: %v", err)
		return
	}
	vt.graphics = append(vt.graphics, img)
}
