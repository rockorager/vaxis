package term

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/log"
)

func (vt *Model) sgr(params [][]int) {
	if len(params) == 0 {
		params = [][]int{{0}}
	}
	for i := 0; i < len(params); i += 1 {
		switch params[i][0] {
		case 0:
			vt.cursor.Attribute = 0
			vt.cursor.Foreground = 0
			vt.cursor.Background = 0
			vt.cursor.UnderlineColor = 0
			vt.cursor.UnderlineStyle = vaxis.UnderlineOff
		case 1:
			vt.cursor.Attribute |= vaxis.AttrBold
		case 2:
			vt.cursor.Attribute |= vaxis.AttrDim
		case 3:
			vt.cursor.Attribute |= vaxis.AttrItalic
		case 4:
			switch len(params[i]) {
			case 1:
				vt.cursor.UnderlineStyle = vaxis.UnderlineSingle
			case 2:
				switch params[i][1] {
				case 0:
					vt.cursor.UnderlineStyle = vaxis.UnderlineOff
				case 1:
					vt.cursor.UnderlineStyle = vaxis.UnderlineSingle
				case 2:
					vt.cursor.UnderlineStyle = vaxis.UnderlineDouble
				case 3:
					vt.cursor.UnderlineStyle = vaxis.UnderlineCurly
				case 4:
					vt.cursor.UnderlineStyle = vaxis.UnderlineDotted
				case 5:
					vt.cursor.UnderlineStyle = vaxis.UnderlineDashed
				}
			}
		case 5:
			vt.cursor.Attribute |= vaxis.AttrBlink
		case 7:
			vt.cursor.Attribute |= vaxis.AttrReverse
		case 8:
			vt.cursor.Attribute |= vaxis.AttrInvisible
		case 9:
			vt.cursor.Attribute |= vaxis.AttrStrikethrough
		case 21:
			// Double underlined, not supported
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
			vt.cursor.Foreground = vaxis.IndexColor(uint8(params[i][0] - 30))
		case 38:
			switch len(params[i]) {
			case 1:
				if len(params[i:]) < 3 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				switch params[i+1][0] {
				case 2:
					if len(params[i:]) < 5 {
						log.Error("[term] malformed SGR sequence")
						return
					}
					vt.cursor.Foreground = vaxis.RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					vt.cursor.Foreground = vaxis.IndexColor(uint8(params[i+2][0]))
					i += 2
				default:
					log.Error("[term] malformed SGR sequence")
					return
				}
			case 3:
				if params[i][1] != 5 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Foreground = vaxis.IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Foreground = vaxis.RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Foreground = vaxis.RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 39:
			vt.cursor.Foreground = 0
		case 40, 41, 42, 43, 44, 45, 46, 47:
			vt.cursor.Background = vaxis.IndexColor(uint8(params[i][0] - 40))
		case 48:
			switch len(params[i]) {
			case 1:
				if len(params[i:]) < 3 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				switch params[i+1][0] {
				case 2:
					if len(params[i:]) < 5 {
						log.Error("[term] malformed SGR sequence")
						return
					}
					vt.cursor.Background = vaxis.RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					vt.cursor.Background = vaxis.IndexColor(uint8(params[i+2][0]))
					i += 2
				default:
					log.Error("[term] malformed SGR sequence")
					return
				}
			case 3:
				if params[i][1] != 5 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Background = vaxis.IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Background = vaxis.RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.Background = vaxis.RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 49:
			vt.cursor.Background = 0
		case 58:
			switch len(params[i]) {
			case 1:
				if len(params[i:]) < 3 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				switch params[i+1][0] {
				case 2:
					if len(params[i:]) < 5 {
						log.Error("[term] malformed SGR sequence")
						return
					}
					vt.cursor.UnderlineColor = vaxis.RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					vt.cursor.UnderlineColor = vaxis.IndexColor(uint8(params[i+2][0]))
					i += 2
				default:
					log.Error("[term] malformed SGR sequence")
					return
				}
			case 3:
				if params[i][1] != 5 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.UnderlineColor = vaxis.IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.UnderlineColor = vaxis.RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				vt.cursor.UnderlineColor = vaxis.RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 90, 91, 92, 93, 94, 95, 96, 97:
			vt.cursor.Foreground = vaxis.IndexColor(uint8(params[i][0] - 90 + 8))
		case 100, 101, 102, 103, 104, 105, 106, 107:
			vt.cursor.Background = vaxis.IndexColor(uint8(params[i][0] - 100 + 8))
		}
	}
}
