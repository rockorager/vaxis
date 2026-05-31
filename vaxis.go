// Package vaxis is a terminal user interface for modern terminals
package vaxis

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"go.rockorager.dev/vaxis/ansi"
	"go.rockorager.dev/vaxis/log"
)

type capabilities struct {
	synchronizedUpdate bool
	unicodeCore        bool
	noZWJ              bool // a terminal may support shaped emoji but not ZWJ
	rgb                bool
	kittyGraphics      bool
	kittyKeyboard      bool
	styledUnderlines   bool
	sixels             bool
	colorThemeUpdates  bool
	reportSizeChars    bool
	reportSizePixels   bool
	osc4               bool
	osc10              bool
	osc11              bool
	osc176             bool
	inBandResize       bool
	explicitWidth      bool
	sgrPixels          bool
}

type cursorState struct {
	row     int
	col     int
	style   CursorStyle
	visible bool
}

type sizeReport struct {
	size   Resize
	chars  bool
	pixels bool
}

// Console provides a terminal-like device for Vaxis to read from and draw to.
type Console interface {
	io.Reader
	io.Writer

	Fd() uintptr
	SetRaw() error
	Reset() error
	Size() (cols int, rows int, xPixels int, yPixels int, err error)
	Close() error
}

// Options are the runtime options which must be supplied to a new [Vaxis]
// object at instantiation
type Options struct {
	// DisableKittyKeyboard disables the use of the Kitty Keyboard protocol.
	// By default, if support is detected the protocol will be used.
	DisableKittyKeyboard bool
	// Deprecated Use CSIuBitMask instead
	//
	// ReportKeyboardEvents will report key release and key repeat events if
	// KittyKeyboardProtocol is enabled and supported by the terminal
	ReportKeyboardEvents bool
	// The size of the event queue channel. This will default to 1024 to
	// prevent any blocking on writes.
	EventQueueSize int
	// Disable mouse events
	DisableMouse bool
	// WithTTY passes an absolute path to use for the TTY Vaxis will draw
	// on. If the file is not a TTY, an error will be returned when calling
	// New
	WithTTY string
	// NoSignals causes Vaxis to not install any signal handlers
	NoSignals bool

	// Deprecated: Vaxis now enables all supported Kitty keyboard flags.
	CSIuBitMask CSIuBitMask

	// WithConsole provides the ability to use a custom console.
	WithConsole Console

	// EnableSGRPixels provides pixel level precision of mouse movement. This has
	// no effect if DisableMouse is true
	EnableSGRPixels bool

	// PrimaryScreen enables primary-screen rendering. When set, Vaxis does not
	// enter the alternate screen. Window returns a live region surface, and
	// Append queues output to be written immediately before that region on the
	// next Render.
	PrimaryScreen *PrimaryScreenOptions
}

// PrimaryScreenOptions configures primary-screen rendering.
type PrimaryScreenOptions struct {
	// RegionHeight is the height of the live region.
	RegionHeight int
}

type CSIuBitMask int

const (
	CSIuDisambiguate CSIuBitMask = 1 << iota
	CSIuReportEvents
	CSIuAlternateKeys
	CSIuAllKeys
	CSIuAssociatedText
)

const kittyKeyboardAllFlags = int(CSIuDisambiguate | CSIuReportEvents | CSIuAlternateKeys | CSIuAllKeys | CSIuAssociatedText)

type Vaxis struct {
	queue            chan Event
	tty              tty
	parser           *ansi.Parser
	tw               *writer
	screenNext       *screen
	screenLast       *screen
	primaryScreen    *primaryScreen
	graphicsNext     []*placement
	graphicsLast     []*placement
	mouseShapeNext   MouseShape
	mouseShapeLast   MouseShape
	appIDLast        appID
	pastePending     bool
	chClipboard      chan string
	chSigWinSz       chan os.Signal
	chSigKill        chan os.Signal
	chCursorPos      chan [2]int
	chQuit           chan bool
	winSize          Resize
	chSizeReport     chan sizeReport
	ready            bool
	caps             capabilities
	graphicsProtocol int
	graphicsIDNext   uint64
	reqCursorPos     bool
	charCache        map[string]int
	cursorNext       cursorState
	cursorLast       cursorState
	closed           bool
	refresh          bool
	kittyFlags       int
	disableMouse     bool
	chFg             chan string
	chFgMu           sync.Mutex
	chBg             chan string
	chBgMu           sync.Mutex
	chColor          chan string
	chColorMu        sync.Mutex
	userCursorStyle  CursorStyle

	xtwinops bool

	withTty     string
	withConsole Console

	termID terminalID

	renders int
	elapsed time.Duration

	mu sync.Mutex

	noSignals bool
}

type primaryScreen struct {
	regionHeight int
	append       []string
	rendered     bool
	resized      bool
	visualRows   int
}

// New creates a new [Vaxis] instance. Calling New will query the underlying
// terminal for supported features and enter the alternate screen unless
// primary-screen rendering is enabled.
func New(opts Options) (*Vaxis, error) {
	switch os.Getenv("VAXIS_LOG_LEVEL") {
	case "trace":
		log.SetLevel(log.LevelTrace)
		log.SetOutput(os.Stderr)
	case "debug":
		log.SetLevel(log.LevelDebug)
		log.SetOutput(os.Stderr)
	case "info":
		log.SetLevel(log.LevelInfo)
		log.SetOutput(os.Stderr)
	case "warn":
		log.SetLevel(log.LevelWarn)
		log.SetOutput(os.Stderr)
	case "error":
		log.SetLevel(log.LevelError)
		log.SetOutput(os.Stderr)
	}

	// Let's give some deadline for our queries responding. If they don't,
	// it means the terminal doesn't respond to Primary Device Attributes
	// and that is a problem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	vx := &Vaxis{
		kittyFlags: kittyKeyboardFlags(opts),
	}
	if opts.PrimaryScreen != nil {
		if opts.PrimaryScreen.RegionHeight <= 0 {
			return nil, fmt.Errorf("primary screen region height must be positive")
		}
		vx.primaryScreen = &primaryScreen{regionHeight: opts.PrimaryScreen.RegionHeight}
	}

	if opts.EventQueueSize < 1 {
		opts.EventQueueSize = 1024
	}

	if opts.DisableMouse {
		vx.disableMouse = true
	}

	vx.noSignals = opts.NoSignals

	switch {
	case opts.WithConsole != nil:
		vx.withConsole = opts.WithConsole
	case opts.WithTTY != "":
		vx.withTty = opts.WithTTY
	}

	vx.queue = make(chan Event, opts.EventQueueSize)
	vx.screenNext = newScreen()
	vx.screenLast = newScreen()
	vx.chClipboard = make(chan string)
	vx.chSigWinSz = make(chan os.Signal, 1)
	vx.chSigKill = make(chan os.Signal, 1)
	vx.chCursorPos = make(chan [2]int)
	vx.chQuit = make(chan bool)
	vx.chSizeReport = make(chan sizeReport, 8)
	vx.charCache = make(map[string]int, 256)
	vx.chFg = make(chan string, 1)
	vx.chBg = make(chan string, 1)
	vx.chColor = make(chan string, 1)

	err = vx.openTty()
	if err != nil {
		return nil, err
	}

	vx.sendQueries()
outer:
	for {
		select {
		case <-ctx.Done():
			log.Warn("terminal did not respond to DA1 query")
			break outer
		case ev := <-vx.queue:
			switch ev := ev.(type) {
			case primaryDeviceAttribute:
				break outer
			case capabilitySixel:
				log.Info("[capability] Sixel graphics")
				vx.mu.Lock()
				vx.caps.sixels = true
				if vx.graphicsProtocol < sixelGraphics {
					vx.graphicsProtocol = sixelGraphics
				}
				vx.mu.Unlock()
			case capabilityOsc4:
				log.Info("[capability] OSC 4 supported")
				vx.mu.Lock()
				vx.caps.osc4 = true
				vx.mu.Unlock()
			case capabilityOsc10:
				log.Info("[capability] OSC 10 supported")
				vx.mu.Lock()
				vx.caps.osc10 = true
				vx.mu.Unlock()
			case capabilityOsc11:
				log.Info("[capability] OSC 11 supported")
				vx.mu.Lock()
				vx.caps.osc11 = true
				vx.mu.Unlock()
			case synchronizedUpdates:
				log.Info("[capability] Synchronized updates")
				vx.mu.Lock()
				vx.caps.synchronizedUpdate = true
				vx.mu.Unlock()
			case unicodeCoreCap:
				log.Info("[capability] Unicode core")
				vx.mu.Lock()
				vx.caps.unicodeCore = true
				vx.mu.Unlock()
			case notifyColorChange:
				log.Info("[capability] Color theme notifications")
				vx.mu.Lock()
				vx.caps.colorThemeUpdates = true
				vx.mu.Unlock()
			case kittyKeyboard:
				log.Info("[capability] Kitty keyboard")
				if opts.DisableKittyKeyboard {
					continue
				}
				vx.mu.Lock()
				vx.caps.kittyKeyboard = true
				vx.mu.Unlock()
			case styledUnderlines:
				log.Info("[capability] Styled underlines")
				vx.mu.Lock()
				vx.caps.styledUnderlines = true
				vx.mu.Unlock()
			case truecolor:
				log.Info("[capability] RGB")
				vx.mu.Lock()
				vx.caps.rgb = true
				vx.mu.Unlock()
			case kittyGraphics:
				log.Info("[capability] Kitty graphics supported")
				vx.mu.Lock()
				vx.caps.kittyGraphics = true
				if vx.graphicsProtocol < kitty {
					vx.graphicsProtocol = kitty
				}
				vx.mu.Unlock()
			case textAreaPix:
				log.Info("[capability] Report screen size: pixels")
				vx.mu.Lock()
				vx.caps.reportSizePixels = true
				vx.mu.Unlock()
			case textAreaChar:
				log.Info("[capability] Report screen size: characters")
				vx.mu.Lock()
				vx.caps.reportSizeChars = true
				vx.mu.Unlock()
			case appID:
				log.Info("[capability] OSC 176 supported")
				vx.mu.Lock()
				vx.caps.osc176 = true
				vx.appIDLast = ev
				vx.mu.Unlock()
			case terminalID:
				vx.mu.Lock()
				vx.termID = ev
				vx.mu.Unlock()
			case capabilitySgrPixels:
				log.Info("[capability] SGR Pixels supported")
				if !opts.EnableSGRPixels {
					continue
				}
				vx.mu.Lock()
				vx.caps.sgrPixels = true
				vx.mu.Unlock()
			}
		}
	}

	if vx.primaryScreen == nil {
		vx.enterAltScreen()
	}
	vx.enableModes()
	if !vx.noSignals {
		vx.setupSignals()
	}
	vx.applyQuirks()

	switch os.Getenv("VAXIS_GRAPHICS") {
	case "none":
		vx.graphicsProtocol = noGraphics
	case "full":
		vx.graphicsProtocol = fullBlock
	case "half":
		vx.graphicsProtocol = halfBlock
	case "sixel":
		vx.graphicsProtocol = sixelGraphics
	case "kitty":
		vx.graphicsProtocol = kitty
	default:
		// Use highest quality block renderer by default. Users will
		// need to fallback on their own if not supported
		if vx.graphicsProtocol < halfBlock {
			vx.graphicsProtocol = halfBlock
		}
	}

	ws, err := vx.reportWinsize()
	if err != nil {
		return nil, err
	}
	if ws.XPixel == 0 || ws.YPixel == 0 {
		log.Debug("pixel size not reported, setting graphics protocol to half block")
		vx.graphicsProtocol = halfBlock
	}
	vx.mu.Lock()
	cols, rows := vx.surfaceSize(ws)
	vx.screenNext.resize(cols, rows)
	vx.screenLast.resize(cols, rows)
	vx.winSize = ws
	vx.ready = true
	vx.mu.Unlock()
	// Set the next style to be a CursorBlock by default.
	vx.cursorNext.style = CursorBlock
	vx.PostEvent(ws)
	return vx, nil
}

func kittyKeyboardFlags(opts Options) int {
	flags := kittyKeyboardAllFlags
	if opts.CSIuBitMask != 0 {
		flags = int(opts.CSIuBitMask)
	}
	if opts.ReportKeyboardEvents {
		flags |= int(CSIuReportEvents)
	}
	return flags
}

// PostEvent inserts an event into the [Vaxis] event loop
func (vx *Vaxis) PostEvent(ev Event) {
	log.Debug("[event] %#v", ev)
	select {
	case vx.queue <- ev:
		return
	default:
		log.Warn("Event dropped: %T", ev)
	}
}

// PostEventBlocking inserts an event into the [Vaxis] event loop. The call will
// block if the queue is full. This method should only be used from a different
// goroutine than the main thread.
func (vx *Vaxis) PostEventBlocking(ev Event) {
	vx.queue <- ev
}

// SyncFunc queues a function to be called from the main thread. vaxis will call
// the function when the event is received in the main thread either through
// PollEvent or Events. A Redraw event will be sent to the host application
// after the function is completed
func (vx *Vaxis) SyncFunc(fn func()) {
	vx.PostEvent(SyncFunc(fn))
}

// PollEvent blocks until there is an Event, and returns that Event
func (vx *Vaxis) PollEvent() Event {
	ev, ok := <-vx.queue
	if !ok {
		return QuitEvent{}
	}
	return ev
}

// Events returns the channel of events.
func (vx *Vaxis) Events() chan Event {
	return vx.queue
}

// Close shuts down the event loops and returns the terminal to it's original
// state
func (vx *Vaxis) Close() {
	if vx.closed {
		return
	}
	vx.PostEvent(QuitEvent{})
	vx.closed = true

	defer close(vx.chQuit)

	_ = vx.Suspend()
	_ = vx.tty.Close()

	log.Info("Renders: %d", vx.renders)
	if vx.renders != 0 {
		log.Info("Time/render: %s", vx.elapsed/time.Duration(vx.renders))
	}
	log.Info("Cached characters: %d", len(vx.charCache))
}

func (vx *Vaxis) surfaceSize(size Resize) (cols int, rows int) {
	if vx.primaryScreen == nil {
		return size.Cols, size.Rows
	}
	return size.Cols, min(vx.primaryScreen.regionHeight, size.Rows)
}

// Size returns the current terminal size known to Vaxis.
func (vx *Vaxis) Size() Resize {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.winSize
}

// SetPrimaryScreenRegionHeight changes the primary-screen live region height.
// It panics unless Vaxis was created with PrimaryScreen options.
func (vx *Vaxis) SetPrimaryScreenRegionHeight(height int) {
	if height <= 0 {
		panic("vaxis: primary screen region height must be positive")
	}
	vx.mu.Lock()
	defer vx.mu.Unlock()
	if vx.primaryScreen == nil {
		panic("vaxis: SetPrimaryScreenRegionHeight called outside primary screen mode")
	}
	if vx.primaryScreen.regionHeight == height {
		return
	}
	if vx.primaryScreen.rendered {
		vx.primaryScreen.visualRows = vx.primaryVisualRowsForWidth(vx.winSize.Cols)
		vx.primaryScreen.resized = true
	}
	vx.primaryScreen.regionHeight = height
	cols, rows := vx.surfaceSize(vx.winSize)
	vx.screenNext.resize(cols, rows)
	vx.screenLast.resize(cols, rows)
	vx.refresh = true
}

// Resize applies a resize event to Vaxis' internal screen buffers. Applications
// should call this when handling a [Resize] event before drawing the next frame.
func (vx *Vaxis) Resize(size Resize) {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	cols, rows := vx.surfaceSize(size)
	oldCols, oldRows := vx.screenNext.size()
	if cols != oldCols || rows != oldRows {
		if vx.primaryScreen != nil && vx.primaryScreen.rendered {
			vx.primaryScreen.visualRows = vx.primaryVisualRowsForWidth(cols)
		}
		vx.screenNext.resize(cols, rows)
		vx.screenLast.resize(cols, rows)
		if vx.primaryScreen != nil && vx.primaryScreen.rendered {
			vx.primaryScreen.resized = true
		}
	}
	vx.winSize = size
	vx.refresh = true
}

// Append queues terminal output to be written before the primary-screen live
// region during the next Render. Append panics unless Vaxis was created with
// PrimaryScreen options.
func (vx *Vaxis) Append(p []byte) {
	vx.AppendString(string(p))
}

// AppendString queues terminal output to be written before the primary-screen
// live region during the next Render. AppendString panics unless Vaxis was
// created with PrimaryScreen options.
func (vx *Vaxis) AppendString(s string) {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	if vx.primaryScreen == nil {
		panic("vaxis: AppendString called outside primary screen mode")
	}
	vx.primaryScreen.append = append(vx.primaryScreen.append, s)
}

// AppendWriter returns an io.Writer that queues terminal output to be written
// before the primary-screen live region during Render. Writes panic unless
// Vaxis was created with PrimaryScreen options.
func (vx *Vaxis) AppendWriter() io.Writer {
	return appendWriter{vx: vx}
}

type appendWriter struct {
	vx *Vaxis
}

func (w appendWriter) Write(p []byte) (int, error) {
	w.vx.Append(p)
	return len(p), nil
}

// Render renders the model's content to the terminal
func (vx *Vaxis) Render() {
	start := time.Now()
	// defer renderBuf.Reset()
	if vx.primaryScreen != nil {
		vx.renderPrimary()
	} else {
		vx.render()
	}
	_, _ = vx.tw.Flush()
	// updating cursor state has to be after Flush, we check state change in
	// flush.
	vx.cursorLast = vx.cursorNext
	vx.elapsed += time.Since(start)
	vx.renders += 1
	vx.refresh = false
}

// Refresh forces a full render of the entire screen. Traditionally, this should
// be bound to Ctrl+l
func (vx *Vaxis) Refresh() {
	vx.refresh = true
	vx.Render()
}

func (vx *Vaxis) render() {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	var (
		reposition bool
		cursor     Style
	)
outerLast:
	// Delete any placements we don't have this round
	for _, p1 := range vx.graphicsLast {
		// Delete all previous placements on a refresh
		if vx.refresh {
			p1.deleteFn(vx.tw)
			continue
		}
		for _, p2 := range vx.graphicsNext {
			if samePlacement(p1, p2) {
				continue outerLast
			}
		}
		p1.deleteFn(vx.tw)
	}
	if vx.refresh {
		vx.graphicsLast = []*placement{}
	}
outerNew:
	// draw new placements
	for _, p1 := range vx.graphicsNext {
		for _, p2 := range vx.graphicsLast {
			if samePlacement(p1, p2) {
				// don't write existing placements
				continue outerNew
			}
		}
		vx.tw.writeCUP(p1.row+1, p1.col+1)
		p1.writeTo(vx.tw)
	}
	// Save this frame as the last frame
	vx.graphicsLast = vx.graphicsNext

	if vx.mouseShapeLast != vx.mouseShapeNext {
		_, _ = vx.tw.WriteString(tparm(mouseShape, vx.mouseShapeNext))
		vx.mouseShapeLast = vx.mouseShapeNext
	}
	for row := 0; row < vx.screenNext.rows; row += 1 {
		reposition = true
		nextRow := vx.screenNext.row(row)
		lastRow := vx.screenLast.row(row)
		for col := 0; col < len(nextRow); col += 1 {
			next := nextRow[col]
			if next.sixel {
				lastRow[col].sixel = true
				reposition = true
				continue
			}
			if next == lastRow[col] && !vx.refresh {
				reposition = true
				// Advance the column by the width of this
				// character
				skip := vx.advance(next)
				// skip := advance(next.Content)
				for i := 1; i < skip+1; i += 1 {
					if col+i >= len(nextRow) {
						break
					}
					// null out any cells we end up skipping
					lastRow[col+i] = Cell{}
				}
				col += skip
				continue
			}
			lastRow[col] = next
			if reposition {
				if cursor.Hyperlink != "" {
					cursor.Hyperlink = ""
					vx.tw.writeOSC8("", "")
				}
				vx.tw.writeCUP(row+1, col+1)
				reposition = false
			}
			// TODO Optimizations
			// 1. We could save two bytes when both FG and BG change
			// by combining into a single sequence. It saves one
			// "\x1b" and one "m". It adds a lot of complexity
			// though
			//
			// 2. We could save some more bytes when FG, BG, and Attr
			// all change. Lots of complexity to add this

			if cursor.Foreground != next.Foreground {
				fg := next.Foreground
				ps := fg.Params()
				if !vx.caps.rgb {
					ps = fg.asIndex().Params()
				}
				switch len(ps) {
				case 0:
					_, _ = vx.tw.WriteString(fgReset)
				case 1:
					switch {
					case ps[0] < 8:
						_, _ = vx.tw.WriteString(fgIndexedSeq[int(ps[0])])
					case ps[0] < 16:
						_, _ = vx.tw.WriteString(fgBrightSeq[int(ps[0]-8)])
					default:
						vx.tw.writeSGRIndexed(38, ps[0])
					}
				case 3:
					vx.tw.writeSGRRGB(38, ps[0], ps[1], ps[2])
				}
			}

			if cursor.Background != next.Background {
				bg := next.Background
				ps := bg.Params()
				if !vx.caps.rgb {
					ps = bg.asIndex().Params()
				}
				switch len(ps) {
				case 0:
					_, _ = vx.tw.WriteString(bgReset)
				case 1:
					switch {
					case ps[0] < 8:
						_, _ = vx.tw.WriteString(bgIndexedSeq[int(ps[0])])
					case ps[0] < 16:
						_, _ = vx.tw.WriteString(bgBrightSeq[int(ps[0]-8)])
					default:
						vx.tw.writeSGRIndexed(48, ps[0])
					}
				case 3:
					vx.tw.writeSGRRGB(48, ps[0], ps[1], ps[2])
				}
			}

			if vx.caps.styledUnderlines {
				if cursor.UnderlineColor != next.UnderlineColor {
					ul := next.UnderlineColor
					ps := ul.Params()
					if !vx.caps.rgb {
						ps = ul.asIndex().Params()
					}
					switch len(ps) {
					case 0:
						_, _ = vx.tw.WriteString(ulColorReset)
					case 1:
						vx.tw.writeSGRIndexed(58, ps[0])
					case 3:
						vx.tw.writeSGRRGB(58, ps[0], ps[1], ps[2])
					}
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
					_, _ = vx.tw.WriteString(boldSet)
				}
				if on&AttrDim != 0 {
					_, _ = vx.tw.WriteString(dimSet)
				}
				if on&AttrItalic != 0 {
					_, _ = vx.tw.WriteString(italicSet)
				}
				if on&AttrBlink != 0 {
					_, _ = vx.tw.WriteString(blinkSet)
				}
				if on&AttrReverse != 0 {
					_, _ = vx.tw.WriteString(reverseSet)
				}
				if on&AttrInvisible != 0 {
					_, _ = vx.tw.WriteString(hiddenSet)
				}
				if on&AttrStrikethrough != 0 {
					_, _ = vx.tw.WriteString(strikethroughSet)
				}
				if on&AttrOverline != 0 {
					_, _ = vx.tw.WriteString(overlineSet)
				}

				// If the bit is changed and is in previous, it
				// was turned off
				off := dAttr & attr
				if off&AttrBold != 0 {
					// Normal intensity isn't in terminfo
					_, _ = vx.tw.WriteString(boldDimReset)
					// Normal intensity turns off dim. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrDim != 0 {
						_, _ = vx.tw.WriteString(dimSet)
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					_, _ = vx.tw.WriteString(boldDimReset)
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						_, _ = vx.tw.WriteString(boldSet)
					}
				}
				if off&AttrItalic != 0 {
					_, _ = vx.tw.WriteString(italicReset)
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					_, _ = vx.tw.WriteString(blinkReset)
				}
				if off&AttrReverse != 0 {
					_, _ = vx.tw.WriteString(reverseReset)
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					_, _ = vx.tw.WriteString(hiddenReset)
				}
				if off&AttrStrikethrough != 0 {
					_, _ = vx.tw.WriteString(strikethroughReset)
				}
				if off&AttrOverline != 0 {
					_, _ = vx.tw.WriteString(overlineReset)
				}
			}

			if cursor.UnderlineStyle != next.UnderlineStyle {
				ulStyle := next.UnderlineStyle
				switch vx.caps.styledUnderlines {
				case true:
					vx.tw.writeUnderlineStyle(ulStyle)
				case false:
					switch ulStyle {
					case UnderlineOff:
						_, _ = vx.tw.WriteString(underlineReset)
					default:
						// Fallback to single underlines
						_, _ = vx.tw.WriteString(underlineSet)
					}
				}
			}

			if cursor.Hyperlink != next.Hyperlink {
				link := next.Hyperlink
				linkPs := next.HyperlinkParams
				if link == "" {
					linkPs = ""
				}
				vx.tw.writeOSC8(linkPs, link)
			}

			cursor = next.Style

			if next.Width == 0 {
				next.Width = vx.characterWidth(next.Grapheme)
			}

			switch {
			case next.Width == 0:
				_, _ = vx.tw.WriteString(" ")
			case next.Width > 1 && vx.caps.explicitWidth:
				vx.tw.writeExplicitWidth(next.Width, next.Grapheme)
			default:
				_, _ = vx.tw.WriteString(next.Grapheme)
			}
			skip := vx.advance(next)
			for i := 1; i < skip+1; i += 1 {
				if col+i >= len(nextRow) {
					break
				}
				// null out any cells we end up skipping
				lastRow[col+i] = Cell{}
			}
			col += skip
		}
	}
	if cursor.Hyperlink != "" {
		vx.tw.writeOSC8("", "")
	}
}

func (vx *Vaxis) renderPrimary() {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	primary := vx.primaryScreen
	if primary == nil {
		return
	}
	regionRows := vx.screenNext.rows
	if regionRows <= 0 || vx.winSize.Rows <= 0 || vx.winSize.Cols <= 0 {
		primary.append = nil
		return
	}
	regionChanged := vx.primaryRegionChanged()
	if primary.rendered && len(primary.append) == 0 && !primary.resized && !vx.refresh && !regionChanged {
		return
	}
	forceRegionPaint := false
	if primary.rendered {
		moveRows := regionRows
		if primary.resized && primary.visualRows > moveRows {
			moveRows = primary.visualRows
		}
		vx.moveToPrimaryRegionStart(moveRows)
		_, _ = vx.tw.WriteString("\x1B[J")
		primary.resized = false
		forceRegionPaint = true
	}
	if len(primary.append) > 0 {
		for _, s := range primary.append {
			_, _ = vx.tw.WriteString(primaryAppendString(s))
		}
	}
	primary.append = nil

	paintedRegion := false
	for row := 0; row < vx.screenNext.rows; row += 1 {
		nextRow := vx.screenNext.row(row)
		lastRow := vx.screenLast.row(row)
		renderRow := make([]Cell, len(nextRow))
		changed := vx.refresh || forceRegionPaint
		for col, cell := range nextRow {
			renderRow[col] = printablePrimaryCell(cell)
			if renderRow[col] != printablePrimaryCell(lastRow[col]) {
				changed = true
			}
		}
		if !changed {
			continue
		}
		copy(lastRow, renderRow)
		_, _ = vx.tw.WriteString(EncodeCells(trimPrimaryRenderRow(renderRow)))
		if row < vx.screenNext.rows-1 {
			_, _ = vx.tw.WriteString("\r\n")
		}
		paintedRegion = true
	}
	if paintedRegion {
		primary.rendered = true
		primary.visualRows = vx.primaryVisualRowsForWidth(vx.screenNext.cols)
	}
	_, _ = vx.tw.WriteString(sgrReset)
}

func (vx *Vaxis) primaryRegionChanged() bool {
	if vx.screenNext.rows != vx.screenLast.rows || vx.screenNext.cols != vx.screenLast.cols {
		return true
	}
	for row := 0; row < vx.screenNext.rows; row += 1 {
		nextRow := vx.screenNext.row(row)
		lastRow := vx.screenLast.row(row)
		for col, cell := range nextRow {
			if printablePrimaryCell(cell) != printablePrimaryCell(lastRow[col]) {
				return true
			}
		}
	}
	return false
}

func (vx *Vaxis) primaryVisualRowsForWidth(width int) int {
	if width <= 0 {
		return vx.screenLast.rows
	}
	rows := 0
	for row := 0; row < vx.screenLast.rows; row++ {
		cells := trimPrimaryRenderRow(vx.screenLast.row(row))
		cols := 0
		for _, cell := range cells {
			w := cell.Width
			if w <= 0 {
				w = 1
			}
			cols += w
		}
		rows += max(1, (cols+width-1)/width)
	}
	return max(1, rows)
}

func printablePrimaryCell(cell Cell) Cell {
	if cell.Grapheme == "" {
		cell.Character = Character{Grapheme: " ", Width: 1}
	}
	return cell
}

func trimPrimaryRenderRow(row []Cell) []Cell {
	end := len(row)
	for end > 0 && isDefaultPrimaryBlank(row[end-1]) {
		end--
	}
	return row[:end]
}

func isDefaultPrimaryBlank(cell Cell) bool {
	return printablePrimaryCell(cell) == Cell{Character: Character{Grapheme: " ", Width: 1}}
}

func (vx *Vaxis) moveToPrimaryRegionStart(regionRows int) {
	_, _ = vx.tw.WriteString("\r")
	if regionRows <= 1 {
		return
	}
	_, _ = vx.tw.WriteString(tparm("\x1B[%dA", regionRows-1))
}

func primaryAppendString(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	return s
}

func (vx *Vaxis) handleSequence(seq ansi.Sequence) {
	log.Trace("[stdin] sequence: %s", seq)
	switch seq := seq.(type) {
	case ansi.Print:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEventBlocking(key)
	case ansi.C0:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEventBlocking(key)
	case ansi.ESC:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEventBlocking(key)
	case ansi.SS3:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEventBlocking(key)
	case ansi.CSI:
		intermediates := seq.Intermediates()
		switch seq.Final {
		case 'c':
			if len(intermediates) == 1 && intermediates[0] == '?' {
				for _, ps := range seq.Params() {
					switch ps {
					case 4:
						vx.PostEventBlocking(capabilitySixel{})
					}
				}
				vx.PostEventBlocking(primaryDeviceAttribute{})
				return
			}
		case 'I':
			vx.PostEventBlocking(FocusIn{})
			return
		case 'O':
			vx.PostEventBlocking(FocusOut{})
			return
		case 'R':
			// KeyF1 or DSRCPR
			// This could be an F1 key, we need to buffer if we have
			// requested a DSRCPR (cursor position report)
			//
			// Kitty keyboard protocol disambiguates this scenario,
			// hopefully people are using that
			vx.mu.Lock()
			reqCursorPos := vx.reqCursorPos
			vx.reqCursorPos = false
			vx.mu.Unlock()
			if reqCursorPos {
				if seq.NumParameters != 2 {
					log.Error("not enough DSRCPR params")
					return
				}
				vx.chCursorPos <- [2]int{
					seq.Param(0),
					seq.Param(1),
				}
				return
			}
		case 'S':
			if len(intermediates) == 1 && intermediates[0] == '?' {
				if seq.NumParameters < 3 {
					break
				}
				switch seq.Param(0) {
				case 2:
					if seq.Param(1) == 0 {
						vx.PostEventBlocking(capabilitySixel{})
					}
				}
				return
			}
		case 'n':
			if len(intermediates) == 1 && intermediates[0] == '?' {
				if seq.NumParameters != 2 {
					break
				}
				switch seq.Param(0) {
				case colorThemeResp: // 997
					m := ColorThemeMode(seq.Param(1))
					vx.PostEventBlocking(ColorThemeUpdate{
						Mode: m,
					})
				}
				return
			}
		case 'y':
			// DECRPM - DEC Report Mode
			if seq.NumParameters < 1 {
				log.Error("not enough DECRPM params")
				return
			}
			switch seq.Param(0) {
			case 1016:
				if seq.NumParameters < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Param(1) {
				case 1, 2:
					vx.PostEventBlocking(capabilitySgrPixels{})
				}
			case 2026:
				if seq.NumParameters < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Param(1) {
				case 1, 2:
					vx.PostEventBlocking(synchronizedUpdates{})
				}
			case 2027:
				if seq.NumParameters < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Param(1) {
				case 1, 2:
					vx.PostEventBlocking(unicodeCoreCap{})
				}
			case 2031:
				if seq.NumParameters < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Param(1) {
				case 1, 2:
					vx.PostEventBlocking(notifyColorChange{})
				}
			}
			return
		case 'u':
			if len(intermediates) == 1 && intermediates[0] == '?' {
				vx.PostEventBlocking(kittyKeyboard{})
				return
			}
		case '~':
			if len(intermediates) == 0 {
				if seq.NumParameters == 0 {
					log.Error("[CSI] unknown sequence with final '~'")
					return
				}
				switch seq.Param(0) {
				case 200:
					vx.pastePending = true
					vx.PostEventBlocking(PasteStartEvent{})
					return
				case 201:
					vx.pastePending = false
					vx.PostEventBlocking(PasteEndEvent{})
					return
				}
			}
		case 'M', 'm':
			vx.mu.Lock()
			ws := vx.winSize
			sgrPixels := vx.caps.sgrPixels
			vx.mu.Unlock()
			mouse, ok := parseMouseEvent(seq, ws, sgrPixels)
			if ok {
				vx.PostEventBlocking(mouse)
			}
			return
		case 't':
			if seq.NumParameters < 3 {
				log.Error("[CSI] unknown sequence: %s", seq)
				return
			}
			// CSI <type> ; <height> ; <width> t
			typ := seq.Param(0)
			h := seq.Param(1)
			w := seq.Param(2)
			switch typ {
			case 4:
				vx.mu.Lock()
				report := vx.caps.reportSizePixels
				vx.mu.Unlock()
				if !report {
					// Gate on this so we only report this
					// once at startup
					vx.PostEventBlocking(textAreaPix{})
					return
				}
				vx.postSizeReport(sizeReport{
					size:   Resize{XPixel: w, YPixel: h},
					pixels: true,
				})
			case 8:
				vx.mu.Lock()
				report := vx.caps.reportSizeChars
				vx.mu.Unlock()
				if !report {
					// Gate on this so we only report this
					// once at startup. This also means we
					// can set the size directly and won't
					// have race conditions
					vx.PostEventBlocking(textAreaChar{})
					return
				}
				vx.postSizeReport(sizeReport{
					size:  Resize{Cols: w, Rows: h},
					chars: true,
				})
			case 48:
				// CSI <type> ; <height> ; <width> ; <height_pix> ; <width_pix> t
				switch seq.NumParameters {
				case 5:
					size := Resize{
						Cols:   w,
						Rows:   h,
						YPixel: seq.Param(3),
						XPixel: seq.Param(4),
					}
					vx.mu.Lock()
					changed := size != vx.winSize
					ready := vx.ready
					vx.caps.inBandResize = true
					vx.mu.Unlock()
					if !ready {
						vx.postSizeReport(sizeReport{
							size:   size,
							chars:  true,
							pixels: true,
						})
						return
					}
					if changed {
						vx.PostEventBlocking(size)
					}
				}
			}
			return
		}

		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEventBlocking(key)
	case ansi.DCS:
		intermediates := seq.Intermediates()
		switch seq.Final {
		case 'r':
			if len(intermediates) < 1 {
				return
			}
			switch intermediates[0] {
			case '+':
				// XTGETTCAP response
				if seq.NumParameters < 1 {
					return
				}
				if seq.Params()[0] == 0 {
					return
				}
				vals := strings.Split(string(seq.Data), "=")
				if len(vals) != 2 {
					log.Error("error parsing XTGETTCAP: %s", string(seq.Data))
				}
				switch vals[0] {
				case hexEncode("Smulx"):
					vx.PostEventBlocking(styledUnderlines{})
				case hexEncode("RGB"):
					vx.PostEventBlocking(truecolor{})
				}
			case '$':
				// DECRQSS response (DECRPSS)

				// DECSCUSR (user cursor style)
				if strings.HasSuffix(string(seq.Data), " q") {
					// Convert the rune into a digit
					cursorStyle := seq.Data[0]
					// Valid cursor styles are 0-6
					if cursorStyle < '0' || cursorStyle > '6' {
						log.Warn("invalid DECSCUSR: %d", cursorStyle)
						return
					}
					log.Debug("User cursor style discovered: %v",
						CursorStyle(cursorStyle-0x30))
					vx.mu.Lock()
					vx.userCursorStyle = CursorStyle(cursorStyle - 0x30)
					vx.mu.Unlock()
				}

			}
		case '|':
			if len(intermediates) < 1 {
				return
			}
			switch intermediates[0] {
			case '!':
				if string(seq.Data) == hexEncode("~VTE") {
					// VTE supports styled underlines but
					// doesn't respond to XTGETTCAP
					vx.PostEventBlocking(styledUnderlines{})
				}
			case '>':
				vx.PostEventBlocking(terminalID(seq.Data))
			}
		}
	case ansi.APC:
		if len(seq.Data) == 0 {
			return
		}
		if strings.HasPrefix(seq.Data, "G") {
			vx.PostEventBlocking(kittyGraphics{})
		}
	case ansi.OSC:
		if seq.InvalidUTF8 {
			return
		}
		if strings.HasPrefix(string(seq.Payload), "4") {
			if vx.CanReportColor() {
				postQueryResponse(vx.chColor, string(seq.Payload))
			} else {
				vx.PostEventBlocking(capabilityOsc4{})
			}
		}
		if strings.HasPrefix(string(seq.Payload), "10") {
			if vx.CanReportForegroundColor() {
				postQueryResponse(vx.chFg, string(seq.Payload))
			} else {
				vx.PostEventBlocking(capabilityOsc10{})
			}
		}
		if strings.HasPrefix(string(seq.Payload), "11") {
			if vx.CanReportBackgroundColor() {
				postQueryResponse(vx.chBg, string(seq.Payload))
			} else {
				vx.PostEventBlocking(capabilityOsc11{})
			}
		}
		if strings.HasPrefix(string(seq.Payload), "52") {
			vals := strings.Split(string(seq.Payload), ";")
			if len(vals) != 3 {
				log.Error("invalid OSC 52 payload")
				return
			}
			b, err := base64.StdEncoding.DecodeString(vals[2])
			if err != nil {
				log.Error("couldn't decode OSC 52: %v", err)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			select {
			case vx.chClipboard <- string(b):
			case <-ctx.Done():
			}
		}
		if strings.HasPrefix(string(seq.Payload), "176") {
			vals := strings.Split(string(seq.Payload), ";")
			if len(vals) != 2 {
				log.Error("invalid OSC 176 payload")
				return
			}
			vx.PostEvent(appID(vals[1]))
		}
	}
}

// QueryColor queries the host terminal for an indexed color and returns
// it as an instance of an RGB vaxis.Color. If the host terminal doesn't
// support this, Color(0) is returned instead. Make sure not to run this
// in the same goroutine as Vaxis runs in or deadlock will occur.
func (vx *Vaxis) QueryColor(c Color) Color {
	return vx.QueryColorContext(context.Background(), c)
}

// QueryColorContext queries the host terminal for an indexed color and waits
// for the response until ctx is cancelled. If the host terminal doesn't
// support this or ctx is cancelled first, Color(0) is returned instead. Make
// sure not to run this in the same goroutine as Vaxis runs in or deadlock will
// occur.
func (vx *Vaxis) QueryColorContext(ctx context.Context, c Color) Color {
	if ctx == nil {
		ctx = context.Background()
	}
	if !vx.CanReportColor() {
		return Color(0)
	}
	p := c.Params()
	if len(p) == 3 {
		// If an RGB color was passed, return it as is.
		return c
	}
	if len(p) != 1 {
		return Color(0)
	}
	if ctx.Err() != nil {
		return Color(0)
	}

	vx.chColorMu.Lock()
	defer vx.chColorMu.Unlock()
	if ctx.Err() != nil {
		return Color(0)
	}

	drainQueryResponses(vx.chColor)
	vx.writeControlString(tparm(osc4, p[0]))
	var resp string
	select {
	case resp = <-vx.chColor:
	case <-ctx.Done():
		return Color(0)
	}

	var r, g, b int
	prefix := fmt.Sprintf("4;%v;", p[0])
	_, err := fmt.Sscanf(resp, prefix+"rgb:%x/%x/%x", &r, &g, &b)
	if err != nil {
		log.Error("QueryColor: failed to parse the OSC 4 response: %s", err)
		return Color(0)
	}
	// The returned value can in principle be 16 bits per channel, however
	// we are not aware of any terminal that would do this, foot for
	// instance just repeats the same 8 bits twice. Hence we only take the
	// lower 8 bits.
	return RGBColor(uint8(r), uint8(g), uint8(b))
}

// QueryForeground queries the host terminal for foreground color and returns
// it as an instance of vaxis.Color. If the host terminal doesn't support this,
// Color(0) is returned instead. Make sure not to run this in the same
// goroutine as Vaxis runs in or deadlock will occur.
func (vx *Vaxis) QueryForeground() Color {
	return vx.QueryForegroundContext(context.Background())
}

// QueryForegroundContext queries the host terminal for foreground color and
// waits for the response until ctx is cancelled. If the host terminal doesn't
// support this or ctx is cancelled first, Color(0) is returned instead. Make
// sure not to run this in the same goroutine as Vaxis runs in or deadlock will
// occur.
func (vx *Vaxis) QueryForegroundContext(ctx context.Context) Color {
	if ctx == nil {
		ctx = context.Background()
	}
	if !vx.CanReportForegroundColor() {
		return Color(0)
	}
	if ctx.Err() != nil {
		return Color(0)
	}

	vx.chFgMu.Lock()
	defer vx.chFgMu.Unlock()
	if ctx.Err() != nil {
		return Color(0)
	}

	drainQueryResponses(vx.chFg)
	vx.writeControlString(osc10)
	var resp string
	select {
	case resp = <-vx.chFg:
	case <-ctx.Done():
		return Color(0)
	}

	var r, g, b int
	_, err := fmt.Sscanf(resp, "10;rgb:%x/%x/%x", &r, &g, &b)
	if err != nil {
		log.Error("QueryForeground: failed to parse the OSC 10 response: %s", err)
		return Color(0)
	}
	// Similar to QueryColor above.
	return RGBColor(uint8(r), uint8(g), uint8(b))
}

// QueryBackground queries the host terminal for background color and returns
// it as an instance of vaxis.Color. If the host terminal doesn't support this,
// Color(0) is returned instead. Make sure not to run this in the same
// goroutine as Vaxis runs in or deadlock will occur.
func (vx *Vaxis) QueryBackground() Color {
	return vx.QueryBackgroundContext(context.Background())
}

// QueryBackgroundContext queries the host terminal for background color and
// waits for the response until ctx is cancelled. If the host terminal doesn't
// support this or ctx is cancelled first, Color(0) is returned instead. Make
// sure not to run this in the same goroutine as Vaxis runs in or deadlock will
// occur.
func (vx *Vaxis) QueryBackgroundContext(ctx context.Context) Color {
	if ctx == nil {
		ctx = context.Background()
	}
	if !vx.CanReportBackgroundColor() {
		return Color(0)
	}
	if ctx.Err() != nil {
		return Color(0)
	}

	vx.chBgMu.Lock()
	defer vx.chBgMu.Unlock()
	if ctx.Err() != nil {
		return Color(0)
	}

	drainQueryResponses(vx.chBg)
	vx.writeControlString(osc11)
	var resp string
	select {
	case resp = <-vx.chBg:
	case <-ctx.Done():
		return Color(0)
	}

	var r, g, b int
	_, err := fmt.Sscanf(resp, "11;rgb:%x/%x/%x", &r, &g, &b)
	if err != nil {
		log.Error("QueryBackground: failed to parse the OSC 11 response: %s", err)
		return Color(0)
	}
	// Similar to QueryColor above.
	return RGBColor(uint8(r), uint8(g), uint8(b))
}

func drainQueryResponses(ch chan string) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func postQueryResponse(ch chan string, resp string) {
	select {
	case ch <- resp:
	default:
	}
}

func (vx *Vaxis) drainSizeReports() {
	for {
		select {
		case <-vx.chSizeReport:
		default:
			return
		}
	}
}

func (vx *Vaxis) postSizeReport(report sizeReport) {
	select {
	case vx.chSizeReport <- report:
	default:
	}
}

func (vx *Vaxis) detectResize(blocking bool) {
	ws, err := vx.reportWinsize()
	if err != nil {
		log.Error("couldn't report winsize: %v", err)
		return
	}
	vx.mu.Lock()
	changed := ws != vx.winSize
	ready := vx.ready
	vx.mu.Unlock()
	if !ready || !changed {
		return
	}
	if blocking {
		vx.PostEventBlocking(ws)
		return
	}
	vx.PostEvent(ws)
}

func (vx *Vaxis) cellPixelSize() (int, int) {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	if vx.winSize.Cols == 0 || vx.winSize.Rows == 0 {
		return 0, 0
	}
	return vx.winSize.XPixel / vx.winSize.Cols, vx.winSize.YPixel / vx.winSize.Rows
}

func (vx *Vaxis) writeControl(p []byte) {
	if vx.tw != nil {
		_, _ = vx.tw.WriteControl(p)
		return
	}
	if vx.tty != nil {
		_, _ = vx.tty.Write(p)
	}
}

func (vx *Vaxis) writeControlString(s string) {
	if vx.tw != nil {
		_, _ = vx.tw.WriteControlString(s)
		return
	}
	if vx.tty != nil {
		_, _ = io.WriteString(vx.tty, s)
	}
}

func (vx *Vaxis) sendQueries() {
	// always query in the alt screen so a terminal who doesn't understand
	// this doesn't get messed up. We are in full control of the alt screen
	vx.enterAltScreen()
	defer vx.exitAltScreen()

	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		vx.PostEvent(truecolor{})
	}

	_, _ = vx.tw.WriteControlString(userCursorStyle)
	_, _ = vx.tw.WriteControlString(decrqm(synchronizedUpdate))
	_, _ = vx.tw.WriteControlString(decrqm(unicodeCore))
	_, _ = vx.tw.WriteControlString(decrqm(colorThemeUpdates))
	_, _ = vx.tw.WriteControlString(decrqm(mouseSGRPixels))
	// We blindly enable in band resize. We get a response immediately if it
	// is supported
	_, _ = vx.tw.WriteControlString(decset(inBandResize))
	_, _ = vx.tw.WriteControlString(xtversion)
	_, _ = vx.tw.WriteControlString(kittyKBQuery)
	_, _ = vx.tw.WriteControlString(kittyGquery)
	_, _ = vx.tw.WriteControlString(xtsmSixelGeom)
	// Can the terminal report its own size?
	_, _ = vx.tw.WriteControlString(textAreaSize)

	// Explicit width query
	vx.tw.writeControlCUP(1, 1)
	vx.tw.writeControlExplicitWidth(1, " ")
	_, col := vx.CursorPosition()
	if col == 1 {
		log.Debug("[capability] explicit width supported")
		vx.mu.Lock()
		vx.caps.explicitWidth = true
		vx.mu.Unlock()
	}

	// Query some terminfo capabilities
	// Just another way to see if we have RGB support
	_, _ = vx.tw.WriteControlString(xtgettcap("RGB"))
	// Does the terminal respond to OSC 4/10/11 queries?
	// Use color index 8 for OSC 4 to ignore buggy implementations that only respond to 0-7.
	_, _ = vx.tw.WriteControlString(tparm(osc4, 8))
	_, _ = vx.tw.WriteControlString(osc10)
	_, _ = vx.tw.WriteControlString(osc11)
	// Back up the current app ID
	_, _ = vx.tw.WriteControlString(getAppID)
	// We request Smulx to check for styled underlines. Technically, Smulx
	// only means the terminal supports different underline types (curly,
	// dashed, etc), but we'll assume the terminal also suppports underline
	// colors (CSI 58 : ...)
	_, _ = vx.tw.WriteControlString(xtgettcap("Smulx"))
	// Need to send tertiary for VTE based terminals. These don't respond to
	// XTGETTCAP
	_, _ = vx.tw.WriteControlString(tertiaryAttributes)
	// Send Device Attributes is last. Everything responds, and when we get
	// a response we'll return from init
	_, _ = vx.tw.WriteControlString(primaryAttributes)
}

// enableModes enables all the modes we want
func (vx *Vaxis) enableModes() {
	// kitty keyboard
	if vx.caps.kittyKeyboard {
		_, _ = vx.tw.WriteControlString(tparm(kittyKBEnable, vx.kittyFlags))
	}
	// sixel scrolling
	if vx.caps.sixels {
		_, _ = vx.tw.WriteControlString(decset(sixelScrolling))
	}
	// Mode 2027, unicode segmentation (for correct emoji/wc widths). We
	// only enable if we don't also have explicitWidth
	if vx.caps.unicodeCore && !vx.caps.explicitWidth {
		_, _ = vx.tw.WriteControlString(decset(unicodeCore))
	}

	// Mode 2031: color scheme updates
	if vx.caps.colorThemeUpdates {
		_, _ = vx.tw.WriteControlString(decset(colorThemeUpdates))
		// Let's query the current mode also
		_, _ = vx.tw.WriteControlString(tparm(dsr, colorThemeReq))
	}
	if vx.caps.inBandResize {
		_, _ = vx.tw.WriteControlString(decset(inBandResize))
	}

	_, _ = vx.tw.WriteControlString(decset(mouseFocusEvents)) // window focus events
	// TODO: query for bracketed paste support?
	_, _ = vx.tw.WriteControlString(decset(bracketedPaste)) // bracketed paste
	_, _ = vx.tw.WriteControlString(decset(cursorKeys))     // application cursor keys
	_, _ = vx.tw.WriteControlString(applicationMode)        // application cursor keys mode

	// TODO: Query for mouse modes or just hope for the best? In the
	// meantime, we enable button events, then all events. Terminals which
	// support both will enable the latter. Terminals which support only the
	// first will enable button events, then ignore the all events mode.
	if !vx.disableMouse {
		_, _ = vx.tw.WriteControlString(decset(mouseButtonEvents))
		_, _ = vx.tw.WriteControlString(decset(mouseAllEvents))
		_, _ = vx.tw.WriteControlString(decset(mouseSGR))
		if vx.caps.sgrPixels {
			_, _ = vx.tw.WriteControlString(decset(mouseSGRPixels))
		}
	}
}

func (vx *Vaxis) disableModes() {
	_, _ = vx.tw.WriteControlString(sgrReset)               // reset fg, bg, attrs
	_, _ = vx.tw.WriteControlString(decrst(bracketedPaste)) // bracketed paste
	_, _ = vx.tw.WriteControlString(decrst(mouseFocusEvents))
	if vx.caps.kittyKeyboard {
		_, _ = vx.tw.WriteControlString(kittyKBPop) // kitty keyboard
	}
	_, _ = vx.tw.WriteControlString(decrst(cursorKeys))
	_, _ = vx.tw.WriteControlString(numericMode)
	if !vx.disableMouse {
		_, _ = vx.tw.WriteControlString(decrst(mouseButtonEvents))
		_, _ = vx.tw.WriteControlString(decrst(mouseAllEvents))
		_, _ = vx.tw.WriteControlString(decrst(mouseSGR))

		if vx.caps.sgrPixels {
			_, _ = vx.tw.WriteControlString(decrst(mouseSGRPixels))
		}
	}
	if vx.caps.sixels {
		_, _ = vx.tw.WriteControlString(decrst(sixelScrolling))
	}
	if vx.caps.unicodeCore && !vx.caps.explicitWidth {
		_, _ = vx.tw.WriteControlString(decrst(unicodeCore))
	}
	if vx.caps.colorThemeUpdates {
		_, _ = vx.tw.WriteControlString(decrst(colorThemeUpdates))
	}
	if vx.caps.osc176 {
		_, _ = vx.tw.WriteControlString(tparm(setAppID, vx.appIDLast))
	}
	if vx.caps.inBandResize {
		_, _ = vx.tw.WriteControlString(decrst(inBandResize))
	}
	// Most terminals default to "text" mouse shape
	_, _ = vx.tw.WriteControlString(tparm(mouseShape, MouseShapeTextInput))
}

func (vx *Vaxis) enterAltScreen() {
	vx.tw.vx.refresh = true
	_, _ = vx.tw.WriteControlString(decset(alternateScreen))
	_, _ = vx.tw.WriteControlString(hideCursorSeq)
}

func (vx *Vaxis) exitAltScreen() {
	vx.HideCursor()
	_, _ = vx.tw.WriteControlString(showCursorSeq)
	_, _ = vx.tw.WriteControlString(clear)
	_, _ = vx.tw.WriteControlString(decrst(alternateScreen))
}

func (vx *Vaxis) exitPrimaryScreen() {
	vx.HideCursor()
	if vx.primaryScreen != nil && vx.primaryScreen.rendered {
		_, _ = vx.tw.WriteControlString("\r\n\r")
	}
	_, _ = vx.tw.WriteControlString(showCursorSeq)
}

// Suspend takes Vaxis out of fullscreen state, disables all terminal modes,
// stops listening for signals, and returns the terminal to it's original state.
// Suspend can be useful to, for example, drop out of the full screen TUI and
// run another TUI. The state of vaxis will be retained, so you can reenter the
// original state by calling Resume
func (vx *Vaxis) Suspend() error {
	// HACK: The parser could be hanging for input. Because we have a handle
	// on a real terminal, we can't "actually" close the FD, so the poll
	// doesn't necessarily wake on the close call. However, we are the only
	// one reading from it...so we have to do a little dance:
	// 1. Send a signal that we want to close
	// 2. Send a DA1 query so there is data on the reader, breaking the read
	//    loop
	// 3. Confirm we have closed
	vx.parser.Close()
	_ = vx.tty.StopInput()
	vx.writeControlString(primaryAttributes)
	vx.parser.WaitClose()

	vx.disableModes()
	if vx.primaryScreen == nil {
		vx.exitAltScreen()
	} else {
		vx.exitPrimaryScreen()
	}

	// Reset to user value, or CursorDefault.
	_, _ = vx.tw.WriteControlString(tparm(cursorStyleSet, int(vx.userCursorStyle)))
	// Always show the cursor on exit
	_, _ = vx.tw.WriteControlString(showCursorSeq)
	// Reset internal state to match reality
	vx.cursorLast.style = vx.userCursorStyle

	signal.Stop(vx.chSigKill)
	signal.Stop(vx.chSigWinSz)
	_ = vx.tty.Reset()
	return nil
}

// openTty opens the /dev/tty device, makes it raw, and starts an input parser
func (vx *Vaxis) openTty() error {
	var t tty
	if vx.withConsole != nil {
		t = consoleTTY{Console: vx.withConsole}
	} else {
		var err error
		t, err = openTTY(vx.withTty)
		if err != nil {
			return err
		}
	}
	vx.tty = t

	err := vx.tty.SetRaw()
	if err != nil {
		return err
	}
	err = vx.tty.StartInput(vx)
	if err != nil {
		return err
	}
	vx.tw = newWriter(vx)
	vx.parser = ansi.NewParser(vx.tty, ansi.ParserModeInput)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				vx.Close()
				panic(err)
			}
		}()
		for {
			select {
			case seq := <-vx.parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					return
				default:
					vx.handleSequence(seq)
				}
			case <-vx.chSigWinSz:
				go vx.detectResize(true)
			case <-vx.chSigKill:
				vx.Close()
				return
			}
		}
	}()
	return nil
}

// Resume returns the application to it's fullscreen state, re-enters raw mode,
// and reenables input parsing. Upon resuming, a Resize event will be delivered.
// It is entirely possible the terminal was resized while suspended.
func (vx *Vaxis) Resume() error {
	err := vx.openTty()
	if err != nil {
		return err
	}

	if vx.primaryScreen == nil {
		vx.enterAltScreen()
	}
	vx.enableModes()

	if !vx.noSignals {
		vx.setupSignals()
	}
	go vx.detectResize(false)
	return nil
}

// HideCursor hides the hardware cursor
func (vx *Vaxis) HideCursor() {
	vx.cursorNext.visible = false
}

// ShowCursor shows the cursor at the given colxrow, with the given style. The
// passed column and row are 0-indexed and global. To show the cursor relative
// to a window, use [Window.ShowCursor]
func (vx *Vaxis) ShowCursor(col int, row int, style CursorStyle) {
	vx.cursorNext.style = style
	vx.cursorNext.col = col
	vx.cursorNext.row = row
	vx.cursorNext.visible = true
}

func (vx *Vaxis) showCursor() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(vx.cursorStyle())
	buf.WriteString(tparm(cup, vx.cursorNext.row+1, vx.cursorNext.col+1))
	buf.WriteString(showCursorSeq)
	return buf.String()
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func (vx *Vaxis) CursorPosition() (row int, col int) {
	// DSRCPR - reports cursor position
	vx.mu.Lock()
	vx.reqCursorPos = true
	vx.mu.Unlock()
	vx.writeControlString(dsrcpr)
	timeout := time.NewTimer(50 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warn("CursorPosition timed out")
		vx.mu.Lock()
		vx.reqCursorPos = false
		vx.mu.Unlock()
		return -1, -1
	case pos := <-vx.chCursorPos:
		return pos[0] - 1, pos[1] - 1
	}
}

// CursorStyle is the style to display the hardware cursor
type CursorStyle int

const (
	CursorDefault = iota
	CursorBlockBlinking
	CursorBlock
	CursorUnderlineBlinking
	CursorUnderline
	CursorBeamBlinking
	CursorBeam
)

func (vx *Vaxis) cursorStyle() string {
	return tparm(cursorStyleSet, int(vx.cursorNext.style))
}

// ClipboardPush copies the provided string to the system clipboard
func (vx *Vaxis) ClipboardPush(s string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	vx.writeControlString(tparm(osc52put, b64))
}

// ClipboardPop requests the content from the system clipboard. ClipboardPop works by
// requesting the data from the underlying terminal, which responds back with
// the data. Depending on usage, this could take some time. Callers can provide
// a context to set a deadline for this function to return. An error will be
// returned if the context is cancelled.
func (vx *Vaxis) ClipboardPop(ctx context.Context) (string, error) {
	vx.writeControlString(osc52pop)
	select {
	case str := <-vx.chClipboard:
		return str, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Notify (attempts) to send a system notification. If title is the empty
// string, OSC9 will be used - otherwise osc777 is used
func (vx *Vaxis) Notify(title string, body string) {
	if title == "" {
		vx.writeControlString(tparm(osc9notify, body))
		return
	}
	vx.writeControlString(tparm(osc777notify, title, body))
}

// SetTitle sets the terminal's title via OSC 2
func (vx *Vaxis) SetTitle(s string) {
	vx.writeControlString(tparm(setTitle, s))
}

// SetPath sets the terminal's working directory via OSC 7
func (vx *Vaxis) NotifyWorkingDirectory(s string) {
	vx.writeControlString(tparm(setCWD, s))
}

// SetAppID sets the terminal's application ID via OSC 176
func (vx *Vaxis) SetAppID(s string) {
	vx.writeControlString(tparm(setAppID, s))
}

// Bell sends a BEL control signal to the terminal
func (vx *Vaxis) Bell() {
	vx.writeControl([]byte{0x07})
}

// advance returns the extra amount to advance the column by when rendering
func (vx *Vaxis) advance(cell Cell) int {
	if cell.Width == 0 {
		cell.Width = vx.characterWidth(cell.Grapheme)
	}
	// TODO: use max(cell.Width-1, 0) when >go1.19
	w := cell.Width - 1
	if w < 0 {
		return 0
	}
	return w
}

// RenderedWidth returns the rendered width of the provided string. The result
// is dependent on if your terminal can support unicode properly.
//
// This is best effort. It will usually be correct, and in the few cases it's
// wrong will end up wrong in the nicer-rendering way (complex emojis will have
// extra space after them. This is preferable to messing up the internal model)
//
// This call can be expensive, callers should consider caching the result for
// strings or characters which will need to be measured frequently
func (vx *Vaxis) RenderedWidth(s string) int {
	if vx.caps.unicodeCore || vx.caps.explicitWidth {
		return gwidth(s, unicodeStd)
	}
	if vx.caps.noZWJ {
		log.Debug("nozwj")
		return gwidth(s, noZWJ)
	}
	return gwidth(s, wcwidth)
}

// characterWidth measures the width of a grapheme cluster, caching the result .
// We only ever call this with characters, making it highly cacheable since
// there is likely to only ever be a finite set of characters in the lifetime of
// an application
func (vx *Vaxis) characterWidth(s string) int {
	w, ok := vx.charCache[s]
	if ok {
		return w
	}
	w = vx.RenderedWidth(s)
	vx.charCache[s] = w
	return w
}

// SetMouseShape sets the shape of the mouse
func (vx *Vaxis) SetMouseShape(shape MouseShape) {
	vx.mouseShapeNext = shape
}

// TerminalID returns the terminal name and version advertised by the terminal,
// if supported. The actual format is implementation-defined, but it is safe to
// assume that the ID will start with the terminal name.
// Some examples: "foot(1.17.2)"; "WezTerm 20210502-154244-3f7122cb"
func (vx *Vaxis) TerminalID() string {
	return string(vx.termID)
}

func (vx *Vaxis) CanRGB() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.rgb
}

func (vx *Vaxis) CanKittyGraphics() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.kittyGraphics
}

func (vx *Vaxis) CanKittyKeyboard() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.kittyKeyboard
}

func (vx *Vaxis) CanSixel() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.sixels
}

func (vx *Vaxis) CanReportColor() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.osc4
}

func (vx *Vaxis) CanReportForegroundColor() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.osc10
}

func (vx *Vaxis) CanReportBackgroundColor() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.osc11
}

func (vx *Vaxis) CanDisplayGraphics() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.sixels || vx.caps.kittyGraphics
}

func (vx *Vaxis) CanSetAppID() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.osc176
}

func (vx *Vaxis) CanUnicodeCore() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.unicodeCore
}

func (vx *Vaxis) CanExplicitWidth() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.explicitWidth
}

func (vx *Vaxis) CanInBandResize() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.inBandResize
}

func (vx *Vaxis) nextGraphicID() uint64 {
	vx.graphicsIDNext += 1
	return vx.graphicsIDNext
}
