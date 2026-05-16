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
			vt.ich(ps(seq))
		case 'A':
			vt.cuu(ps(seq))
		case 'B':
			vt.cud(ps(seq))
		case 'C':
			vt.cuf(ps(seq))
		case 'D':
			vt.cub(ps(seq))
		case 'E':
			vt.cnl(ps(seq))
		case 'F':
			vt.cpl(ps(seq))
		case 'G':
			vt.cha(ps(seq))
		case 'H', 'f':
			vt.cup(seq)
		case 'I':
			vt.cht(ps(seq))
		case 'J':
			vt.ed(ps(seq), false)
		case 'K':
			vt.el(ps(seq), false)
		case 'L':
			vt.il(ps(seq))
		case 'M':
			vt.dl(ps(seq))
		case 'P':
			vt.dch(ps(seq))
		case 'S':
			vt.scrollUp(defaultOne(ps(seq)))
		case 'T':
			if seq.NumParameters != 5 {
				vt.scrollDown(defaultOne(ps(seq)))
			}
		case 'W':
			vt.ctc(seq, false)
		case 'X':
			vt.ech(ps(seq))
		case 'Z':
			vt.cbt(ps(seq))
		case '`':
			vt.hpa(ps(seq))
		case 'a':
			vt.hpr(ps(seq))
		case 'b':
			vt.rep(ps(seq))
		case 'c':
			vt.primaryDeviceAttributes()
		case 'd':
			vt.vpa(ps(seq))
		case 'e':
			vt.vpr(ps(seq))
		case 'g':
			vt.tbc(ps(seq))
		case 'h':
			vt.sm(seq)
		case 'l':
			vt.rm(seq)
		case 'm':
			vt.sgr(seq)
		case 'n':
			vt.deviceStatusReport(ps(seq), false)
		case 'r':
			vt.decstbm(seq)
		case 's':
			if vt.mode.declrmm {
				vt.decslrm(seq)
			} else {
				vt.decsc()
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
			case 'q':
				vt.xtversion()
			}
		case '=':
			if seq.Final == 'c' {
				vt.tertiaryDeviceAttributes()
			}
		case '?':
			switch seq.Final {
			case 'h':
				vt.decset(seq)
			case 'J':
				vt.ed(ps(seq), true)
			case 'K':
				vt.el(ps(seq), true)
			case 'n':
				vt.deviceStatusReport(ps(seq), true)
			case 'r':
				vt.restoreMode(seq)
			case 's':
				vt.saveMode(seq)
			case 'W':
				vt.ctc(seq, true)
			case 'l':
				vt.decrst(seq)
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
