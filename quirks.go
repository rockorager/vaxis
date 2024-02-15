package vaxis

import (
	"os"
	"strings"
)

func (vx *Vaxis) applyQuirks() {
	// TODO: remove this when asciinema supports ':' delimiters in RGB
	if os.Getenv("ASCIINEMA_REC") != "" {
		fgIndexSet = strings.ReplaceAll(fgIndexSet, ":", ";")
		fgRGBSet = strings.ReplaceAll(fgRGBSet, ":", ";")
		bgIndexSet = strings.ReplaceAll(bgIndexSet, ":", ";")
		bgRGBSet = strings.ReplaceAll(bgRGBSet, ":", ";")
		// Asciinema also doesn't support any advanced image protocols
		vx.graphicsProtocol = halfBlock
	}
	if os.Getenv("VAXIS_FORCE_LEGACY_SGR") != "" {
		fgIndexSet = strings.ReplaceAll(fgIndexSet, ":", ";")
		fgRGBSet = strings.ReplaceAll(fgRGBSet, ":", ";")
		bgIndexSet = strings.ReplaceAll(bgIndexSet, ":", ";")
		bgRGBSet = strings.ReplaceAll(bgRGBSet, ":", ";")
	}
	if os.Getenv("VAXIS_FORCE_WCWIDTH") != "" {
		vx.caps.unicodeCore = false
	}
	if os.Getenv("VAXIS_FORCE_UNICODE") != "" {
		vx.caps.unicodeCore = true
	}
}
