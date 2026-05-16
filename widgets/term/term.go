package term

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
	"github.com/creack/pty"
	"github.com/rockorager/go-uucode"
)

type (
	column int
	row    int
)

// Model models a virtual terminal
type Model struct {
	// If true, OSC8 enables the output of OSC8 strings. Otherwise, any OSC8
	// sequences will be stripped
	OSC8 bool
	// Set the TERM environment variable to be passed to the command's
	// environment. If not set, xterm-256color will be used unless Kitty
	// keyboard passthrough is enabled.
	TERM string
	// EnableKittyKeyboard allows child applications to negotiate Kitty keyboard
	// protocol state through the terminal widget. Only enable this when the
	// host terminal supports Kitty keyboard encoding.
	EnableKittyKeyboard bool
	// AllowKeyboardActionMode allows ANSI KAM (SM 2) to suppress keyboard input.
	// It is disabled by default because applications can otherwise make the
	// terminal ignore user typing.
	AllowKeyboardActionMode bool
	// EnquiryResponse is written to the child PTY in response to ENQ (0x05).
	// Empty means ENQ is ignored.
	EnquiryResponse string

	mu sync.Mutex

	vx *vaxis.Vaxis

	activeScreen  screenBuffer
	altScreen     screenBuffer
	primaryScreen screenBuffer

	charsets            charsets
	cursor              cursor
	margin              margin
	mode                mode
	savedMode           mode
	tabStop             []column
	title               string
	workingDirectoryURL string
	mouseShape          vaxis.MouseShape
	theme               vaxis.ColorThemeMode
	colors              terminalColors
	shellRedrawsPrompt  semanticPromptRedraw
	semanticPromptClick semanticPromptClick
	size                vaxis.Resize
	status              statusDisplay

	previousChar    vaxis.Character
	hasPreviousChar bool

	primaryKittyKeyboard kittyKeyboardStack
	altKittyKeyboard     kittyKeyboardStack
	// lastCol is a flag indicating we printed in the last col
	lastCol bool
	// scrollOffset is the number of historical rows above the active screen
	// currently visible. Zero means the viewport is pinned to the active screen.
	scrollOffset int

	primaryState cursorState
	altState     cursorState

	cmd    *exec.Cmd
	dirty  bool
	parser *ansi.Parser
	pty    *os.File

	eventHandler func(vaxis.Event)
	events       chan vaxis.Event
	focused      int32
	graphics     []*Image
	timer        *time.Timer
	syncTimer    *time.Timer
	replyQueue   chan termReply
	replyCancel  context.CancelFunc
}

type cursorState struct {
	charsets charsets
	cursor   cursor
	decom    bool
	lastCol  bool
	saved    bool
}

type Option func(*Model)

var synchronizedOutputResetDelay = time.Second

func noopEventHandler(vaxis.Event) {}

// WithVaxis attaches the host Vaxis instance used to render this terminal.
// Kitty keyboard passthrough is enabled only when the host terminal advertised
// support to Vaxis.
func WithVaxis(vx *vaxis.Vaxis) Option {
	return func(m *Model) {
		m.vx = vx
		m.EnableKittyKeyboard = vx != nil && vx.CanKittyKeyboard()
	}
}

// WithKittyKeyboard controls Kitty keyboard passthrough directly. Most callers
// should prefer WithVaxis so passthrough follows detected host capabilities.
func WithKittyKeyboard(enabled bool) Option {
	return func(m *Model) {
		m.EnableKittyKeyboard = enabled
	}
}

// WithKeyboardActionMode controls whether ANSI KAM (SM 2) is honored for
// terminal input. When enabled, KAM suppresses key and paste input while set.
func WithKeyboardActionMode(enabled bool) Option {
	return func(m *Model) {
		m.AllowKeyboardActionMode = enabled
	}
}

func WithEnquiryResponse(response string) Option {
	return func(m *Model) {
		m.EnquiryResponse = response
	}
}

type margin struct {
	top    row
	bottom row
	left   column
	right  column
}

func New(opts ...Option) *Model {
	m := &Model{
		OSC8:         true,
		charsets:     defaultCharsets(),
		mode:         defaultMode(),
		primaryState: defaultCursorState(),
		altState:     defaultCursorState(),
		eventHandler: noopEventHandler,
		// Buffering to 2 events. If there is ever a case where one
		// sequence can trigger two events, this should be increased
		events:     make(chan vaxis.Event, 2),
		timer:      time.NewTimer(0),
		mouseShape: vaxis.MouseShapeTextInput,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.setDefaultTabStops()
	return m
}

func (vt *Model) defaultTERM() string {
	if vt.EnableKittyKeyboard {
		return "xterm-kitty"
	}
	return "xterm-256color"
}

func defaultCursorState() cursorState {
	return cursorState{
		charsets: defaultCharsets(),
	}
}

func (vt *Model) StartWithSize(cmd *exec.Cmd, width int, height int) error {
	if cmd == nil {
		return fmt.Errorf("no command to run")
	}
	vt.cmd = cmd

	if vt.TERM == "" {
		vt.TERM = vt.defaultTERM()
	}

	env := os.Environ()
	if cmd.Env != nil {
		env = cmd.Env
	}
	cmd.Env = append(env, "TERM="+vt.TERM)

	// Start the command with a pty.
	var err error
	winsize := pty.Winsize{
		Cols: uint16(width),
		Rows: uint16(height),
	}
	vt.pty, err = pty.StartWithAttrs(
		cmd,
		&winsize,
		&syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
			Ctty:    1,
		})
	if err != nil {
		return err
	}

	vt.resize(width, height)
	vt.startReplyWorker()
	vt.parser = ansi.NewParser(vt.pty, ansi.ParserModeOutput)
	go func() {
		defer vt.recover()
		for {
			select {
			case seq := <-vt.parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					err := cmd.Wait()
					vt.dispatchEvent(EventClosed{
						Term:  vt,
						Error: err,
					})
					return
				default:
					vt.update(seq)
				}
			case ev := <-vt.events:
				vt.dispatchEvent(ev)
			case <-vt.timer.C:
				vt.mu.Lock()
				vt.timer.Stop()
				vt.mu.Unlock()
				vt.dispatchEvent(vaxis.Redraw{})
			}
		}
	}()
	return nil
}

// Start starts the terminal with the specified command. Start returns when the
// command has been successfully started.
func (vt *Model) Start(cmd *exec.Cmd) error {
	return vt.StartWithSize(cmd, 80, 24)
}

// Update is called from the host application. This is user input
func (vt *Model) Update(msg vaxis.Event) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.invalidate()
	switch msg := msg.(type) {
	case vaxis.Key:
		if vt.handleViewportKey(msg) {
			return
		}
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		str := vt.encodeKey(msg)
		str = vt.linefeedModeInput(str)
		vt.writePtyString(str)
	case vaxis.PasteStartEvent:
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		if vt.mode.paste {
			vt.writePtyString("\x1B[200~")
			return
		}
	case vaxis.PasteEndEvent:
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		if vt.mode.paste {
			vt.writePtyString("\x1B[201~")
			return
		}
	case vaxis.Mouse:
		if promptClick := vt.handlePromptClick(msg); promptClick != "" {
			vt.writePtyString(promptClick)
			return
		}
		if vt.handleViewportMouse(msg) {
			return
		}
		mouse := vt.handleMouse(msg)
		vt.writePtyString(mouse)
		return
	case vaxis.ColorThemeUpdate:
		vt.theme = msg.Mode
		if vt.mode.colorScheme {
			vt.writePtyString(colorSchemeReport(msg.Mode))
			return
		}
	case vaxis.Resize:
		if vt.size != msg {
			vt.setSynchronizedOutput(false)
		}
		vt.size = msg
		vt.resize(msg.Cols, msg.Rows)
		vt.resizePty(msg.Cols, msg.Rows)
		if vt.mode.inBandSizeReports {
			vt.inBandSizeReport()
		}
	case vaxis.FocusIn:
		atomicStore(&vt.focused, true)
		if vt.mode.focusEvents {
			vt.reportFocus()
		}
	case vaxis.FocusOut:
		atomicStore(&vt.focused, false)
		if vt.mode.focusEvents {
			vt.reportFocus()
		}
	}
}

func (vt *Model) keyboardActionModeBlocksInput() bool {
	return vt.AllowKeyboardActionMode && vt.mode.kam
}

func (vt *Model) linefeedModeInput(s string) string {
	if !vt.mode.lnm || s == "" {
		return s
	}
	return strings.ReplaceAll(s, "\r", "\r\n")
}

func (vt *Model) setSynchronizedOutput(enabled bool) {
	vt.mode.synchronizedOutput = enabled
	if vt.syncTimer != nil {
		vt.syncTimer.Stop()
		vt.syncTimer = nil
	}
	if !enabled {
		return
	}
	vt.syncTimer = time.AfterFunc(synchronizedOutputResetDelay, func() {
		vt.mu.Lock()
		defer vt.mu.Unlock()
		if !vt.mode.synchronizedOutput {
			return
		}
		vt.mode.synchronizedOutput = false
		vt.invalidate()
	})
}

func (vt *Model) writePtyString(s string) {
	if vt.pty == nil || s == "" {
		return
	}
	_, _ = vt.pty.WriteString(s)
}

// only call invalidate while a lock is held
func (vt *Model) invalidate() {
	if vt.dirty {
		return
	}
	vt.dirty = true
	vt.timer.Reset(8 * time.Millisecond)
}

// update is called from the PTY routine...this is updating the internal model
// based on the underlying process
func (vt *Model) update(seq ansi.Sequence) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	defer vt.invalidate()
	applySequence(vt, seq)
}

func (vt *Model) String() string {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	return vt.activeScreen.String()
}

func (vt *Model) maxScrollOffset() int {
	if vt.mode.smcup {
		return 0
	}
	return vt.primaryScreen.scrollbackLen()
}

func (vt *Model) clampScrollOffset() {
	maxOffset := vt.maxScrollOffset()
	if vt.scrollOffset > maxOffset {
		vt.scrollOffset = maxOffset
	}
	if vt.scrollOffset < 0 {
		vt.scrollOffset = 0
	}
}

func (vt *Model) scrollViewport(lines int) bool {
	if vt.mode.smcup {
		return false
	}
	before := vt.scrollOffset
	vt.scrollOffset += lines
	vt.clampScrollOffset()
	return vt.scrollOffset != before
}

func (vt *Model) handleViewportKey(msg vaxis.Key) bool {
	if msg.Modifiers&vaxis.ModShift == 0 {
		return false
	}
	page := max(1, vt.height()-1)
	switch msg.Keycode {
	case vaxis.KeyPgUp:
		vt.scrollViewport(page)
		return true
	case vaxis.KeyPgDown:
		vt.scrollViewport(-page)
		return true
	default:
		return false
	}
}

func (vt *Model) handleViewportMouse(msg vaxis.Mouse) bool {
	if vt.mode.mouseEvent != mouseEventNone {
		return false
	}
	if vt.mode.smcup {
		return false
	}
	switch msg.Button {
	case vaxis.MouseWheelUp:
		vt.scrollViewport(3)
		return true
	case vaxis.MouseWheelDown:
		vt.scrollViewport(-3)
		return true
	default:
		return false
	}
}

func (vt *Model) visibleLine(r int) []cell {
	historyLen := vt.activeScreen.scrollbackLen()
	if vt.scrollOffset == 0 || historyLen == 0 {
		return vt.activeScreen.line(row(r))
	}
	source := historyLen - vt.scrollOffset + r
	if source < historyLen {
		line, ok := vt.activeScreen.scrollbackLine(source)
		if ok {
			return line.cells
		}
	}
	activeRow := source - historyLen
	if activeRow < 0 {
		activeRow = 0
	}
	if activeRow >= vt.height() {
		activeRow = vt.height() - 1
	}
	return vt.activeScreen.line(row(activeRow))
}

func (vt *Model) postEvent(ev vaxis.Event) {
	select {
	case vt.events <- ev:
	default:
		log.Warn("[term] event queue full; dropping %T", ev)
	}
}

func (vt *Model) setMouseShape(shape vaxis.MouseShape) {
	if vt.mouseShape == shape {
		return
	}
	vt.mouseShape = shape
	vt.postEvent(EventMouseShape{Shape: shape})
}

func (vt *Model) Attach(fn func(ev vaxis.Event)) {
	if fn == nil {
		fn = noopEventHandler
	}
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.eventHandler = fn
}

func (vt *Model) Detach() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.eventHandler = noopEventHandler
}

func (vt *Model) dispatchEvent(ev vaxis.Event) {
	vt.mu.Lock()
	handler := vt.eventHandler
	vt.mu.Unlock()
	if handler == nil {
		handler = noopEventHandler
	}
	handler(ev)
}

func (vt *Model) recover() {
	err := recover()
	if err == nil {
		return
	}
	ret := strings.Builder{}
	ret.WriteString(fmt.Sprintf("cursor row=%d col=%d\n", vt.cursor.row, vt.cursor.col))
	ret.WriteString(fmt.Sprintf("margin left=%d right=%d\n", vt.margin.left, vt.margin.right))
	ret.WriteString(fmt.Sprintf("%s\n", err))
	ret.Write(debug.Stack())

	vt.postEvent(EventPanic(fmt.Errorf(ret.String())))
	vt.Close()
}

func (vt *Model) Resize(w int, h int) {
	vt.resize(w, h)
	vt.resizePty(w, h)
}

func (vt *Model) resizePty(w int, h int) {
	if vt.pty == nil {
		return
	}
	_ = pty.Setsize(vt.pty, &pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	})
}

func (vt *Model) resize(w int, h int) {
	oldWidth := vt.width()
	oldHeight := vt.height()
	defer func() {
		if oldWidth != w {
			vt.setDefaultTabStops()
		}
	}()
	if oldWidth != w || oldHeight != h {
		vt.setSynchronizedOutput(false)
		vt.clearPromptForRedraw()
	}

	primary := vt.primaryScreen
	alt := vt.altScreen
	viewportSourceRow := -1
	if vt.scrollOffset > 0 && !vt.mode.smcup {
		viewportSourceRow = primary.scrollbackLen() - vt.scrollOffset
	}
	if primary.width() == w {
		resized, primaryDelta, ok := primary.resizeHeight(h, vt.cursor.Style.Background)
		if !ok {
			goto reflowResize
		}
		vt.primaryScreen = resized
		cursorDelta := primaryDelta
		if alt.width() == w {
			resizedAlt, altDelta, ok := alt.resizeHeight(h, vt.cursor.Style.Background)
			if ok {
				vt.altScreen = resizedAlt
				if vt.mode.smcup {
					cursorDelta = altDelta
				}
				if vt.altState.saved {
					vt.altState.cursor.row += row(altDelta)
				}
			} else {
				vt.altScreen = newScreenBuffer(w, h, 0)
			}
		} else {
			vt.altScreen = newScreenBuffer(w, h, 0)
		}
		vt.resetMargins(w, h)
		vt.cursor.row += row(cursorDelta)
		if vt.primaryState.saved {
			vt.primaryState.cursor.row += row(primaryDelta)
		}
		if vt.cursor.col >= column(w) {
			vt.cursor.col = column(w) - 1
		}
		if vt.primaryState.saved && vt.primaryState.cursor.col >= column(w) {
			vt.primaryState.cursor.col = column(w) - 1
		}
		if vt.altState.saved && vt.altState.cursor.col >= column(w) {
			vt.altState.cursor.col = column(w) - 1
		}
		vt.clampScrollOffset()
		if vt.mode.smcup {
			vt.activeScreen = vt.altScreen
		} else {
			vt.activeScreen = vt.primaryScreen
		}
		vt.clampCursor()
		return
	}

reflowResize:
	vt.resetMargins(w, h)
	vt.lastCol = false
	if vt.primaryState.saved {
		vt.primaryState.lastCol = false
	}
	if vt.altState.saved {
		vt.altState.lastCol = false
	}
	if !vt.mode.decawm {
		vt.resizeNoReflow(w, h, primary, alt)
		return
	}
	if vt.mode.smcup {
		if resized, savedRow, savedCol, ok := primary.resizeReflowCursor(w, h, vt.cursor.Style.Background, vt.primaryState.cursor.row, vt.primaryState.cursor.col, vt.primaryState.saved); ok {
			vt.primaryScreen = resized
			if vt.primaryState.saved {
				vt.primaryState.cursor.row = savedRow
				vt.primaryState.cursor.col = savedCol
			}
		} else {
			vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
		}
		resizedAlt, newRow, newCol, ok := alt.resizeNoReflowCursor(w, h, vt.cursor.Style.Background, vt.cursor.row, vt.cursor.col, true)
		if ok {
			vt.altScreen = resizedAlt
			vt.cursor.row = newRow
			vt.cursor.col = newCol
		} else {
			vt.altScreen = newScreenBuffer(w, h, 0)
		}
		if _, savedRow, savedCol, ok := alt.resizeNoReflowCursor(w, h, vt.cursor.Style.Background, vt.altState.cursor.row, vt.altState.cursor.col, vt.altState.saved); ok && vt.altState.saved {
			vt.altState.cursor.row = savedRow
			vt.altState.cursor.col = savedCol
		}
		vt.activeScreen = vt.altScreen
	} else {
		resized, newRow, newCol, ok := primary.resizeReflowCursor(w, h, vt.cursor.Style.Background, vt.cursor.row, vt.cursor.col, true)
		if ok {
			vt.primaryScreen = resized
			vt.cursor.row = newRow
			vt.cursor.col = newCol
		} else {
			vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
		}
		if _, savedRow, savedCol, ok := primary.resizeReflowCursor(w, h, vt.cursor.Style.Background, vt.primaryState.cursor.row, vt.primaryState.cursor.col, vt.primaryState.saved); ok && vt.primaryState.saved {
			vt.primaryState.cursor.row = savedRow
			vt.primaryState.cursor.col = savedCol
		}
		if resizedAlt, ok := alt.resizeNoReflow(w, h, vt.cursor.Style.Background); ok {
			vt.altScreen = resizedAlt
		} else {
			vt.altScreen = newScreenBuffer(w, h, 0)
		}
		if _, savedRow, savedCol, ok := alt.resizeNoReflowCursor(w, h, vt.cursor.Style.Background, vt.altState.cursor.row, vt.altState.cursor.col, vt.altState.saved); ok && vt.altState.saved {
			vt.altState.cursor.row = savedRow
			vt.altState.cursor.col = savedCol
		}
		vt.activeScreen = vt.primaryScreen
	}
	vt.clampCursor()
	if vt.cursor.col >= column(w) {
		vt.cursor.col = column(w) - 1
	}
	vt.remapViewportAfterReflow(primary, viewportSourceRow, w)
	vt.clampScrollOffset()
}

func (vt *Model) resizeNoReflow(w int, h int, primary screenBuffer, alt screenBuffer) {
	if vt.mode.smcup {
		if resized, ok := primary.resizeNoReflow(w, h, vt.cursor.Style.Background); ok {
			vt.primaryScreen = resized
		} else {
			vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
		}
		resizedAlt, newRow, newCol, ok := alt.resizeNoReflowCursor(w, h, vt.cursor.Style.Background, vt.cursor.row, vt.cursor.col, true)
		if ok {
			vt.altScreen = resizedAlt
			vt.cursor.row = newRow
			vt.cursor.col = newCol
		} else {
			vt.altScreen = newScreenBuffer(w, h, 0)
		}
		vt.activeScreen = vt.altScreen
	} else {
		resized, newRow, newCol, ok := primary.resizeNoReflowCursor(w, h, vt.cursor.Style.Background, vt.cursor.row, vt.cursor.col, true)
		if ok {
			vt.primaryScreen = resized
			vt.cursor.row = newRow
			vt.cursor.col = newCol
		} else {
			vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
		}
		if resizedAlt, ok := alt.resizeNoReflow(w, h, vt.cursor.Style.Background); ok {
			vt.altScreen = resizedAlt
		} else {
			vt.altScreen = newScreenBuffer(w, h, 0)
		}
		vt.activeScreen = vt.primaryScreen
	}
	vt.clampCursor()
	vt.clampScrollOffset()
}

func (vt *Model) remapViewportAfterReflow(oldPrimary screenBuffer, viewportSourceRow int, width int) {
	if viewportSourceRow < 0 || vt.mode.smcup {
		return
	}
	reflowRow, _, ok := oldPrimary.reflowSourcePosition(width, viewportSourceRow, 0)
	if !ok {
		return
	}
	historyLen := vt.primaryScreen.scrollbackLen()
	if reflowRow >= historyLen {
		vt.scrollOffset = 0
		return
	}
	vt.scrollOffset = historyLen - reflowRow
}

func (vt *Model) resetMargins(w int, h int) {
	vt.margin.top = 0
	vt.margin.bottom = row(h) - 1
	vt.margin.left = 0
	vt.margin.right = column(w) - 1
}

func (vt *Model) width() int {
	return vt.activeScreen.width()
}

func (vt *Model) height() int {
	return vt.activeScreen.height()
}

func (vt *Model) resetWrap() {
	vt.resetPendingWrap()
	if vt.cursor.row < 0 || vt.cursor.row >= row(vt.height()) {
		return
	}
	r := vt.activeScreen.row(vt.cursor.row)
	if !r.wrapped {
		return
	}
	r.wrapped = false
	next := vt.cursor.row + 1
	if next < row(vt.height()) {
		vt.activeScreen.row(next).wrapContinuation = false
	}
}

func (vt *Model) resetPendingWrap() {
	pending := vt.lastCol
	vt.lastCol = false
	if pending && vt.cursor.col > vt.margin.right {
		vt.cursor.col = vt.margin.right
	}
}

func (vt *Model) clampCursor() {
	if vt.cursor.row < 0 {
		vt.cursor.row = 0
	}
	if vt.cursor.col < 0 {
		vt.cursor.col = 0
	}
	if vt.cursor.row >= row(vt.height()) {
		vt.cursor.row = row(vt.height()) - 1
	}
	if vt.cursor.col > vt.margin.right {
		vt.cursor.col = vt.margin.right
	}
}

// print sets the current cell contents to the given rune. The attributes will
// be copied from the current cursor attributes
func (vt *Model) print(seq ansi.Print) {
	if vt.status != statusDisplayMain {
		return
	}

	if utf8.RuneCountInString(seq.Grapheme) > 1 {
		seq.Grapheme, seq.Width = sanitizeVariationSelectors(seq.Grapheme, seq.Width)
		if seq.Grapheme == "" {
			return
		}
	}

	if !vt.mode.graphemeCluster && utf8.RuneCountInString(seq.Grapheme) > 1 {
		vt.printWithoutGraphemeClustering(seq.Grapheme)
		return
	}

	if !vt.mode.graphemeCluster {
		seq.Width = graphemeWidthWithoutVariationSelectors(seq.Grapheme, seq.Width)
	}

	w := seq.Width
	if w > 0 {
		vt.previousChar = vaxis.Character{
			Grapheme: seq.Grapheme,
			Width:    seq.Width,
		}
		vt.hasPreviousChar = true
	}

	if len(seq.Grapheme) == 1 {
		set := vt.charsets.designations[vt.charsets.selected]
		if shifted := applyCharsetGrapheme(set, seq.Grapheme[0]); shifted != "" {
			seq.Grapheme = shifted
		}
	} else {
		set := vt.charsets.designations[vt.charsets.selected]
		r, _ := utf8.DecodeRuneInString(seq.Grapheme)
		shifted := applyCharset(set, r)
		if shifted == ' ' {
			seq.Grapheme = " "
			seq.Width = 1
		}
	}

	// If we are single-shifted, move the previous charset into the current
	if vt.charsets.singleShift {
		vt.charsets.selected = vt.charsets.saved
		vt.charsets.singleShift = false
	}

	w = seq.Width
	if w == 0 {
		vt.appendZeroWidth(seq.Grapheme)
		return
	}

	rightLimit := vt.margin.right
	if vt.cursor.col > vt.margin.right {
		rightLimit = column(vt.width()) - 1
	}
	if vt.lastCol && vt.margin.right < column(vt.width())-1 && vt.cursor.col == vt.margin.right+1 {
		rightLimit = vt.margin.right
	}

	overflow := vt.cursor.col+column(w)-1 > rightLimit
	if !vt.mode.decawm && overflow {
		return
	}

	// handle wrapping
	var wrap bool
	// We printed in the last column last time
	if vt.lastCol {
		wrap = true
	}
	// We don't have room for this character so wrap
	if overflow {
		wrap = true
	}
	// We aren't in wrap mode, never wrap
	if !vt.mode.decawm {
		wrap = false
	}

	if wrap {
		vt.lastCol = false
		markSoftWrap := rightLimit == column(vt.width())-1
		if markSoftWrap {
			vt.activeScreen.row(vt.cursor.row).wrapped = true
		}
		if vt.cursor.row == vt.margin.bottom {
			vt.scrollUp(1)
		} else if vt.cursor.row < row(vt.height()-1) {
			vt.cursor.row += 1
		}
		vt.cursor.col = vt.margin.left
		if markSoftWrap {
			vt.activeScreen.row(vt.cursor.row).wrapContinuation = true
			vt.markSemanticContinuation()
		}
	}

	col := vt.cursor.col
	rw := vt.cursor.row

	if vt.mode.irm {
		vt.eraseWideAt(rw, col, vt.cursor.Style.Background, false)
		line := vt.activeScreen.line(rw)
		for i := rightLimit; i >= col+column(w); i -= 1 {
			line[i] = line[i-column(w)]
		}
		vt.eraseWideOverflow(rw, col, rightLimit, vt.cursor.Style.Background)
	}
	if col > column(vt.width())-1 {
		col = column(vt.width()) - 1
	}
	if rw > row(vt.height()-1) {
		rw = row(vt.height() - 1)
	}
	vt.eraseWideAt(rw, col, vt.cursor.Style.Background, false)

	cell := cell{
		Cell: vaxis.Cell{
			Character: vaxis.Character{
				Grapheme: seq.Grapheme,
				Width:    seq.Width,
			},
			Style: vt.cursor.Style,
		},
		protected:       vt.cursor.protected,
		semanticContent: vt.cursor.semanticContent,
	}

	vt.activeScreen.setCell(rw, col, cell)

	// Set trailing cells to a space if wide rune
	for i := column(1); i < column(w); i += 1 {
		if col+i > rightLimit {
			break
		}
		trailing := vt.activeScreen.cell(rw, col+i)
		trailing.Character.Grapheme = " "
		trailing.Style = vt.cursor.Style
		trailing.protected = vt.cursor.protected
		trailing.semanticContent = vt.cursor.semanticContent
	}

	switch {
	case !vt.mode.decawm && vt.cursor.col+column(w) > rightLimit:
	default:
		vt.cursor.col += column(w)
	}
	if vt.cursor.col >= rightLimit+1 && vt.mode.decawm {
		vt.lastCol = true
	}
}

func (vt *Model) printWithoutGraphemeClustering(grapheme string) {
	for len(grapheme) > 0 {
		r, size := utf8.DecodeRuneInString(grapheme)
		if r == utf8.RuneError && size == 0 {
			return
		}
		vt.print(ansi.Print{
			Grapheme: string(r),
			Width:    uucode.RuneWidth(r),
		})
		grapheme = grapheme[size:]
	}
}

func sanitizeVariationSelectors(grapheme string, width int) (string, int) {
	if !strings.ContainsRune(grapheme, '\uFE0E') && !strings.ContainsRune(grapheme, '\uFE0F') {
		return grapheme, width
	}

	var b strings.Builder
	b.Grow(len(grapheme))
	var last rune
	changed := false
	for _, r := range grapheme {
		if r == '\uFE0E' || r == '\uFE0F' {
			if !uucode.IsEmojiVariationBase(last) {
				changed = true
				continue
			}
		}
		b.WriteRune(r)
		last = r
	}
	if !changed {
		return grapheme, width
	}
	grapheme = b.String()
	if grapheme == "" {
		return "", 0
	}
	return grapheme, uucode.StringWidth(grapheme)
}

func graphemeWidthWithoutVariationSelectors(grapheme string, fallback int) int {
	if !strings.ContainsRune(grapheme, '\uFE0E') && !strings.ContainsRune(grapheme, '\uFE0F') {
		return fallback
	}

	var b strings.Builder
	b.Grow(len(grapheme))
	for _, r := range grapheme {
		switch r {
		case '\uFE0E', '\uFE0F':
			continue
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return fallback
	}
	return uucode.StringWidth(b.String())
}

func (vt *Model) appendZeroWidth(grapheme string) {
	col := vt.cursor.col - 1
	if vt.lastCol && vt.mode.decawm {
		col = vt.margin.right
	} else if !vt.mode.decawm && vt.cursor.col <= vt.margin.right {
		cell := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col)
		if vt.hasPreviousChar && cell.Character == vt.previousChar {
			col = vt.cursor.col
		}
	}
	if col < 0 || col > vt.margin.right {
		return
	}

	cell := vt.activeScreen.cell(vt.cursor.row, col)
	if cell.Character.Width == 0 && cell.Character.Grapheme == " " && col > 0 {
		col--
		cell = vt.activeScreen.cell(vt.cursor.row, col)
	}
	if cell.Character.Grapheme == "" {
		return
	}
	oldGrapheme := cell.Character.Grapheme
	oldWidth := cell.Character.Width
	if isVariationSelector(grapheme) {
		base, _ := utf8.DecodeRuneInString(oldGrapheme)
		if !uucode.IsEmojiVariationBase(base) {
			return
		}
	}
	cell.Character.Grapheme += grapheme
	if !vt.mode.graphemeCluster {
		return
	}
	newWidth := uucode.StringWidth(cell.Character.Grapheme)
	switch {
	case oldWidth > 1 && newWidth == 1:
		cell.Character.Width = 1
		for i := column(1); i < column(oldWidth) && col+i < column(vt.width()); i += 1 {
			vt.activeScreen.eraseCell(vt.cursor.row, col+i, cell.Style.Background)
		}
		if vt.lastCol {
			vt.resetPendingWrap()
		}
	case oldWidth == 1 && newWidth > 1:
		if col+column(newWidth)-1 > vt.margin.right {
			if vt.lastCol && vt.mode.decawm {
				markSoftWrap := vt.margin.right == column(vt.width())-1
				if markSoftWrap {
					vt.activeScreen.row(vt.cursor.row).wrapped = true
				}
				wrappedCell := *cell
				wrappedCell.Character.Width = newWidth
				cell.Character.Grapheme = " "
				cell.Character.Width = 0
				if vt.cursor.row == vt.margin.bottom {
					vt.scrollUp(1)
				} else if vt.cursor.row < row(vt.height()-1) {
					vt.cursor.row += 1
				}
				vt.cursor.col = vt.margin.left
				vt.lastCol = false
				if markSoftWrap {
					vt.activeScreen.row(vt.cursor.row).wrapContinuation = true
					vt.markSemanticContinuation()
				}
				vt.activeScreen.setCell(vt.cursor.row, vt.cursor.col, wrappedCell)
				for i := column(1); i < column(newWidth) && vt.cursor.col+i <= vt.margin.right; i += 1 {
					tail := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col+i)
					tail.Character.Grapheme = " "
					tail.Character.Width = 0
					tail.Style = wrappedCell.Style
					tail.protected = wrappedCell.protected
					tail.semanticContent = wrappedCell.semanticContent
					tail.Hyperlink = wrappedCell.Hyperlink
					tail.HyperlinkParams = wrappedCell.HyperlinkParams
				}
				vt.cursor.col += column(newWidth)
				if vt.cursor.col >= vt.margin.right+1 {
					vt.lastCol = true
				}
				return
			}
			cell.Character.Grapheme = oldGrapheme
			cell.Character.Width = oldWidth
			return
		}
		cell.Character.Width = newWidth
		for i := column(1); i < column(newWidth); i += 1 {
			tail := vt.activeScreen.cell(vt.cursor.row, col+i)
			tail.Character.Grapheme = " "
			tail.Character.Width = 0
			tail.Style = cell.Style
			tail.protected = cell.protected
			tail.semanticContent = cell.semanticContent
			tail.Hyperlink = cell.Hyperlink
			tail.HyperlinkParams = cell.HyperlinkParams
		}
		if col+column(newWidth)-1 >= vt.margin.right && vt.mode.decawm {
			vt.cursor.col = vt.margin.right + 1
			vt.lastCol = true
		}
	}
}

func isVariationSelector(grapheme string) bool {
	return grapheme == "\uFE0E" || grapheme == "\uFE0F"
}

// scrollUp shifts all text upward by n rows. Semantically, this is backwards -
// usually scroll up would mean you shift rows down
func (vt *Model) scrollUp(n int) {
	captured := vt.activeScreen.scrollUp(
		vt.margin.top,
		vt.margin.bottom,
		vt.margin.left,
		vt.margin.right,
		n,
		vt.cursor.Style.Background,
	)
	if captured > 0 && vt.scrollOffset > 0 {
		vt.scrollOffset += captured
		vt.clampScrollOffset()
	}
}

// scrollDown shifts all lines down by n rows.
func (vt *Model) scrollDown(n int) {
	vt.activeScreen.scrollDown(
		vt.margin.top,
		vt.margin.bottom,
		vt.margin.left,
		vt.margin.right,
		n,
		vt.cursor.Style.Background,
	)
}

func (vt *Model) Close() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.stopReplyWorker()
	if vt.cmd != nil && vt.cmd.Process != nil {
		vt.cmd.Process.Kill()
	}
	if vt.syncTimer != nil {
		vt.syncTimer.Stop()
		vt.syncTimer = nil
	}
	vt.pty.Close()
}

func (vt *Model) Draw(win vaxis.Window) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.dirty = false
	width, height := win.Size()
	if int(width) != vt.width() || int(height) != vt.height() {
		win.Width = width
		win.Height = height
		vt.Resize(width, height)
	}
	for r := 0; r < vt.height(); r += 1 {
		line := vt.visibleLine(r)
		for col := 0; col < vt.width(); {
			cell := line[col]
			w := cell.Width

			if cell.Grapheme == "" {
				cell.Grapheme = " "
			}

			win.SetCell(col, r, vt.renderCell(cell.Cell))
			if w == 0 {
				w = 1
			}
			col += w
		}
	}
	if cursorCol, cursorRow, ok := vt.cursorViewportPosition(); ok {
		win.ShowCursor(cursorCol, cursorRow, vt.effectiveCursorStyle())
	} else if win.Vx != nil {
		win.Vx.HideCursor()
	}
	vx := win.Vx
	vt.vx = vx
outer:
	for _, img := range vt.graphics {
		for _, imgVx := range img.vaxii {
			if vx != imgVx.vx {
				continue
			}
			// We have already created an image for this
			// Vaxis. All we have to do is draw it
			win := win.New(img.origin.col, img.origin.row, -1, -1)
			imgVx.vxImage.Draw(win)
			continue outer
		}
		// We haven't encountered this vaxis before
		vxImg, err := vx.NewImage(img.img)
		if err != nil {
			log.Error("couldn't create Vaxis image: %v", err)
			continue
		}
		// We "resize" the image to the full window size. This will
		// trigger the encoding
		vxImg.Resize(win.Size())
		img.vaxii = append(img.vaxii, &vaxisImage{
			vx:      vx,
			vxImage: vxImg,
		})
	}
}

func (vt *Model) cursorViewportPosition() (int, int, bool) {
	if vt.mode.synchronizedOutput || !vt.mode.dectcem || !atomicLoad(&vt.focused) {
		return 0, 0, false
	}
	historyLen := vt.activeScreen.scrollbackLen()
	topSourceRow := historyLen
	if vt.scrollOffset > 0 && historyLen > 0 {
		topSourceRow = historyLen - vt.scrollOffset
	}
	cursorSourceRow := historyLen + int(vt.cursor.row)
	viewportRow := cursorSourceRow - topSourceRow
	if viewportRow < 0 || viewportRow >= vt.height() {
		return 0, 0, false
	}
	return int(vt.cursor.col), viewportRow, true
}

func (vt *Model) renderCell(cell vaxis.Cell) vaxis.Cell {
	if vt.mode.decscnm {
		cell.Attribute ^= vaxis.AttrReverse
	}
	return cell
}

func (vt *Model) Focus() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	atomicStore(&vt.focused, true)
	if vt.mode.focusEvents {
		vt.reportFocus()
	}
}

func (vt *Model) Blur() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	atomicStore(&vt.focused, false)
	if vt.mode.focusEvents {
		vt.reportFocus()
	}
}

func (vt *Model) reportFocus() {
	if atomicLoad(&vt.focused) {
		vt.writePtyString("\x1b[I")
		return
	}
	vt.writePtyString("\x1b[O")
}

// func (vt *VT) HandleEvent(e tcell.Event) bool {
// 	vt.mu.Lock()
// 	defer vt.mu.Unlock()
// 	switch e := e.(type) {
// 	case *tcell.EventKey:
// 		vt.pty.WriteString(keyCode(e))
// 		return true
// 	case *tcell.EventPaste:
// 		switch {
// 		case vt.mode&paste == 0:
// 			return false
// 		case e.Start():
// 			vt.pty.WriteString(info.PasteStart)
// 			return true
// 		case e.End():
// 			vt.pty.WriteString(info.PasteEnd)
// 			return true
// 		}
// 	case *tcell.EventMouse:
// 		str := vt.handleMouse(e)
// 		vt.pty.WriteString(str)
// 	}
// 	return false
// }

func atomicLoad(val *int32) bool {
	return atomic.LoadInt32(val) == 1
}

func atomicStore(addr *int32, val bool) {
	if val {
		atomic.StoreInt32(addr, 1)
	} else {
		atomic.StoreInt32(addr, 0)
	}
}
