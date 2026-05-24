package term

import (
	"strconv"
	"strings"

	"go.rockorager.dev/vaxis"
)

type dynamicColor struct {
	color vaxis.Color
	set   bool
}

func (c *dynamicColor) setColor(color vaxis.Color) {
	c.color = color
	c.set = true
}

func (c *dynamicColor) reset() {
	*c = dynamicColor{}
}

type terminalColors struct {
	palette             [256]vaxis.Color
	paletteMask         [4]uint64
	paletteDirty        bool
	foreground          dynamicColor
	background          dynamicColor
	cursor              dynamicColor
	pointerForeground   dynamicColor
	pointerBackground   dynamicColor
	tektronixForeground dynamicColor
	tektronixBackground dynamicColor
	highlightBackground dynamicColor
	tektronixCursor     dynamicColor
	highlightForeground dynamicColor
}

func (c *terminalColors) setPalette(index uint8, color vaxis.Color) {
	c.palette[index] = color
	c.paletteMask[index/64] |= 1 << (index % 64)
	c.paletteDirty = true
}

func (c *terminalColors) resetPalette(index uint8) {
	c.palette[index] = 0
	c.paletteMask[index/64] &^= 1 << (index % 64)
	c.paletteDirty = true
}

func (c *terminalColors) resetAllPalette() {
	dirty := false
	for i := range c.palette {
		if c.paletteSet(uint8(i)) {
			dirty = true
		}
		c.palette[i] = 0
	}
	c.paletteMask = [4]uint64{}
	if dirty {
		c.paletteDirty = true
	}
}

func (c *terminalColors) paletteSet(index uint8) bool {
	return c.paletteMask[index/64]&(1<<(index%64)) != 0
}

func (c *terminalColors) dynamic(kind int) *dynamicColor {
	switch kind {
	case 10:
		return &c.foreground
	case 11:
		return &c.background
	case 12:
		return &c.cursor
	case 13:
		return &c.pointerForeground
	case 14:
		return &c.pointerBackground
	case 15:
		return &c.tektronixForeground
	case 16:
		return &c.tektronixBackground
	case 17:
		return &c.highlightBackground
	case 18:
		return &c.tektronixCursor
	case 19:
		return &c.highlightForeground
	default:
		return nil
	}
}

func parseOSCColor(s string) (vaxis.Color, bool) {
	s = strings.Trim(s, " ")
	if color, ok := oscNamedColor(strings.ToLower(s)); ok {
		return color, true
	}
	switch {
	case strings.HasPrefix(s, "#"):
		return parseOSCSharpColor(s[1:])
	case strings.HasPrefix(s, "rgb:"):
		return parseOSCRGBColor(s[4:], false)
	case strings.HasPrefix(s, "rgbi:"):
		return parseOSCRGBColor(s[5:], true)
	default:
		return 0, false
	}
}

func oscNamedColor(name string) (vaxis.Color, bool) {
	switch name {
	case "aliceblue":
		return vaxis.RGBColor(0xf0, 0xf8, 0xff), true
	case "black":
		return vaxis.RGBColor(0, 0, 0), true
	case "blue":
		return vaxis.RGBColor(0, 0, 255), true
	case "cyan", "aqua":
		return vaxis.RGBColor(0, 255, 255), true
	case "forestgreen":
		return vaxis.RGBColor(34, 139, 34), true
	case "green", "lime":
		return vaxis.RGBColor(0, 255, 0), true
	case "lawngreen":
		return vaxis.RGBColor(124, 252, 0), true
	case "magenta", "fuchsia":
		return vaxis.RGBColor(255, 0, 255), true
	case "medium spring green", "mediumspringgreen":
		return vaxis.RGBColor(0, 250, 154), true
	case "red":
		return vaxis.RGBColor(255, 0, 0), true
	case "white":
		return vaxis.RGBColor(255, 255, 255), true
	case "yellow":
		return vaxis.RGBColor(255, 255, 0), true
	default:
		return 0, false
	}
}

func parseOSCSharpColor(s string) (vaxis.Color, bool) {
	var size int
	switch len(s) {
	case 3:
		size = 1
	case 6:
		size = 2
	case 9:
		size = 3
	case 12:
		size = 4
	default:
		return 0, false
	}
	r, ok := parseOSCHexChannel(s[:size])
	if !ok {
		return 0, false
	}
	g, ok := parseOSCHexChannel(s[size : size*2])
	if !ok {
		return 0, false
	}
	b, ok := parseOSCHexChannel(s[size*2:])
	if !ok {
		return 0, false
	}
	return vaxis.RGBColor(r, g, b), true
}

func parseOSCRGBColor(s string, intensity bool) (vaxis.Color, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return 0, false
	}
	var rgb [3]uint8
	for i, part := range parts {
		var ok bool
		if intensity {
			rgb[i], ok = parseOSCIntensityChannel(part)
		} else {
			rgb[i], ok = parseOSCHexChannel(part)
		}
		if !ok {
			return 0, false
		}
	}
	return vaxis.RGBColor(rgb[0], rgb[1], rgb[2]), true
}

func parseOSCHexChannel(s string) (uint8, bool) {
	if len(s) == 0 || len(s) > 4 {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, false
	}
	max := uint64(0xf)
	for i := 1; i < len(s); i += 1 {
		max = max<<4 | 0xf
	}
	return uint8(v * 0xff / max), true
}

func parseOSCIntensityChannel(s string) (uint8, bool) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || v > 1 {
		return 0, false
	}
	return uint8(v * 255), true
}
