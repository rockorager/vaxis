package vaxis

import (
	"os"
	"strings"

	"go.rockorager.dev/vaxis/log"
)

func (vx *Vaxis) applyQuirks() {
	id := string(vx.termID)
	switch {
	case strings.HasPrefix(id, "kitty"):
		log.Debug("kitty identified. applying quirks")
		vx.caps.noZWJ = true
	case id == "tmux 3.4":
		// tmux 3.4 has unicode support, but doesn't advertise via 2027
		vx.caps.unicodeCore = true

	}

	if !vx.caps.osc8 {
		switch {
		case strings.HasPrefix(id, "kitty"),
			strings.HasPrefix(id, "foot("),
			strings.HasPrefix(id, "WezTerm "),
			strings.HasPrefix(id, "ghostty "),
			strings.HasPrefix(id, "alacritty("),
			strings.HasPrefix(id, "contour "),
			strings.HasPrefix(id, "mintty "),
			strings.HasPrefix(id, "rio "),
			strings.HasPrefix(id, "tmux "),
			strings.HasPrefix(id, "iTerm2 "),
			strings.Contains(id, "VTE"):
			vx.caps.osc8 = true
		default:
			switch os.Getenv("TERM") {
			case "foot", "foot-extra", "xterm-kitty",
				"alacritty", "contour", "wezterm",
				"ghostty", "rio", "mintty":
				vx.caps.osc8 = true
			}
		}
	}

	if os.Getenv("ASCIINEMA_REC") != "" {
		// Asciinema doesn't support any advanced image protocols
		vx.graphicsProtocol = halfBlock
	}
	if os.Getenv("VAXIS_FORCE_LEGACY_SGR") != "" {
		sgrParamSeparator = ';'
		fgIndexSet = strings.ReplaceAll(fgIndexSet, ":", ";")
		fgRGBSet = strings.ReplaceAll(fgRGBSet, ":", ";")
		bgIndexSet = strings.ReplaceAll(bgIndexSet, ":", ";")
		bgRGBSet = strings.ReplaceAll(bgRGBSet, ":", ";")
	}
	if os.Getenv("VAXIS_FORCE_WCWIDTH") != "" {
		vx.caps.unicodeCore = false
		vx.caps.explicitWidth = false
	}
	if os.Getenv("VAXIS_FORCE_UNICODE") != "" {
		vx.caps.unicodeCore = true
	}
	if os.Getenv("VAXIS_FORCE_NOZWJ") != "" {
		vx.caps.noZWJ = true
		vx.caps.explicitWidth = false
	}
	if os.Getenv("VAXIS_DISABLE_NOZWJ") != "" {
		vx.caps.noZWJ = false
	}
	if os.Getenv("VAXIS_FORCE_XTWINOPS") != "" {
		vx.xtwinops = true
	}
}
