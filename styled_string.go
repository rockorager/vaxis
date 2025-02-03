package vaxis

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rivo/uniseg"
)

const (
	ssFgIndexSet = "\x1b[38:5:%dm"
	ssFgRGBSet   = "\x1b[38:2:%d:%d:%dm"
	ssBgIndexSet = "\x1b[48:5:%dm"
	ssBgRGBSet   = "\x1b[48:2:%d:%d:%dm"
)

type StyledString struct {
	Cells []Cell
}

func (vx *Vaxis) NewStyledString(s string, defaultStyle Style) *StyledString {
	ss := &StyledString{
		Cells: make([]Cell, 0, len(s)),
	}
	style := defaultStyle
	width := 0
	grapheme := ""
	seq := ""
	for len(s) > 0 {
		switch {
		case strings.HasPrefix(s, "\x1b["):
			s = strings.TrimPrefix(s, "\x1b[")
			seq, s, _ = strings.Cut(s, "m")
			if s == "" {
				// we don't need to process this sequence since
				// we don't have anything after it
				return ss
			}
			if seq == "" {
				style = defaultStyle
				continue
			}
			params := strings.Split(seq, ";")
			for _, param := range params {
				subs := strings.Split(param, ":")
				switch subs[0] {
				case "0":
					style = defaultStyle
				case "1":
					style.Attribute |= AttrBold
				case "2":
					style.Attribute |= AttrDim
				case "3":
					style.Attribute |= AttrItalic
				case "4":
					if len(subs) > 1 {
						switch subs[1] {
						case "0":
							style.UnderlineStyle = UnderlineOff
						case "1":
							style.UnderlineStyle = UnderlineSingle
						case "2":
							style.UnderlineStyle = UnderlineDouble
						case "3":
							style.UnderlineStyle = UnderlineCurly
						case "4":
							style.UnderlineStyle = UnderlineDotted
						case "5":
							style.UnderlineStyle = UnderlineDashed
						}
					} else {
						style.UnderlineStyle = UnderlineSingle
					}
				case "5":
					style.Attribute |= AttrBlink
				case "7":
					style.Attribute |= AttrReverse
				case "8":
					style.Attribute |= AttrInvisible
				case "9":
					style.Attribute |= AttrStrikethrough
				case "22":
					style.Attribute &^= AttrBold
					style.Attribute &^= AttrDim
				case "23":
					style.Attribute &^= AttrItalic
				case "24":
					style.UnderlineStyle = UnderlineOff
				case "25":
					style.Attribute &^= AttrBlink
				case "27":
					style.Attribute &^= AttrReverse
				case "28":
					style.Attribute &^= AttrInvisible
				case "29":
					style.Attribute &^= AttrStrikethrough
				case "30":
					style.Foreground = IndexColor(0)
				case "31":
					style.Foreground = IndexColor(1)
				case "32":
					style.Foreground = IndexColor(2)
				case "33":
					style.Foreground = IndexColor(3)
				case "34":
					style.Foreground = IndexColor(4)
				case "35":
					style.Foreground = IndexColor(5)
				case "36":
					style.Foreground = IndexColor(6)
				case "37":
					style.Foreground = IndexColor(7)
				case "38":
					switch len(subs) {
					case 3:
						idx, _ := strconv.Atoi(subs[2])
						style.Foreground = IndexColor(uint8(idx))
					case 5:
						r, _ := strconv.Atoi(subs[2])
						g, _ := strconv.Atoi(subs[3])
						b, _ := strconv.Atoi(subs[4])
						style.Foreground = RGBColor(uint8(r), uint8(g), uint8(b))
					}
				case "39":
					style.Foreground = 0
				case "40":
					style.Background = IndexColor(0)
				case "41":
					style.Background = IndexColor(1)
				case "42":
					style.Background = IndexColor(2)
				case "43":
					style.Background = IndexColor(3)
				case "44":
					style.Background = IndexColor(4)
				case "45":
					style.Background = IndexColor(5)
				case "46":
					style.Background = IndexColor(6)
				case "47":
					style.Background = IndexColor(7)
				case "48":
					switch len(subs) {
					case 3:
						idx, _ := strconv.Atoi(subs[2])
						style.Background = IndexColor(uint8(idx))
					case 5:
						r, _ := strconv.Atoi(subs[2])
						g, _ := strconv.Atoi(subs[3])
						b, _ := strconv.Atoi(subs[4])
						style.Background = RGBColor(uint8(r), uint8(g), uint8(b))
					}
				case "49":
					style.Background = 0
				case "58":
					switch len(subs) {
					case 3:
						idx, _ := strconv.Atoi(subs[2])
						style.UnderlineColor = IndexColor(uint8(idx))
					case 5:
						r, _ := strconv.Atoi(subs[2])
						g, _ := strconv.Atoi(subs[3])
						b, _ := strconv.Atoi(subs[4])
						style.UnderlineColor = RGBColor(uint8(r), uint8(g), uint8(b))
					}
				case "90":
					style.Foreground = IndexColor(8)
				case "91":
					style.Foreground = IndexColor(9)
				case "92":
					style.Foreground = IndexColor(10)
				case "93":
					style.Foreground = IndexColor(11)
				case "94":
					style.Foreground = IndexColor(12)
				case "95":
					style.Foreground = IndexColor(13)
				case "96":
					style.Foreground = IndexColor(14)
				case "97":
					style.Foreground = IndexColor(15)
				case "100":
					style.Background = IndexColor(8)
				case "101":
					style.Background = IndexColor(9)
				case "102":
					style.Background = IndexColor(10)
				case "103":
					style.Background = IndexColor(11)
				case "104":
					style.Background = IndexColor(12)
				case "105":
					style.Background = IndexColor(13)
				case "106":
					style.Background = IndexColor(14)
				case "107":
					style.Background = IndexColor(15)
				}
			}
		default:
			grapheme, s, width, _ = uniseg.FirstGraphemeClusterInString(s, -1)
			switch {
			case vx.caps.unicodeCore || vx.caps.explicitWidth:
				// we're done
			case vx.caps.noZWJ:
				width = gwidth(grapheme, noZWJ)
			default:
				width = gwidth(grapheme, wcwidth)
			}
			ss.Cells = append(ss.Cells, Cell{
				Character: Character{
					Grapheme: grapheme,
					Width:    width,
				},
				Style: style,
			})
		}
	}

	return ss
}

// Returns the rendered width of the styled string
func (ss *StyledString) Len() int {
	total := 0
	for _, ch := range ss.Cells {
		total += ch.Width
	}
	return total
}

func (ss *StyledString) Encode() string {
	bldr := &strings.Builder{}
	cursor := Style{}
	for _, next := range ss.Cells {
		if cursor.Foreground != next.Foreground {
			fg := next.Foreground
			ps := fg.Params()
			switch len(ps) {
			case 0:
				_, _ = bldr.WriteString(fgReset)
			case 1:
				switch {
				case ps[0] < 8:
					fmt.Fprintf(bldr, fgSet, ps[0])
				case ps[0] < 16:
					fmt.Fprintf(bldr, fgBrightSet, ps[0]-8)
				default:
					fmt.Fprintf(bldr, ssFgIndexSet, ps[0])
				}
			case 3:
				fmt.Fprintf(bldr, ssFgRGBSet, ps[0], ps[1], ps[2])
			}
		}

		if cursor.Background != next.Background {
			bg := next.Background
			ps := bg.Params()
			switch len(ps) {
			case 0:
				_, _ = bldr.WriteString(bgReset)
			case 1:
				switch {
				case ps[0] < 8:
					fmt.Fprintf(bldr, bgSet, ps[0])
				case ps[0] < 16:
					fmt.Fprintf(bldr, bgBrightSet, ps[0]-8)
				default:
					fmt.Fprintf(bldr, ssBgIndexSet, ps[0])
				}
			case 3:
				fmt.Fprintf(bldr, ssBgRGBSet, ps[0], ps[1], ps[2])
			}
		}

		if cursor.UnderlineColor != next.UnderlineColor {
			ul := next.UnderlineColor
			ps := ul.Params()
			switch len(ps) {
			case 0:
				_, _ = bldr.WriteString(ulColorReset)
			case 1:
				_, _ = fmt.Fprintf(bldr, ulIndexSet, ps[0])
			case 3:
				_, _ = fmt.Fprintf(bldr, ulRGBSet, ps[0], ps[1], ps[2])
			}
		}

		if cursor.Attribute != next.Attribute {
			attr := cursor.Attribute
			// find the ones that have changed
			dAttr := attr ^ next.Attribute
			// If the bit is changed and in next, it was
			// turned on
			on := dAttr & next.Attribute

			if on&AttrBold != 0 {
				_, _ = bldr.WriteString(boldSet)
			}
			if on&AttrDim != 0 {
				_, _ = bldr.WriteString(dimSet)
			}
			if on&AttrItalic != 0 {
				_, _ = bldr.WriteString(italicSet)
			}
			if on&AttrBlink != 0 {
				_, _ = bldr.WriteString(blinkSet)
			}
			if on&AttrReverse != 0 {
				_, _ = bldr.WriteString(reverseSet)
			}
			if on&AttrInvisible != 0 {
				_, _ = bldr.WriteString(hiddenSet)
			}
			if on&AttrStrikethrough != 0 {
				_, _ = bldr.WriteString(strikethroughSet)
			}

			// If the bit is changed and is in previous, it
			// was turned off
			off := dAttr & attr
			if off&AttrBold != 0 {
				// Normal intensity isn't in terminfo
				_, _ = bldr.WriteString(boldDimReset)
				// Normal intensity turns off dim. If it
				// should be on, let's turn it back on
				if next.Attribute&AttrDim != 0 {
					_, _ = bldr.WriteString(dimSet)
				}
			}
			if off&AttrDim != 0 {
				// Normal intensity isn't in terminfo
				_, _ = bldr.WriteString(boldDimReset)
				// Normal intensity turns off bold. If it
				// should be on, let's turn it back on
				if next.Attribute&AttrBold != 0 {
					_, _ = bldr.WriteString(boldSet)
				}
			}
			if off&AttrItalic != 0 {
				_, _ = bldr.WriteString(italicReset)
			}
			if off&AttrBlink != 0 {
				// turn off blink isn't in terminfo
				_, _ = bldr.WriteString(blinkReset)
			}
			if off&AttrReverse != 0 {
				_, _ = bldr.WriteString(reverseReset)
			}
			if off&AttrInvisible != 0 {
				// turn off invisible isn't in terminfo
				_, _ = bldr.WriteString(hiddenReset)
			}
			if off&AttrStrikethrough != 0 {
				_, _ = bldr.WriteString(strikethroughReset)
			}
		}

		if cursor.UnderlineStyle != next.UnderlineStyle {
			ulStyle := next.UnderlineStyle
			_, _ = bldr.WriteString(tparm(ulStyleSet, ulStyle))
		}

		if cursor.Hyperlink != next.Hyperlink {
			link := next.Hyperlink
			linkPs := next.HyperlinkParams
			if link == "" {
				linkPs = ""
			}
			_, _ = bldr.WriteString(tparm(osc8, linkPs, link))
		}
		cursor = next.Style
		bldr.WriteString(next.Grapheme)
	}
	empty := Style{}
	if cursor != empty {
		bldr.WriteString(sgrReset)
	}
	return bldr.String()
}
