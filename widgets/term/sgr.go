package term

import (
	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ansi"
)

func (vt *Model) sgr(seq ansi.CSI) {
	if seq.NumParameters == 0 {
		vt.resetSGR()
		return
	}

	var all []uint32
	if seq.NumParameters > len(seq.Parameters) {
		all = seq.ExtraParameters
	} else {
		all = seq.Parameters[:seq.NumParameters]
	}

	for i := 0; i < len(all); {
		next := i + 1
		for next < len(all) && seq.ColonAfter(next-1) {
			next += 1
		}
		params := all[i:next]
		if len(params) == 0 {
			i = next
			continue
		}
		if len(params) > 1 {
			switch params[0] {
			case 4, 38, 48, 58:
			default:
				i = next
				continue
			}
		}
		if consumed := vt.applySGRParam(all, i, params); consumed > 0 {
			i += consumed
			continue
		}
		i = next
	}
}

func (vt *Model) resetSGR() {
	vt.cursor.Attribute = 0
	vt.cursor.Foreground = 0
	vt.cursor.Background = 0
	vt.cursor.UnderlineColor = 0
	vt.cursor.UnderlineStyle = vaxis.UnderlineOff
}

func (vt *Model) applySGRParam(all []uint32, i int, params []uint32) int {
	switch params[0] {
	case 0:
		vt.resetSGR()
	case 1:
		vt.cursor.Attribute |= vaxis.AttrBold
	case 2:
		vt.cursor.Attribute |= vaxis.AttrDim
	case 3:
		vt.cursor.Attribute |= vaxis.AttrItalic
	case 4:
		switch len(params) {
		case 1:
			vt.cursor.UnderlineStyle = vaxis.UnderlineSingle
		case 2:
			vt.cursor.UnderlineStyle = underlineStyle(params[1])
		}
	case 5, 6:
		vt.cursor.Attribute |= vaxis.AttrBlink
	case 7:
		vt.cursor.Attribute |= vaxis.AttrReverse
	case 8:
		vt.cursor.Attribute |= vaxis.AttrInvisible
	case 9:
		vt.cursor.Attribute |= vaxis.AttrStrikethrough
	case 21:
		vt.cursor.UnderlineStyle = vaxis.UnderlineDouble
	case 22:
		vt.cursor.Attribute &^= vaxis.AttrBold
		vt.cursor.Attribute &^= vaxis.AttrDim
	case 23:
		vt.cursor.Attribute &^= vaxis.AttrItalic
	case 24:
		vt.cursor.UnderlineStyle = vaxis.UnderlineOff
	case 25:
		vt.cursor.Attribute &^= vaxis.AttrBlink
	case 27:
		vt.cursor.Attribute &^= vaxis.AttrReverse
	case 28:
		vt.cursor.Attribute &^= vaxis.AttrInvisible
	case 29:
		vt.cursor.Attribute &^= vaxis.AttrStrikethrough
	case 30, 31, 32, 33, 34, 35, 36, 37:
		vt.cursor.Foreground = vaxis.IndexColor(uint8(params[0] - 30))
	case 38:
		return vt.applySGRColor(all, i, params, sgrColorForeground)
	case 39:
		vt.cursor.Foreground = 0
	case 40, 41, 42, 43, 44, 45, 46, 47:
		vt.cursor.Background = vaxis.IndexColor(uint8(params[0] - 40))
	case 48:
		return vt.applySGRColor(all, i, params, sgrColorBackground)
	case 49:
		vt.cursor.Background = 0
	case 53:
		vt.cursor.Attribute |= vaxis.AttrOverline
	case 55:
		vt.cursor.Attribute &^= vaxis.AttrOverline
	case 58:
		return vt.applySGRColor(all, i, params, sgrColorUnderline)
	case 59:
		vt.cursor.UnderlineColor = 0
	case 90, 91, 92, 93, 94, 95, 96, 97:
		vt.cursor.Foreground = vaxis.IndexColor(uint8(params[0] - 90 + 8))
	case 100, 101, 102, 103, 104, 105, 106, 107:
		vt.cursor.Background = vaxis.IndexColor(uint8(params[0] - 100 + 8))
	}
	return 0
}

func underlineStyle(style uint32) vaxis.UnderlineStyle {
	switch style {
	case 0:
		return vaxis.UnderlineOff
	case 1:
		return vaxis.UnderlineSingle
	case 2:
		return vaxis.UnderlineDouble
	case 3:
		return vaxis.UnderlineCurly
	case 4:
		return vaxis.UnderlineDotted
	case 5:
		return vaxis.UnderlineDashed
	default:
		return vaxis.UnderlineSingle
	}
}

type sgrColorTarget uint8

const (
	sgrColorForeground sgrColorTarget = iota
	sgrColorBackground
	sgrColorUnderline
)

func (vt *Model) applySGRColor(all []uint32, i int, params []uint32, target sgrColorTarget) int {
	if len(params) == 1 {
		if len(all)-i < 2 {
			return 0
		}
		switch all[i+1] {
		case 2:
			if len(all)-i < 5 {
				return 0
			}
			vt.setSGRColor(target, vaxis.RGBColor(
				uint8(all[i+2]),
				uint8(all[i+3]),
				uint8(all[i+4]),
			))
			return 5
		case 5:
			if len(all)-i < 3 {
				return 0
			}
			vt.setSGRColor(target, vaxis.IndexColor(uint8(all[i+2])))
			return 3
		}
		return 0
	}

	switch len(params) {
	case 3:
		if params[1] != 5 {
			return 0
		}
		vt.setSGRColor(target, vaxis.IndexColor(uint8(params[2])))
	case 5:
		if params[1] != 2 {
			return 0
		}
		vt.setSGRColor(target, vaxis.RGBColor(
			uint8(params[2]),
			uint8(params[3]),
			uint8(params[4]),
		))
	case 6:
		if params[1] != 2 {
			return 0
		}
		vt.setSGRColor(target, vaxis.RGBColor(
			uint8(params[3]),
			uint8(params[4]),
			uint8(params[5]),
		))
	}
	return 0
}

func (vt *Model) setSGRColor(target sgrColorTarget, color vaxis.Color) {
	switch target {
	case sgrColorForeground:
		vt.cursor.Foreground = color
	case sgrColorBackground:
		vt.cursor.Background = color
	case sgrColorUnderline:
		vt.cursor.UnderlineColor = color
	}
}
