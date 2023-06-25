package term

import (
	"git.sr.ht/~rockorager/rtk"
)

func (vt *Model) sgr(params [][]int) {
	if len(params) == 0 {
		params = [][]int{{0}}
	}
	for i := 0; i < len(params); i += 1 {
		switch params[i][0] {
		case 0:
			vt.cursor.attrs = 0
			vt.cursor.fg = 0
			vt.cursor.bg = 0
		case 1:
			vt.cursor.attrs |= rtk.AttrBold
		case 2:
			vt.cursor.attrs |= rtk.AttrDim
		case 3:
			vt.cursor.attrs |= rtk.AttrItalic
		case 4:
			vt.cursor.attrs |= rtk.AttrUnderline
		case 5:
			vt.cursor.attrs |= rtk.AttrBlink
		case 7:
			vt.cursor.attrs |= rtk.AttrReverse
		case 8:
			vt.cursor.attrs |= rtk.AttrInvisible
		case 9:
			vt.cursor.attrs |= rtk.AttrStrikethrough
		case 21:
			// Double underlined, not supported
		case 22:
			vt.cursor.attrs &^= rtk.AttrBold
			vt.cursor.attrs &^= rtk.AttrDim
		case 23:
			vt.cursor.attrs &^= rtk.AttrItalic
		case 24:
			vt.cursor.attrs &^= rtk.AttrUnderline
		case 25:
			vt.cursor.attrs &^= rtk.AttrBlink
		case 27:
			vt.cursor.attrs &^= rtk.AttrReverse
		case 28:
			vt.cursor.attrs &^= rtk.AttrInvisible
		case 29:
			vt.cursor.attrs &^= rtk.AttrStrikethrough
		case 30, 31, 32, 33, 34, 35, 36, 37:
			vt.cursor.fg = rtk.IndexColor(uint8(params[i][0] - 30))
		case 38:
			switch len(params[i]) {
			case 1:
				if len(params[i:]) < 3 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				switch params[i+1][0] {
				case 2:
					if len(params[i:]) < 5 {
						vt.Logger.Error("[term] malformed SGR sequence")
						return
					}
					vt.cursor.fg = rtk.RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					vt.cursor.fg = rtk.IndexColor(uint8(params[i+2][0]))
					i += 2
				default:
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
			case 3:
				if params[i][1] != 5 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.fg = rtk.IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.fg = rtk.RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.fg = rtk.RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 39:
			vt.cursor.fg = 0
		case 40, 41, 42, 43, 44, 45, 46, 47:
			vt.cursor.bg = rtk.IndexColor(uint8(params[i][0] - 40))
		case 48:
			switch len(params[i]) {
			case 1:
				if len(params[i:]) < 3 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				switch params[i+1][0] {
				case 2:
					if len(params[i:]) < 5 {
						vt.Logger.Error("[term] malformed SGR sequence")
						return
					}
					vt.cursor.bg = rtk.RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					vt.cursor.bg = rtk.IndexColor(uint8(params[i+2][0]))
					i += 2
				default:
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
			case 3:
				if params[i][1] != 5 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.bg = rtk.IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.bg = rtk.RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					vt.Logger.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.bg = rtk.RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 49:
			vt.cursor.bg = 0
		case 90, 91, 92, 93, 94, 95, 96, 97:
			vt.cursor.fg = rtk.IndexColor(uint8(params[i][0] - 90 + 8))
		case 100, 101, 102, 103, 104, 105, 106, 107:
			vt.cursor.bg = rtk.IndexColor(uint8(params[i][0] - 100 + 8))
		}
	}
}
