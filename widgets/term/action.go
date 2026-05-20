package term

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
	"git.sr.ht/~rockorager/vaxis/sixel"
)

func applySequence(vt *Model, seq ansi.Sequence) {
	switch seq := seq.(type) {
	case ansi.Print:
		vt.print(seq)
	case ansi.C0:
		applyC0(vt, rune(seq))
	case ansi.ESC:
		applyESC(vt, seq)
	case ansi.SS3:
		vt.charsets.saved = vt.charsets.selected
		vt.charsets.selected = g3
		vt.charsets.singleShift = true
		vt.print(ansi.Print{Grapheme: string(rune(seq)), Width: 1})
	case ansi.CSI:
		applyCSI(vt, seq)
	case ansi.OSC:
		if seq.InvalidUTF8 {
			return
		}
		vt.osc(string(seq.Payload))
	case ansi.DCS:
		if seq.NumIntermediate == 1 && seq.Intermediate[0] == '$' && seq.Final == 'q' {
			vt.decrqss(seq)
		} else if seq.NumIntermediate == 1 && seq.Intermediate[0] == '+' && seq.Final == 'q' {
			vt.xtgettcap(seq)
		} else if seq.Final == 'q' && seq.NumIntermediate == 0 {
			sixelAction{seq: seq}.apply(vt)
		}
	case ansi.APC:
		vt.postEvent(EventAPC{Payload: seq.Data})
	}
}

func applyC0(vt *Model, r rune) {
	switch r {
	case 0x05:
		vt.enquiry()
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
		case '~':
			vt.charsets.gr = g1
		case '}':
			vt.charsets.gr = g2
		case '|':
			vt.charsets.gr = g3
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
			vt.hpr(defaultOneIfMissing(seq))
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
			vt.vpr(defaultOneIfMissing(seq))
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
				vt.enqueueReplyString(secondaryDeviceAttributesReply)
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
				if seq.NumParameters == 1 {
					vt.decrqm(seq.Param(0), true)
				}
			case '}':
				vt.decsasd(seq)
			}
		}
	case 2:
		if seq.Intermediate[0] == '?' && seq.Intermediate[1] == '$' && seq.Final == 'p' && seq.NumParameters == 1 {
			vt.decrqm(seq.Param(0), false)
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

const (
	primaryDeviceAttributesReply   = "\x1B[?62;4;22c"
	secondaryDeviceAttributesReply = "\x1B[>1;10;0c"
	tertiaryDeviceAttributesReply  = "\x1BP!|00000000\x1B\\"
)

func (vt *Model) primaryDeviceAttributes() {
	vt.enqueueReplyString(primaryDeviceAttributesReply)
}

func (vt *Model) tertiaryDeviceAttributes() {
	vt.enqueueReplyString(tertiaryDeviceAttributesReply)
}

func (vt *Model) deviceStatusReport(n int, private bool) {
	if private {
		switch n {
		case 996:
			if reply := colorSchemeReport(vt.theme); reply != "" {
				vt.enqueueReplyString(reply)
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

func colorSchemeReport(theme vaxis.ColorThemeMode) string {
	switch theme {
	case vaxis.DarkMode, vaxis.LightMode:
		return fmt.Sprintf("\x1B[?997;%dn", theme)
	default:
		return ""
	}
}

func (vt *Model) xtwinops(seq ansi.CSI) {
	if xtwinopsTitleStack(seq) {
		return
	}
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

func xtwinopsTitleStack(seq ansi.CSI) bool {
	if seq.NumParameters != 2 && seq.NumParameters != 3 {
		return false
	}
	op := seq.Param(0)
	target := seq.Param(1)
	return (op == 22 || op == 23) && (target == 0 || target == 2)
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
	if style == int(vaxis.CursorDefault) {
		style = int(vaxis.CursorBlock)
	}
	vt.cursor.style = vaxis.CursorStyle(style)
	switch vt.cursor.style {
	case vaxis.CursorBlockBlinking, vaxis.CursorUnderlineBlinking, vaxis.CursorBeamBlinking:
		vt.mode.cursorBlinking = true
	default:
		vt.mode.cursorBlinking = false
	}
}

func (vt *Model) effectiveCursorStyle() vaxis.CursorStyle {
	return cursorStyleWithBlink(vt.cursor.style, vt.mode.cursorBlinking)
}

func (vt *Model) decrqssCursorStyle() vaxis.CursorStyle {
	style := vt.effectiveCursorStyle()
	if style == vaxis.CursorDefault {
		if vt.mode.cursorBlinking {
			return vaxis.CursorBlockBlinking
		}
		return vaxis.CursorBlock
	}
	return style
}

func cursorStyleWithBlink(style vaxis.CursorStyle, blinking bool) vaxis.CursorStyle {
	if blinking {
		switch style {
		case vaxis.CursorUnderline, vaxis.CursorUnderlineBlinking:
			return vaxis.CursorUnderlineBlinking
		case vaxis.CursorBeam, vaxis.CursorBeamBlinking:
			return vaxis.CursorBeamBlinking
		default:
			return vaxis.CursorBlockBlinking
		}
	}

	switch style {
	case vaxis.CursorBlockBlinking:
		return vaxis.CursorBlock
	case vaxis.CursorUnderlineBlinking:
		return vaxis.CursorUnderline
	case vaxis.CursorBeamBlinking:
		return vaxis.CursorBeam
	default:
		return style
	}
}

func (vt *Model) decrqss(seq ansi.DCS) {
	if len(seq.Data) > 2 {
		return
	}

	switch string(seq.Data) {
	case "m":
		vt.enqueueReplyString("\x1BP1$r" + vt.sgrStatusString() + "m\x1B\\")
	case "r":
		vt.enqueueReplyString(fmt.Sprintf("\x1BP1$r%d;%dr\x1B\\", vt.margin.top+1, vt.margin.bottom+1))
	case "s":
		if !vt.mode.declrmm {
			vt.enqueueReplyString("\x1BP0$r\x1B\\")
			return
		}
		vt.enqueueReplyString(fmt.Sprintf("\x1BP1$r%d;%ds\x1B\\", vt.margin.left+1, vt.margin.right+1))
	case " q":
		vt.enqueueReplyString(fmt.Sprintf("\x1BP1$r%d q\x1B\\", vt.decrqssCursorStyle()))
	default:
		vt.enqueueReplyString("\x1BP0$r\x1B\\")
	}
}

func (vt *Model) xtgettcap(seq ansi.DCS) {
	start := 0
	for i, r := range seq.Data {
		if r != ';' {
			continue
		}
		vt.xtgettcapReply(seq.Data[start:i])
		start = i + 1
	}
	vt.xtgettcapReply(seq.Data[start:])
}

func (vt *Model) xtgettcapReply(keyRunes []rune) {
	if len(keyRunes) == 0 {
		return
	}
	key := strings.ToUpper(string(keyRunes))
	value, ok := xtgettcapValues[key]
	if !ok {
		return
	}
	vt.enqueueReplyString("\x1BP1+r" + key + value + "\x1B\\")
}

var xtgettcapValues = map[string]string{
	"4158":       "",
	"544E":       "=" + hexUpperString("xterm-256color"),
	"436F":       "=" + hexUpperString("256"),
	"524742":     "=" + hexUpperString("8"),
	"5463":       "",
	"5375":       "",
	"5854":       "",
	"536D756C78": "=" + hexUpperString(`\E[4:%p1%dm`),
	"536574756C63": "=" + hexUpperString(
		`\E[58:2::%p1%{65536}%/%d:%p1%{256}%/%{255}%&%d:%p1%{255}%&%d%;m`,
	),
	"5373": "=" + hexUpperString(`\E[%p1%d q`),
	"5365": "=" + hexUpperString(`\E[0 q`),
	"4D73": "=" + hexUpperString(`\E]52;%p1%s;%p2%s\007`),
}

func hexUpperString(s string) string {
	const digits = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		c := s[i]
		b.WriteByte(digits[c>>4])
		b.WriteByte(digits[c&0x0f])
	}
	return b.String()
}

func (vt *Model) sgrStatusString() string {
	var b strings.Builder
	b.WriteByte('0')
	if vt.cursor.Attribute&vaxis.AttrBold != 0 {
		b.WriteString(";1")
	}
	if vt.cursor.Attribute&vaxis.AttrDim != 0 {
		b.WriteString(";2")
	}
	if vt.cursor.Attribute&vaxis.AttrItalic != 0 {
		b.WriteString(";3")
	}
	if vt.cursor.UnderlineStyle != vaxis.UnderlineOff {
		b.WriteString(";4")
	}
	if vt.cursor.Attribute&vaxis.AttrBlink != 0 {
		b.WriteString(";5")
	}
	if vt.cursor.Attribute&vaxis.AttrReverse != 0 {
		b.WriteString(";7")
	}
	if vt.cursor.Attribute&vaxis.AttrInvisible != 0 {
		b.WriteString(";8")
	}
	if vt.cursor.Attribute&vaxis.AttrStrikethrough != 0 {
		b.WriteString(";9")
	}
	writeSGRStatusColor(&b, vt.cursor.Foreground, false)
	writeSGRStatusColor(&b, vt.cursor.Background, true)
	return b.String()
}

func writeSGRStatusColor(b *strings.Builder, color vaxis.Color, background bool) {
	params := color.Params()
	switch len(params) {
	case 1:
		idx := params[0]
		if idx < 8 {
			if background {
				b.WriteString(";4")
			} else {
				b.WriteString(";3")
			}
			b.WriteString(strconv.Itoa(int(idx)))
		} else if idx < 16 {
			if background {
				b.WriteString(";10")
			} else {
				b.WriteString(";9")
			}
			b.WriteString(strconv.Itoa(int(idx - 8)))
		} else {
			if background {
				b.WriteString(";48:5:")
			} else {
				b.WriteString(";38:5:")
			}
			b.WriteString(strconv.Itoa(int(idx)))
		}
	case 3:
		if background {
			b.WriteString(";48:2::")
		} else {
			b.WriteString(";38:2::")
		}
		b.WriteString(strconv.Itoa(int(params[0])))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(int(params[1])))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(int(params[2])))
	}
}

type sixelAction struct{ seq ansi.DCS }

func (a sixelAction) apply(vt *Model) {
	buf := bytes.NewBuffer(nil)
	buf.Write([]byte{'\x1B', 'P'})
	for i, p := range a.seq.Params() {
		buf.WriteString(strconv.Itoa(int(p)))
		if i < a.seq.NumParameters-1 {
			buf.WriteByte(';')
		}
	}
	buf.WriteByte('q')
	buf.WriteString(string(a.seq.Data))
	buf.Write([]byte{0x1B, '\\'})

	log.Info("SIXEL %d", buf.Len())
	dec := sixel.NewDecoder(buf)
	img := &Image{}
	err := dec.Decode(&img.img)
	if err != nil {
		log.Error("couldn't decode sixel: %v", err)
		return
	}
	vt.positionSixel(img)
	vt.graphics = append(vt.graphics, img)
}

func (vt *Model) positionSixel(img *Image) {
	bounds := img.img.Bounds()
	img.cols = vt.sixelCellCols(bounds.Dx())
	img.rows = vt.sixelCellRows(bounds.Dy())

	if !vt.mode.sixelScrolling {
		img.origin.row = 0
		img.origin.col = 0
		img.sourceRow = vt.activeScreen.scrollbackLen()
		return
	}

	startRow := vt.cursor.row
	startCol := vt.cursor.col
	scrolls := 0
	if vt.cursorInHorizontalMargins() && startRow <= vt.margin.bottom {
		scrolls = max(0, int(startRow)+img.rows-1-int(vt.margin.bottom))
	}
	img.origin.row = max(0, int(startRow)-scrolls)
	img.origin.col = int(startCol)
	img.sourceRow = vt.activeScreen.scrollbackLen() + int(startRow)

	for i := 0; i < img.rows-1; i += 1 {
		vt.ind()
	}
	if vt.mode.sixelCursorRight {
		vt.cursor.col = min(startCol+column(img.cols), column(vt.width())-1)
	} else {
		vt.cursor.col = startCol
	}
	vt.lastCol = false
}

func (vt *Model) sixelCellCols(pixelWidth int) int {
	cellWidth, _ := vt.sixelCellPixels()
	return max(1, (pixelWidth+cellWidth-1)/cellWidth)
}

func (vt *Model) sixelCellRows(pixelHeight int) int {
	_, cellHeight := vt.sixelCellPixels()
	return max(1, (pixelHeight+cellHeight-1)/cellHeight)
}

func (vt *Model) sixelCellPixels() (int, int) {
	cellWidth := 1
	cellHeight := 1
	if vt.size.Cols > 0 && vt.size.XPixel > 0 {
		cellWidth = max(1, vt.size.XPixel/vt.size.Cols)
	}
	if vt.size.Rows > 0 && vt.size.YPixel > 0 {
		cellHeight = max(1, vt.size.YPixel/vt.size.Rows)
	}
	return cellWidth, cellHeight
}
