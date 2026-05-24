package term

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.rockorager.dev/vaxis"
)

const (
	maxTitleLen         = 1024
	oscStringTerminator = "\x1b\\"
)

func oscColorReply(selector string, rgb []uint8) string {
	return fmt.Sprintf("\x1b]%s;rgb:%02x/%02x/%02x%s", selector, rgb[0], rgb[1], rgb[2], oscStringTerminator)
}

func (vt *Model) osc(data string) {
	selector, val, found := cutString(data, ";")
	if !found {
		selector = data
		val = ""
	}
	switch selector {
	case "0", "2":
		if !found {
			return
		}
		if !utf8.ValidString(val) {
			return
		}
		if len(val) > maxTitleLen {
			val = val[:maxTitleLen]
		}
		vt.title = val
		vt.postEvent(EventTitle(val))
	case "4":
		vt.oscPaletteColor(val)
	case "5":
		// Ghostty parses OSC 5 special colors, but the terminal handler
		// currently treats them as unsupported no-ops.
	case "7":
		vt.reportWorkingDirectory(val)
	case "8":
		if vt.OSC8 {
			params, url, ok := osc8(val)
			if !ok {
				return
			}
			vt.cursor.Hyperlink = url
			vt.cursor.HyperlinkParams = params
		}
	case "9":
		if cmd, arg, found := cutString(val, ";"); found && cmd == "9" {
			vt.reportWorkingDirectory(arg)
			return
		}
		if val == "12" || strings.HasPrefix(val, "12;") {
			vt.semanticPromptOSC("A")
			return
		}
		if progress, ok := parseConEmuProgress(val); ok {
			vt.postEvent(progress)
			return
		}
		if isConEmuOSC9Noop(val) {
			return
		}
		vt.postEvent(EventNotify{Body: val})
	case "10":
		if val == "?" {
			vx := vt.vx
			vt.enqueueReply(func(ctx context.Context) (string, bool) {
				if vx == nil {
					return "", false
				}
				rgb := vx.QueryForegroundContext(ctx).Params()
				if len(rgb) == 0 {
					return "", false
				}
				return oscColorReply("10", rgb), true
			})
			return
		}
		vt.oscDynamicColor(10, val)
	case "11":
		if val == "?" {
			vx := vt.vx
			vt.enqueueReply(func(ctx context.Context) (string, bool) {
				if vx == nil {
					return "", false
				}
				rgb := vx.QueryBackgroundContext(ctx).Params()
				if len(rgb) == 0 {
					return "", false
				}
				return oscColorReply("11", rgb), true
			})
			return
		}
		vt.oscDynamicColor(11, val)
	case "12":
		vt.oscDynamicColor(12, val)
	case "13":
		vt.oscDynamicColor(13, val)
	case "14":
		vt.oscDynamicColor(14, val)
	case "15":
		vt.oscDynamicColor(15, val)
	case "16":
		vt.oscDynamicColor(16, val)
	case "17":
		vt.oscDynamicColor(17, val)
	case "18":
		vt.oscDynamicColor(18, val)
	case "19":
		vt.oscDynamicColor(19, val)
	case "21":
		vt.oscKittyColor(val)
	case "104":
		vt.oscResetPalette(val)
	case "105":
		// Ghostty parses OSC 105 special-color resets as explicit no-ops.
	case "110":
		vt.oscResetDynamicColor(10, val)
	case "111":
		vt.oscResetDynamicColor(11, val)
	case "112":
		vt.oscResetDynamicColor(12, val)
	case "113":
		vt.oscResetDynamicColor(13, val)
	case "114":
		vt.oscResetDynamicColor(14, val)
	case "115":
		vt.oscResetDynamicColor(15, val)
	case "116":
		vt.oscResetDynamicColor(16, val)
	case "117":
		vt.oscResetDynamicColor(17, val)
	case "118":
		vt.oscResetDynamicColor(18, val)
	case "119":
		vt.oscResetDynamicColor(19, val)
	case "22":
		shape, ok := parseMouseShape(val)
		if !ok {
			return
		}
		vt.setMouseShape(shape)
	case "52":
		val, ok := osc52Data(val)
		if !ok || val == "?" {
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return
		}
		if vt.vx == nil {
			return
		}
		vt.vx.ClipboardPush(string(decodedBytes))
	case "66":
		// Ghostty parses Kitty text sizing, but currently treats it as an
		// unimplemented OSC callback.
	case "133":
		vt.semanticPromptOSC(val)
	case "1337":
		vt.osc1337(val)
	case "777":
		selector, val, found := cutString(val, ";")
		if !found {
			return
		}
		switch selector {
		case "notify":
			title, body, found := cutString(val, ";")
			if !found {
				return
			}
			vt.postEvent(EventNotify{
				Title: title,
				Body:  body,
			})
		}
	case "3008":
		// Ghostty parses hierarchical context signalling, but currently treats
		// it as an unimplemented OSC callback.
	case "5522":
		// Ghostty parses Kitty's clipboard protocol, but currently treats it as
		// an unimplemented OSC callback. This is separate from OSC 52.
	}
}

func (vt *Model) osc1337(val string) {
	key, value, hasValue := strings.Cut(val, "=")
	switch {
	case strings.EqualFold(key, "CurrentDir"):
		if !hasValue || value == "" {
			return
		}
		vt.reportWorkingDirectory(value)
	case strings.EqualFold(key, "Copy"):
		if !hasValue || !strings.HasPrefix(value, ":") {
			return
		}
		data := value[1:]
		if data == "" || data == "?" {
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(data)
		if err != nil || vt.vx == nil {
			return
		}
		vt.vx.ClipboardPush(string(decodedBytes))
	}
}

type semanticContent uint8

const (
	semanticOutput semanticContent = iota
	semanticPromptContent
	semanticInput
)

type semanticPrompt uint8

const (
	semanticPromptNone semanticPrompt = iota
	semanticPromptPrimary
	semanticPromptContinuation
)

type semanticPromptKind uint8

const (
	semanticPromptKindInitial semanticPromptKind = iota
	semanticPromptKindRight
	semanticPromptKindContinuation
	semanticPromptKindSecondary
)

type semanticPromptRedraw uint8

const (
	semanticPromptRedrawTrue semanticPromptRedraw = iota
	semanticPromptRedrawFalse
	semanticPromptRedrawLast
)

type semanticPromptClick uint8

const (
	semanticPromptClickNone semanticPromptClick = iota
	semanticPromptClickEvents
	semanticPromptClickLine
	semanticPromptClickMultiple
	semanticPromptClickConservativeVertical
	semanticPromptClickSmartVertical
)

func (vt *Model) semanticPromptOSC(val string) {
	cmd, options, hasOptions := strings.Cut(val, ";")
	if len(cmd) != 1 {
		return
	}

	switch cmd[0] {
	case 'L':
		if hasOptions {
			return
		}
		vt.semanticPromptFreshLine()
	case 'A', 'N':
		vt.semanticPromptFreshLine()
		vt.setSemanticPromptContent(semanticPromptKindOption(options))
		if redraw, ok := semanticPromptRedrawOption(options); ok {
			vt.shellRedrawsPrompt = redraw
		}
		vt.setSemanticPromptClickOption(options)
	case 'P':
		vt.setSemanticPromptContent(semanticPromptKindOption(options))
	case 'B':
		vt.setSemanticInput(false)
	case 'I':
		vt.setSemanticInput(true)
	case 'C':
		vt.setSemanticOutput()
		if vt.activeScreen.state != nil && vt.height() > 0 && vt.cursor.col == 0 {
			vt.activeScreen.row(vt.cursor.row).semanticPrompt = semanticPromptNone
		}
	case 'D':
		vt.setSemanticOutput()
	default:
		return
	}
}

func (vt *Model) setSemanticPromptClickOption(options string) {
	if enabled, ok := semanticPromptBoolOption(options, "click_events"); ok && enabled {
		vt.semanticPromptClick = semanticPromptClickEvents
		return
	}
	if click, ok := semanticPromptClickOption(options); ok {
		vt.semanticPromptClick = click
	}
}

func semanticPromptBoolOption(options string, key string) (bool, bool) {
	for {
		option, rest, found := strings.Cut(options, ";")
		optKey, value, hasValue := strings.Cut(option, "=")
		if hasValue && optKey == key {
			switch value {
			case "0":
				return false, true
			case "1":
				return true, true
			default:
				return false, false
			}
		}
		if !found {
			return false, false
		}
		options = rest
	}
}

func semanticPromptClickOption(options string) (semanticPromptClick, bool) {
	for {
		option, rest, found := strings.Cut(options, ";")
		key, value, hasValue := strings.Cut(option, "=")
		if hasValue && key == "cl" {
			switch value {
			case "line":
				return semanticPromptClickLine, true
			case "m":
				return semanticPromptClickMultiple, true
			case "v":
				return semanticPromptClickConservativeVertical, true
			case "w":
				return semanticPromptClickSmartVertical, true
			default:
				return semanticPromptClickNone, false
			}
		}
		if !found {
			return semanticPromptClickNone, false
		}
		options = rest
	}
}

func semanticPromptRedrawOption(options string) (semanticPromptRedraw, bool) {
	for {
		option, rest, found := strings.Cut(options, ";")
		key, value, hasValue := strings.Cut(option, "=")
		if hasValue && key == "redraw" {
			switch value {
			case "0":
				return semanticPromptRedrawFalse, true
			case "1":
				return semanticPromptRedrawTrue, true
			case "last":
				return semanticPromptRedrawLast, true
			default:
				return semanticPromptRedrawTrue, false
			}
		}
		if !found {
			return semanticPromptRedrawTrue, false
		}
		options = rest
	}
}

func semanticPromptKindOption(options string) semanticPromptKind {
	for {
		option, rest, found := strings.Cut(options, ";")
		if option == "k=r" {
			return semanticPromptKindRight
		}
		if option == "k=c" {
			return semanticPromptKindContinuation
		}
		if option == "k=s" {
			return semanticPromptKindSecondary
		}
		if option == "k=i" {
			return semanticPromptKindInitial
		}
		if !found {
			return semanticPromptKindInitial
		}
		options = rest
	}
}

func (vt *Model) semanticPromptFreshLine() {
	if vt.activeScreen.state == nil || vt.height() == 0 {
		return
	}
	left := vt.margin.left
	if vt.cursor.col < vt.margin.left {
		left = 0
	}
	if vt.cursor.col == left {
		return
	}
	vt.cr()
	vt.ind()
}

func (vt *Model) setSemanticPromptContent(kind semanticPromptKind) {
	vt.cursor.semanticContent = semanticPromptContent
	vt.cursor.semanticClearEOL = false
	if vt.activeScreen.state == nil || vt.height() == 0 {
		return
	}
	rowPrompt := semanticPromptPrimary
	switch kind {
	case semanticPromptKindContinuation, semanticPromptKindSecondary:
		rowPrompt = semanticPromptContinuation
	}
	vt.activeScreen.row(vt.cursor.row).semanticPrompt = rowPrompt
}

func (vt *Model) setSemanticInput(clearEOL bool) {
	vt.cursor.semanticContent = semanticInput
	vt.cursor.semanticClearEOL = clearEOL
}

func (vt *Model) setSemanticOutput() {
	vt.cursor.semanticContent = semanticOutput
	vt.cursor.semanticClearEOL = false
}

func (vt *Model) markSemanticContinuation() {
	if vt.activeScreen.state == nil || vt.height() == 0 {
		return
	}
	if vt.cursor.semanticClearEOL {
		vt.setSemanticOutput()
		return
	}
	switch vt.cursor.semanticContent {
	case semanticPromptContent, semanticInput:
		vt.activeScreen.row(vt.cursor.row).semanticPrompt = semanticPromptContinuation
	}
}

func (vt *Model) clearPromptForRedraw() {
	if vt.mode.smcup || vt.shellRedrawsPrompt == semanticPromptRedrawFalse ||
		vt.cursor.semanticContent == semanticOutput ||
		vt.primaryScreen.state == nil || vt.primaryScreen.height() == 0 ||
		vt.cursor.row < 0 || vt.cursor.row >= row(vt.primaryScreen.height()) {
		return
	}

	bg := vt.cursor.Background
	right := column(vt.primaryScreen.width() - 1)
	if vt.shellRedrawsPrompt == semanticPromptRedrawLast {
		vt.primaryScreen.eraseRow(vt.cursor.row, 0, right, bg)
		return
	}

	start, ok := vt.promptRedrawStartRow()
	if !ok {
		return
	}
	for r := start; r < row(vt.primaryScreen.height()); r += 1 {
		vt.primaryScreen.eraseRow(r, 0, right, bg)
	}
}

func (vt *Model) promptRedrawStartRow() (row, bool) {
	for r := vt.cursor.row; r >= 0; r -= 1 {
		if vt.primaryScreen.row(r).semanticPrompt == semanticPromptPrimary {
			return r, true
		}
	}
	return 0, false
}

func (vt *Model) cursorIsAtPrompt() bool {
	if vt.mode.smcup || vt.activeScreen.state == nil || vt.height() == 0 {
		return false
	}
	if vt.activeScreen.row(vt.cursor.row).semanticPrompt != semanticPromptNone {
		return true
	}
	return vt.cursor.semanticContent == semanticPromptContent || vt.cursor.semanticContent == semanticInput
}

func isConEmuOSC9Noop(val string) bool {
	if val == "" {
		return false
	}
	switch val[0] {
	case '1':
		switch {
		case strings.HasPrefix(val, "1;"):
			return true
		case val == "10":
			return true
		case strings.HasPrefix(val, "10;"):
			return len(val) >= 4 && val[3] >= '0' && val[3] <= '3'
		case strings.HasPrefix(val, "11;"):
			return true
		case val == "12":
			return true
		}
	case '2', '3', '6', '7', '8':
		return len(val) >= 2 && val[1] == ';'
	case '4':
		return len(val) >= 3 && val[1] == ';' && val[2] >= '0' && val[2] <= '4'
	case '5':
		return true
	}
	return false
}

func parseConEmuProgress(val string) (EventProgress, bool) {
	if len(val) < 3 || val[0] != '4' || val[1] != ';' {
		return EventProgress{}, false
	}

	progress := EventProgress{}
	switch val[2] {
	case '0':
		progress.State = ProgressRemove
	case '1':
		progress.State = ProgressSet
		progress.HasProgress = true
	case '2':
		progress.State = ProgressError
	case '3':
		progress.State = ProgressIndeterminate
	case '4':
		progress.State = ProgressPause
	default:
		return EventProgress{}, false
	}

	switch progress.State {
	case ProgressRemove, ProgressIndeterminate:
		return progress, true
	}
	if len(val) < 4 || val[3] != ';' {
		return progress, true
	}
	n, err := strconv.Atoi(val[4:])
	if err != nil {
		progress.Progress = 0
		progress.HasProgress = false
		return progress, true
	}
	progress.Progress = min(max(n, 0), 100)
	progress.HasProgress = true
	return progress, true
}

func osc52Data(val string) (string, bool) {
	if val == "" {
		return "", false
	}
	if val[0] == ';' {
		return val[1:], true
	}
	if len(val) < 2 || val[1] != ';' {
		return "", false
	}
	return val[2:], true
}

func (vt *Model) reportWorkingDirectory(url string) {
	vt.workingDirectoryURL = url
	vt.postEvent(EventWorkingDirectory{URL: url})
}

func (vt *Model) oscPaletteColor(val string) {
	if val == "" {
		return
	}
	parts := strings.Split(val, ";")
	queryIndexesBuf := [16]uint8{}
	queryIndexes := queryIndexesBuf[:0]
loop:
	for i := 0; i < len(parts); i += 2 {
		if i+1 >= len(parts) {
			break
		}
		index, err := strconv.ParseUint(parts[i], 10, 9)
		if err != nil {
			break
		}
		if index >= 256 {
			if index <= 260 {
				continue
			}
			break
		}
		switch spec := parts[i+1]; spec {
		case "?":
			queryIndexes = append(queryIndexes, uint8(index))
		default:
			color, ok := parseOSCColor(spec)
			if !ok {
				break loop
			}
			vt.colors.setPalette(uint8(index), color)
		}
	}
	if len(queryIndexes) == 0 {
		return
	}

	vx := vt.vx
	vt.enqueueReply(func(ctx context.Context) (string, bool) {
		if vx == nil {
			return "", false
		}
		var b strings.Builder
		for _, index := range queryIndexes {
			rgb := vx.QueryColorContext(ctx, vaxis.IndexColor(index)).Params()
			if len(rgb) == 0 {
				continue
			}
			b.WriteString(oscColorReply(fmt.Sprintf("4;%d", index), rgb))
		}
		resp := b.String()
		return resp, resp != ""
	})
}

func (vt *Model) oscResetPalette(val string) {
	if strings.Trim(val, ";") == "" {
		vt.colors.resetAllPalette()
		return
	}
	parts := strings.Split(val, ";")
	for _, part := range parts {
		if part == "" {
			continue
		}
		index, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			continue
		}
		vt.colors.resetPalette(uint8(index))
	}
}

func (vt *Model) oscDynamicColor(kind int, val string) {
	for _, spec := range strings.Split(val, ";") {
		if spec == "" {
			continue
		}
		target := vt.colors.dynamic(kind)
		if target == nil {
			return
		}
		if spec == "?" {
			kind += 1
			continue
		}
		color, ok := parseOSCColor(spec)
		if !ok {
			return
		}
		target.setColor(color)
		kind += 1
	}
}

func (vt *Model) oscResetDynamicColor(kind int, val string) {
	if strings.Trim(val, ";") != "" {
		return
	}
	if color := vt.colors.dynamic(kind); color != nil {
		color.reset()
	}
}

func (vt *Model) oscKittyColor(val string) {
	for _, part := range strings.Split(val, ";") {
		key, spec, found := strings.Cut(part, "=")
		if key == "" {
			continue
		}
		if !found {
			spec = ""
		}
		spec = strings.Trim(spec, " ")
		if spec == "" {
			vt.resetKittyColor(key)
			continue
		}
		if spec == "?" {
			continue
		}
		color, ok := parseOSCColor(spec)
		if !ok {
			continue
		}
		vt.setKittyColor(key, color)
	}
}

func (vt *Model) setKittyColor(key string, color vaxis.Color) {
	switch key {
	case "foreground":
		vt.colors.foreground.setColor(color)
	case "background":
		vt.colors.background.setColor(color)
	case "cursor":
		vt.colors.cursor.setColor(color)
	default:
		index, err := strconv.ParseUint(key, 10, 8)
		if err == nil {
			vt.colors.setPalette(uint8(index), color)
		}
	}
}

func (vt *Model) resetKittyColor(key string) {
	switch key {
	case "foreground":
		vt.colors.foreground.reset()
	case "background":
		vt.colors.background.reset()
	case "cursor":
		vt.colors.cursor.reset()
	default:
		index, err := strconv.ParseUint(key, 10, 8)
		if err == nil {
			vt.colors.resetPalette(uint8(index))
		}
	}
}

func parseMouseShape(s string) (vaxis.MouseShape, bool) {
	switch s {
	case "default", "left_ptr":
		return vaxis.MouseShapeDefault, true
	case "context-menu":
		return vaxis.MouseShapeContextMenu, true
	case "help", "question_arrow":
		return vaxis.MouseShapeHelp, true
	case "pointer", "hand":
		return vaxis.MouseShapeClickable, true
	case "progress", "left_ptr_watch":
		return vaxis.MouseShapeBusyBackground, true
	case "wait", "watch":
		return vaxis.MouseShapeBusy, true
	case "cell":
		return vaxis.MouseShapeCell, true
	case "crosshair", "cross":
		return vaxis.MouseShapeCrosshair, true
	case "text", "xterm":
		return vaxis.MouseShapeTextInput, true
	case "vertical-text":
		return vaxis.MouseShapeVerticalText, true
	case "alias", "dnd-link":
		return vaxis.MouseShapeAlias, true
	case "copy", "dnd-copy":
		return vaxis.MouseShapeCopy, true
	case "move", "dnd-move":
		return vaxis.MouseShapeMove, true
	case "no-drop", "dnd-no-drop":
		return vaxis.MouseShapeNoDrop, true
	case "not-allowed", "crossed_circle":
		return vaxis.MouseShapeNotAllowed, true
	case "grab", "hand1":
		return vaxis.MouseShapeGrab, true
	case "grabbing":
		return vaxis.MouseShapeGrabbing, true
	case "all-scroll", "fleur":
		return vaxis.MouseShapeAllScroll, true
	case "col-resize":
		return vaxis.MouseShapeResizeColumn, true
	case "row-resize":
		return vaxis.MouseShapeResizeRow, true
	case "n-resize", "top_side":
		return vaxis.MouseShapeResizeNorth, true
	case "e-resize", "right_side":
		return vaxis.MouseShapeResizeEast, true
	case "s-resize", "bottom_side":
		return vaxis.MouseShapeResizeSouth, true
	case "w-resize", "left_side":
		return vaxis.MouseShapeResizeWest, true
	case "ne-resize", "top_right_corner":
		return vaxis.MouseShapeResizeNorthEast, true
	case "nw-resize", "top_left_corner":
		return vaxis.MouseShapeResizeNorthWest, true
	case "se-resize", "bottom_right_corner":
		return vaxis.MouseShapeResizeSouthEast, true
	case "sw-resize", "bottom_left_corner":
		return vaxis.MouseShapeResizeSouthWest, true
	case "ew-resize":
		return vaxis.MouseShapeResizeHorizontal, true
	case "ns-resize":
		return vaxis.MouseShapeResizeVertical, true
	case "nesw-resize":
		return vaxis.MouseShapeResizeNESW, true
	case "nwse-resize":
		return vaxis.MouseShapeResizeNWSE, true
	case "zoom-in":
		return vaxis.MouseShapeZoomIn, true
	case "zoom-out":
		return vaxis.MouseShapeZoomOut, true
	default:
		return "", false
	}
}

// parses an osc8 payload into params and URL.
func osc8(val string) (string, string, bool) {
	// OSC 8 ; params ; url ST
	// params: key1=value1:key2=value2
	params, url, found := cutString(val, ";")
	if !found {
		return "", "", false
	}
	id := osc8ID(params)
	if url == "" && id != "" {
		return "", "", false
	}
	if url == "" {
		return "", "", true
	}
	if id == "" {
		return "", url, true
	}
	return "id=" + id, url, true
}

func osc8ID(params string) string {
	for _, param := range strings.Split(params, ":") {
		key, val, found := cutString(param, "=")
		if !found {
			continue
		}
		switch key {
		case "id":
			return val
		}
	}
	return ""
}

// Copied from stdlib to here for go 1.16 compat
func cutString(s string, sep string) (before string, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
