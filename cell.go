package vaxis

import (
	"fmt"
	"strings"

	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
)

// Cell represents a single cell in a terminal window. It contains a [Character]
// and a [Style], which fully defines the value. The zero value is rendered as
// an empty space
type Cell struct {
	Character
	Style
	// sixel marks if this cell has had a sixel graphic drawn on it.
	// If true, it won't be drawn in the render cycle.
	sixel bool
}

// Parses an SGR styled string into a slice of [Cell]s. This function does not
// depend on a Vaxis instance. The underlying cells will always be measured
// using correct unicode methods. If you are directly using these in a Vaxis
// window, you should either properly measure the graphemes based on your
// terminals capabilities or set the widths to 0 to enable vaxis to measure them
func ParseStyledString(s string) []Cell {
	r := strings.NewReader(s)
	parser := ansi.NewParser(r)
	defer parser.Close()
	cells := make([]Cell, 0, len(s)/2) // best effort
	style := Style{}
	for seq := range parser.Next() {
		switch seq := seq.(type) {
		case ansi.Print:
			cells = append(cells, Cell{
				Character: Character{
					Grapheme: seq.Grapheme,
					Width:    seq.Width,
				},
				Style: style,
			})
		case ansi.CSI:
			switch seq.Final {
			case 'm':
				parseSGR(seq.Parameters, &style)
			}
		case ansi.OSC:
			// TODO: OSC8 handling??
		default:
			// We don't handle anything else
		}
		parser.Finish(seq)
	}
	return cells
}

func EncodeCells(cells []Cell) string {
	bldr := &strings.Builder{}
	cursor := Style{}
	for _, next := range cells {
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
					fmt.Fprintf(bldr, fgIndexSet, ps[0])
				}
			case 3:
				fmt.Fprintf(bldr, fgRGBSet, ps[0], ps[1], ps[2])
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
					fmt.Fprintf(bldr, bgIndexSet, ps[0])
				}
			case 3:
				fmt.Fprintf(bldr, bgRGBSet, ps[0], ps[1], ps[2])
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

// parseSGR applies the SGR style to the passed in style
func parseSGR(params [][]int, style *Style) {
	if len(params) == 0 {
		params = [][]int{{0}}
	}
	for i := 0; i < len(params); i += 1 {
		switch params[i][0] {
		case 0:
			style.Attribute = 0
			style.Foreground = 0
			style.Background = 0
			style.UnderlineColor = 0
			style.UnderlineStyle = UnderlineOff
		case 1:
			style.Attribute |= AttrBold
		case 2:
			style.Attribute |= AttrDim
		case 3:
			style.Attribute |= AttrItalic
		case 4:
			switch len(params[i]) {
			case 1:
				// No subparams
				style.UnderlineStyle = UnderlineSingle
			case 2:
				// Has subparams
				switch params[i][1] {
				case 0:
					style.UnderlineStyle = UnderlineOff
				case 1:
					style.UnderlineStyle = UnderlineSingle
				case 2:
					style.UnderlineStyle = UnderlineDouble
				case 3:
					style.UnderlineStyle = UnderlineCurly
				case 4:
					style.UnderlineStyle = UnderlineDotted
				case 5:
					style.UnderlineStyle = UnderlineDashed
				}
			}
		case 5:
			style.Attribute |= AttrBlink
		case 7:
			style.Attribute |= AttrReverse
		case 8:
			style.Attribute |= AttrInvisible
		case 9:
			style.Attribute |= AttrStrikethrough
		case 21:
			// Double underlined, not supported
		case 22:
			style.Attribute &^= AttrBold
			style.Attribute &^= AttrDim
		case 23:
			style.Attribute &^= AttrItalic
		case 24:
			style.UnderlineStyle = UnderlineOff
		case 25:
			style.Attribute &^= AttrBlink
		case 27:
			style.Attribute &^= AttrReverse
		case 28:
			style.Attribute &^= AttrInvisible
		case 29:
			style.Attribute &^= AttrStrikethrough
		case 30, 31, 32, 33, 34, 35, 36, 37:
			style.Foreground = IndexColor(uint8(params[i][0] - 30))
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
					style.Foreground = RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					style.Foreground = IndexColor(uint8(params[i+2][0]))
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
				style.Foreground = IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.Foreground = RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.Foreground = RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 39:
			style.Foreground = 0
		case 40, 41, 42, 43, 44, 45, 46, 47:
			style.Background = IndexColor(uint8(params[i][0] - 40))
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
					style.Background = RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					style.Background = IndexColor(uint8(params[i+2][0]))
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
				style.Background = IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.Background = RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.Background = RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 49:
			style.Background = 0
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
					style.UnderlineColor = RGBColor(
						uint8(params[i+2][0]),
						uint8(params[i+3][0]),
						uint8(params[i+4][0]),
					)
					i += 4
				case 5:
					style.UnderlineColor = IndexColor(uint8(params[i+2][0]))
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
				style.UnderlineColor = IndexColor(uint8(params[i][2]))
			case 5:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.UnderlineColor = RGBColor(
					uint8(params[i][2]),
					uint8(params[i][3]),
					uint8(params[i][4]),
				)
			case 6:
				if params[i][1] != 2 {
					log.Error("[term] malformed SGR sequence")
					return
				}
				style.UnderlineColor = RGBColor(
					uint8(params[i][3]),
					uint8(params[i][4]),
					uint8(params[i][5]),
				)
			}
		case 59:
			style.UnderlineColor = 0
		case 90, 91, 92, 93, 94, 95, 96, 97:
			style.Foreground = IndexColor(uint8(params[i][0] - 90 + 8))
		case 100, 101, 102, 103, 104, 105, 106, 107:
			style.Background = IndexColor(uint8(params[i][0] - 100 + 8))
		}
	}
}
