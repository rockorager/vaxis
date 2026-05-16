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
	selection           *selectionRange
	selectionMouse      mouseSelectionState

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
	var pendingResize ptyResize
	var pendingWrites []ptyWrite
	vt.mu.Lock()
	defer func() {
		vt.mu.Unlock()
		pendingResize.apply()
		for _, write := range pendingWrites {
			write.apply()
		}
	}()
	if resize, ok := msg.(vaxis.Resize); ok {
		var pendingWrite ptyWrite
		pendingResize, pendingWrite = vt.updateResize(resize)
		pendingWrites = append(pendingWrites, pendingWrite)
		return
	}
	vt.invalidate()
	switch msg := msg.(type) {
	case vaxis.Key:
		if vt.handleViewportKey(msg) {
			return
		}
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		vt.clearSelectionLocked()
		str := vt.encodeKey(msg)
		str = vt.linefeedModeInput(str)
		if str != "" {
			vt.scrollViewportBottom()
		}
		pendingWrites = append(pendingWrites, vt.pendingPtyWrite(str))
	case vaxis.PasteStartEvent:
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		vt.clearSelectionLocked()
		if vt.mode.paste {
			vt.scrollViewportBottom()
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite("\x1B[200~"))
			return
		}
	case vaxis.PasteEndEvent:
		if vt.keyboardActionModeBlocksInput() {
			return
		}
		vt.clearSelectionLocked()
		if vt.mode.paste {
			vt.scrollViewportBottom()
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite("\x1B[201~"))
			return
		}
	case vaxis.Mouse:
		if vt.handleSelectionMouse(msg) {
			return
		}
		if promptClick := vt.handlePromptClick(msg); promptClick != "" {
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite(promptClick))
			return
		}
		if vt.handleViewportMouse(msg) {
			return
		}
		mouse := vt.handleMouse(msg)
		pendingWrites = append(pendingWrites, vt.pendingPtyWrite(mouse))
		return
	case vaxis.ColorThemeUpdate:
		vt.theme = msg.Mode
		if vt.mode.colorScheme {
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite(colorSchemeReport(msg.Mode)))
			return
		}
	case vaxis.FocusIn:
		atomicStore(&vt.focused, true)
		if vt.mode.focusEvents {
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite(vt.focusReport()))
		}
	case vaxis.FocusOut:
		atomicStore(&vt.focused, false)
		if vt.mode.focusEvents {
			pendingWrites = append(pendingWrites, vt.pendingPtyWrite(vt.focusReport()))
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

type ptyWrite struct {
	pty  *os.File
	data string
}

func (write ptyWrite) apply() {
	if write.pty == nil || write.data == "" {
		return
	}
	_, _ = write.pty.WriteString(write.data)
}

func (vt *Model) pendingPtyWrite(s string) ptyWrite {
	if vt.pty == nil || s == "" {
		return ptyWrite{}
	}
	return ptyWrite{pty: vt.pty, data: s}
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

func (vt *Model) scrollViewportBottom() bool {
	before := vt.scrollOffset
	vt.scrollOffset = 0
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
	line, _, ok := vt.visibleSourceLine(r)
	if !ok {
		return nil
	}
	return line
}

func (vt *Model) visibleSourceLine(r int) ([]cell, int, bool) {
	source, ok := vt.viewportSourceRow(r)
	if !ok {
		return nil, 0, false
	}
	line, _, ok := vt.sourceRowLine(source)
	return line, source, ok
}

func (vt *Model) viewportSourceRow(r int) (int, bool) {
	if r < 0 || r >= vt.height() {
		return 0, false
	}
	historyLen := vt.activeScreen.scrollbackLen()
	if vt.scrollOffset == 0 || historyLen == 0 {
		return historyLen + r, true
	}
	return historyLen - vt.scrollOffset + r, true
}

func (vt *Model) sourceRowLine(source int) ([]cell, screenRow, bool) {
	historyLen := vt.activeScreen.scrollbackLen()
	if source < historyLen {
		line, ok := vt.activeScreen.scrollbackLine(source)
		if ok {
			return line.cells, line.row, true
		}
		return nil, screenRow{}, false
	}
	activeRow := source - historyLen
	if activeRow < 0 || activeRow >= vt.height() {
		return nil, screenRow{}, false
	}
	return vt.activeScreen.line(row(activeRow)), *vt.activeScreen.row(row(activeRow)), true
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
	vt.mu.Lock()
	pendingResize, pendingWrite := vt.updateResize(vaxis.Resize{Cols: w, Rows: h})
	vt.mu.Unlock()
	pendingResize.apply()
	pendingWrite.apply()
}

func (vt *Model) updateResize(msg vaxis.Resize) (ptyResize, ptyWrite) {
	msg = vt.resizeWithKnownPixels(msg)
	if vt.size == msg {
		return ptyResize{}, ptyWrite{}
	}
	vt.invalidate()
	vt.setSynchronizedOutput(false)
	vt.size = msg
	if vt.width() == msg.Cols && vt.height() == msg.Rows {
		return ptyResize{}, vt.pendingInBandSizeReport()
	}
	vt.resize(msg.Cols, msg.Rows)
	return vt.pendingPtyResize(msg), vt.pendingInBandSizeReport()
}

func (vt *Model) resizeWithKnownPixels(size vaxis.Resize) vaxis.Resize {
	if size.Cols <= 0 || size.Rows <= 0 {
		return size
	}
	if size.XPixel == 0 && vt.size.XPixel > 0 && vt.size.Cols > 0 {
		size.XPixel = size.Cols * vt.size.XPixel / vt.size.Cols
	}
	if size.YPixel == 0 && vt.size.YPixel > 0 && vt.size.Rows > 0 {
		size.YPixel = size.Rows * vt.size.YPixel / vt.size.Rows
	}
	return size
}

func (vt *Model) pendingInBandSizeReport() ptyWrite {
	if !vt.mode.inBandSizeReports {
		return ptyWrite{}
	}
	return vt.pendingPtyWrite(vt.inBandSizeReportString())
}

type ptyResize struct {
	pty  *os.File
	size vaxis.Resize
}

func (resize ptyResize) apply() {
	if resize.pty == nil {
		return
	}
	_ = pty.Setsize(resize.pty, &pty.Winsize{
		Cols: uint16(resize.size.Cols),
		Rows: uint16(resize.size.Rows),
		X:    uint16(resize.size.XPixel),
		Y:    uint16(resize.size.YPixel),
	})
}

func (vt *Model) pendingPtyResize(size vaxis.Resize) ptyResize {
	if vt.pty == nil {
		return ptyResize{}
	}
	return ptyResize{pty: vt.pty, size: size}
}

type drawSnapshot struct {
	cells         []positionedCell
	cursorCol     int
	cursorRow     int
	cursorStyle   vaxis.CursorStyle
	cursorVisible bool
	allGraphics   []*Image
	graphics      []positionedImage
	vx            *vaxis.Vaxis
}

type positionedCell struct {
	col  int
	row  int
	cell vaxis.Cell
}

func (vt *Model) resize(w int, h int) {
	oldWidth := vt.width()
	oldHeight := vt.height()
	if vt.selection != nil && (oldWidth != w || oldHeight != h) {
		vt.clearSelectionLocked()
	}
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
	oldSourceRows := primary.scrollbackLen() + primary.height()
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
		droppedSourceRows := oldSourceRows - (vt.primaryScreen.scrollbackLen() + vt.primaryScreen.height())
		vt.reflowGraphics(primary, oldWidth, max(0, droppedSourceRows))
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
		if oldWidth != w {
			vt.clearGraphicsLocked()
		}
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
	vt.reflowGraphics(primary, oldWidth, primary.reflowDroppedSourceRows(w, h))
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

func (vt *Model) reflowGraphics(oldPrimary screenBuffer, oldWidth int, droppedSourceRows int) {
	if vt.mode.smcup || len(vt.graphics) == 0 {
		return
	}
	if oldWidth <= 0 || oldWidth == vt.width() {
		vt.shiftGraphicsSourceRows(droppedSourceRows)
		vt.validateGraphics()
		return
	}
	kept := vt.graphics[:0]
	for _, img := range vt.graphics {
		sourceRow, col, ok := oldPrimary.reflowSourcePosition(vt.width(), img.sourceRow, img.origin.col)
		if !ok {
			img.destroy()
			continue
		}
		img.sourceRow = sourceRow - droppedSourceRows
		img.origin.col = col
		if !vt.graphicFits(img) {
			img.destroy()
			continue
		}
		kept = append(kept, img)
	}
	vt.graphics = kept
}

func (vt *Model) validateGraphics() {
	kept := vt.graphics[:0]
	for _, img := range vt.graphics {
		if !vt.graphicFits(img) {
			img.destroy()
			continue
		}
		kept = append(kept, img)
	}
	vt.graphics = kept
}

func (vt *Model) graphicFits(img *Image) bool {
	if img.sourceRow < 0 || img.sourceRow+img.rows > vt.activeScreen.scrollbackLen()+vt.height() {
		return false
	}
	if img.origin.col < 0 || img.origin.col+img.cols > vt.width() {
		return false
	}
	return true
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
	historyLen := vt.activeScreen.scrollbackLen()
	captured := vt.activeScreen.scrollUp(
		vt.margin.top,
		vt.margin.bottom,
		vt.margin.left,
		vt.margin.right,
		n,
		vt.cursor.Style.Background,
	)
	if captured > 0 {
		vt.shiftGraphicsSourceRows(historyLen + captured - vt.activeScreen.scrollbackLen())
	}
	if captured > 0 && vt.scrollOffset > 0 {
		vt.scrollOffset += captured
		vt.clampScrollOffset()
	}
	if captured > 0 {
		vt.clearSelectionLocked()
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

func (vt *Model) shiftGraphicsSourceRows(dropped int) {
	if dropped <= 0 || len(vt.graphics) == 0 {
		return
	}
	kept := vt.graphics[:0]
	for _, img := range vt.graphics {
		img.sourceRow -= dropped
		if img.sourceRow+img.rows <= 0 {
			img.destroy()
			continue
		}
		kept = append(kept, img)
	}
	vt.graphics = kept
}

func (vt *Model) clearGraphicsLocked() {
	for _, img := range vt.graphics {
		for _, cached := range img.vaxii {
			cached.vx.RemoveImage(cached.vxImage)
		}
		img.destroy()
	}
	vt.graphics = nil
}

func (vt *Model) Close() {
	vt.mu.Lock()
	vt.stopReplyWorker()
	cmd := vt.cmd
	ptyFile := vt.pty
	vt.pty = nil
	if vt.cmd != nil && vt.cmd.Process != nil {
		cmd = vt.cmd
	}
	if vt.syncTimer != nil {
		vt.syncTimer.Stop()
		vt.syncTimer = nil
	}
	vt.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if ptyFile != nil {
		_ = ptyFile.Close()
	}
}

func (vt *Model) Draw(win vaxis.Window) {
	var snapshot drawSnapshot
	vt.mu.Lock()
	vt.dirty = false
	snapshot = vt.snapshotDraw(win.Vx)
	vt.mu.Unlock()

	for _, cell := range snapshot.cells {
		win.SetCell(cell.col, cell.row, cell.cell)
	}
	if snapshot.cursorVisible {
		win.ShowCursor(snapshot.cursorCol, snapshot.cursorRow, snapshot.cursorStyle)
	}
	vt.removeGraphicsPlacements(snapshot.vx, snapshot.allGraphics)
	vt.drawGraphics(win, snapshot.vx, snapshot.graphics)
}

func (vt *Model) snapshotDraw(vx *vaxis.Vaxis) drawSnapshot {
	snapshot := drawSnapshot{
		cells:       make([]positionedCell, 0, vt.width()*vt.height()),
		allGraphics: append([]*Image(nil), vt.graphics...),
		graphics:    vt.visibleGraphics(),
		vx:          vx,
	}
	vt.vx = vx
	for r := 0; r < vt.height(); r += 1 {
		line, sourceRow, ok := vt.visibleSourceLine(r)
		if !ok {
			continue
		}
		for col := 0; col < vt.width(); {
			cell := line[col]
			w := cell.Width

			if cell.Grapheme == "" {
				cell.Grapheme = " "
			}

			rendered := cell.Cell
			if vt.selectionContains(sourceRow, col) {
				rendered.Attribute ^= vaxis.AttrReverse
			}
			snapshot.cells = append(snapshot.cells, positionedCell{
				col:  col,
				row:  r,
				cell: vt.renderCell(rendered),
			})
			if w == 0 {
				w = 1
			}
			col += w
		}
	}
	if cursorCol, cursorRow, ok := vt.cursorViewportPosition(); ok {
		snapshot.cursorCol = cursorCol
		snapshot.cursorRow = cursorRow
		snapshot.cursorStyle = vt.effectiveCursorStyle()
		snapshot.cursorVisible = true
	}
	return snapshot
}

func (vt *Model) removeGraphicsPlacements(vx *vaxis.Vaxis, graphics []*Image) {
	if vx == nil {
		return
	}
	for _, img := range graphics {
		for _, cached := range img.vaxii {
			if cached.vx == vx {
				vx.RemoveImage(cached.vxImage)
			}
		}
	}
}

func (vt *Model) visibleGraphics() []positionedImage {
	if len(vt.graphics) == 0 {
		return nil
	}
	topSourceRow := vt.topViewportSourceRow()
	visible := make([]positionedImage, 0, len(vt.graphics))
	for _, img := range vt.graphics {
		viewportRow := img.sourceRow - topSourceRow
		if viewportRow < 0 || viewportRow+img.rows > vt.height() {
			continue
		}
		if img.origin.col < 0 || img.origin.col+img.cols > vt.width() {
			continue
		}
		visible = append(visible, positionedImage{
			img: img,
			row: viewportRow,
			col: img.origin.col,
		})
	}
	return visible
}

func (vt *Model) topViewportSourceRow() int {
	historyLen := vt.activeScreen.scrollbackLen()
	if vt.scrollOffset > 0 && historyLen > 0 {
		return historyLen - vt.scrollOffset
	}
	return historyLen
}

func (vt *Model) drawGraphics(win vaxis.Window, vx *vaxis.Vaxis, graphics []positionedImage) {
	if vx == nil {
		return
	}
outer:
	for _, graphic := range graphics {
		img := graphic.img
		if vxImg := vt.cachedVaxisImage(img, vx); vxImg != nil {
			win := win.New(graphic.col, graphic.row, -1, -1)
			vxImg.Draw(win)
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
		if cached, ok := vt.cacheVaxisImage(img, vx, vxImg); ok {
			win := win.New(graphic.col, graphic.row, -1, -1)
			cached.Draw(win)
		}
	}
}

func (vt *Model) cachedVaxisImage(img *Image, vx *vaxis.Vaxis) vaxis.Image {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	for _, imgVx := range img.vaxii {
		if vx == imgVx.vx {
			return imgVx.vxImage
		}
	}
	return nil
}

func (vt *Model) cacheVaxisImage(img *Image, vx *vaxis.Vaxis, vxImg vaxis.Image) (vaxis.Image, bool) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	for _, imgVx := range img.vaxii {
		if vx == imgVx.vx {
			return imgVx.vxImage, true
		}
	}
	img.vaxii = append(img.vaxii, &vaxisImage{
		vx:      vx,
		vxImage: vxImg,
	})
	return nil, false
}

func (vt *Model) cursorViewportPosition() (int, int, bool) {
	if vt.mode.synchronizedOutput || !vt.mode.dectcem || !atomicLoad(&vt.focused) {
		return 0, 0, false
	}
	historyLen := vt.activeScreen.scrollbackLen()
	topSourceRow := vt.topViewportSourceRow()
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
	var pendingWrite ptyWrite
	vt.mu.Lock()
	atomicStore(&vt.focused, true)
	if vt.mode.focusEvents {
		pendingWrite = vt.pendingPtyWrite(vt.focusReport())
	}
	vt.mu.Unlock()
	pendingWrite.apply()
}

func (vt *Model) Blur() {
	var pendingWrite ptyWrite
	vt.mu.Lock()
	atomicStore(&vt.focused, false)
	if vt.mode.focusEvents {
		pendingWrite = vt.pendingPtyWrite(vt.focusReport())
	}
	vt.mu.Unlock()
	pendingWrite.apply()
}

func (vt *Model) focusReport() string {
	if atomicLoad(&vt.focused) {
		return "\x1b[I"
	}
	return "\x1b[O"
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
