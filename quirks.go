package vaxis

import (
	"os"
	"strings"

	"git.sr.ht/~rockorager/vaxis/log"
)

func (vx *Vaxis) applyQuirks() {
	id := string(vx.termID)
	switch {
	case strings.HasPrefix(id, "kitty"):
		log.Debug("kitty identified. applying quirks")
		vx.caps.noZWJ = true
	}

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
	if os.Getenv("VAXIS_FORCE_NOZWJ") != "" {
		vx.caps.noZWJ = true
	}
	if os.Getenv("VAXIS_DISABLE_NOZWJ") != "" {
		vx.caps.noZWJ = false
	}
	if os.Getenv("VAXIS_FORCE_XTWINOPS") != "" {
		vx.xtwinops = true
	}
}
