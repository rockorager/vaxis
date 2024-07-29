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
	"sync/atomic"
	"time"

	"github.com/containerd/console"

	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
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
	osc11              bool
	osc176             bool
	inBandResize       bool
}

type cursorState struct {
	row     int
	col     int
	style   CursorStyle
	visible bool
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

	// CSIuBitMask is the bit mask to use for CSIu key encoding, when
	// available. This has no effect if DisableKittyKeyboard is true
	CSIuBitMask CSIuBitMask
}

type CSIuBitMask int

const (
	CSIuDisambiguate CSIuBitMask = 1 << iota
	CSIuReportEvents
	CSIuAlternateKeys
	CSIuAllKeys
	CSIuAssociatedText
)

type Vaxis struct {
	queue            chan Event
	console          console.Console
	parser           *ansi.Parser
	tw               *writer
	screenNext       *screen
	screenLast       *screen
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
	nextSize         Resize
	chSizeDone       chan bool
	caps             capabilities
	graphicsProtocol int
	graphicsIDNext   uint64
	reqCursorPos     int32
	charCache        map[string]int
	cursorNext       cursorState
	cursorLast       cursorState
	closed           bool
	refresh          bool
	kittyFlags       int
	disableMouse     bool
	chBg             chan string

	xtwinops bool

	withTty string

	termID terminalID

	renders int
	elapsed time.Duration

	mu     sync.Mutex
	resize int32
}

// New creates a new [Vaxis] instance. Calling New will query the underlying
// terminal for supported features and enter the alternate screen
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
		kittyFlags: int(CSIuDisambiguate),
	}

	if opts.CSIuBitMask > CSIuDisambiguate {
		vx.kittyFlags = int(opts.CSIuBitMask)
	}

	if opts.ReportKeyboardEvents {
		vx.kittyFlags |= int(CSIuReportEvents)
	}

	if opts.EventQueueSize < 1 {
		opts.EventQueueSize = 1024
	}

	if opts.DisableMouse {
		vx.disableMouse = true
	}

	var tgts []*os.File
	switch opts.WithTTY {
	case "":
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			tgts = []*os.File{os.Stderr, os.Stdout, os.Stdin}
			break
		}
		tgts = []*os.File{f, os.Stderr, os.Stdout, os.Stdin}
	default:
		vx.withTty = opts.WithTTY
		f, err := os.OpenFile(opts.WithTTY, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		tgts = []*os.File{f}
	}

	vx.queue = make(chan Event, opts.EventQueueSize)
	vx.screenNext = newScreen()
	vx.screenLast = newScreen()
	vx.chClipboard = make(chan string)
	vx.chSigWinSz = make(chan os.Signal, 1)
	vx.chSigKill = make(chan os.Signal, 1)
	vx.chCursorPos = make(chan [2]int)
	vx.chQuit = make(chan bool)
	vx.chSizeDone = make(chan bool, 1)
	vx.charCache = make(map[string]int, 256)
	vx.chBg = make(chan string, 1)
	err = vx.openTty(tgts)
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
				vx.caps.sixels = true
				if vx.graphicsProtocol < sixelGraphics {
					vx.graphicsProtocol = sixelGraphics
				}
			case capabilityOsc11:
				vx.caps.osc11 = true
				log.Info("[capability] OSC 11 supported")
			case synchronizedUpdates:
				log.Info("[capability] Synchronized updates")
				vx.caps.synchronizedUpdate = true
			case unicodeCoreCap:
				log.Info("[capability] Unicode core")
				vx.caps.unicodeCore = true
			case notifyColorChange:
				log.Info("[capability] Color theme notifications")
				vx.caps.colorThemeUpdates = true
			case kittyKeyboard:
				log.Info("[capability] Kitty keyboard")
				if opts.DisableKittyKeyboard {
					continue
				}
				vx.caps.kittyKeyboard = true
			case styledUnderlines:
				vx.caps.styledUnderlines = true
				log.Info("[capability] Styled underlines")
			case truecolor:
				vx.caps.rgb = true
				log.Info("[capability] RGB")
			case kittyGraphics:
				log.Info("[capability] Kitty graphics supported")
				vx.caps.kittyGraphics = true
				if vx.graphicsProtocol < kitty {
					vx.graphicsProtocol = kitty
				}
			case textAreaPix:
				vx.caps.reportSizePixels = true
				log.Info("[capability] Report screen size: pixels")
			case textAreaChar:
				vx.caps.reportSizeChars = true
				log.Info("[capability] Report screen size: characters")
			case appID:
				vx.caps.osc176 = true
				vx.appIDLast = ev
				log.Info("[capability] OSC 176 supported")
			case terminalID:
				vx.termID = ev
			case inBandResizeEvents:
				vx.caps.inBandResize = true
			}
		}
	}

	vx.enterAltScreen()
	vx.enableModes()
	if !opts.NoSignals {
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
	vx.screenNext.resize(ws.Cols, ws.Rows)
	vx.screenLast.resize(ws.Cols, ws.Rows)
	vx.winSize = ws
	vx.PostEvent(vx.winSize)
	return vx, nil
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

	vx.Suspend()
	vx.console.Close()

	log.Info("Renders: %d", vx.renders)
	if vx.renders != 0 {
		log.Info("Time/render: %s", vx.elapsed/time.Duration(vx.renders))
	}
	log.Info("Cached characters: %d", len(vx.charCache))
}

// Resize manually triggers a resize event. Normally, vaxis listens to SIGWINCH
// for resize events, however in some use cases a manual resize trigger may be
// needed
func (vx *Vaxis) Resize() {
	atomicStore(&vx.resize, true)
	vx.PostEvent(Redraw{})
}

// Render renders the model's content to the terminal
func (vx *Vaxis) Render() {
	if atomicLoad(&vx.resize) {
		defer atomicStore(&vx.resize, false)
		ws, err := vx.reportWinsize()
		if err != nil {
			log.Error("couldn't report winsize: %v", err)
			return
		}
		if ws.Cols != vx.winSize.Cols || ws.Rows != vx.winSize.Rows {
			vx.screenNext.resize(ws.Cols, ws.Rows)
			vx.screenLast.resize(ws.Cols, ws.Rows)
			vx.winSize = ws
			vx.refresh = true
			vx.PostEvent(vx.winSize)
			return
		}
	}
	start := time.Now()
	// defer renderBuf.Reset()
	vx.render()
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
		reposition = true
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
		_, _ = vx.tw.WriteString(tparm(cup, p1.row+1, p1.col+1))
		p1.writeTo(vx.tw)
	}
	// Save this frame as the last frame
	vx.graphicsLast = vx.graphicsNext

	if vx.mouseShapeLast != vx.mouseShapeNext {
		_, _ = vx.tw.WriteString(tparm(mouseShape, vx.mouseShapeNext))
		vx.mouseShapeLast = vx.mouseShapeNext
	}
	for row := range vx.screenNext.buf {
		reposition = true
		for col := 0; col < len(vx.screenNext.buf[row]); col += 1 {
			next := vx.screenNext.buf[row][col]
			if next.sixel {
				vx.screenLast.buf[row][col].sixel = true
				reposition = true
				continue
			}
			if next == vx.screenLast.buf[row][col] && !vx.refresh {
				reposition = true
				// Advance the column by the width of this
				// character
				skip := vx.advance(next)
				// skip := advance(next.Content)
				for i := 1; i < skip+1; i += 1 {
					if col+i >= len(vx.screenNext.buf[row]) {
						break
					}
					// null out any cells we end up skipping
					vx.screenLast.buf[row][col+i] = Cell{}
				}
				col += skip
				continue
			}
			vx.screenLast.buf[row][col] = next
			if reposition {
				if cursor.Hyperlink != "" {
					_, _ = vx.tw.WriteString(tparm(osc8, "", ""))
				}
				_, _ = vx.tw.WriteString(tparm(cup, row+1, col+1))
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
						vx.tw.Printf(fgSet, ps[0])
					case ps[0] < 16:
						vx.tw.Printf(fgBrightSet, ps[0]-8)
					default:
						vx.tw.Printf(fgIndexSet, ps[0])
					}
				case 3:
					vx.tw.Printf(fgRGBSet, ps[0], ps[1], ps[2])
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
						vx.tw.Printf(bgSet, ps[0])
					case ps[0] < 16:
						vx.tw.Printf(bgBrightSet, ps[0]-8)
					default:
						vx.tw.Printf(bgIndexSet, ps[0])
					}
				case 3:
					vx.tw.Printf(bgRGBSet, ps[0], ps[1], ps[2])
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
						_, _ = vx.tw.Printf(ulIndexSet, ps[0])
					case 3:
						_, _ = vx.tw.Printf(ulRGBSet, ps[0], ps[1], ps[2])
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
			}

			if cursor.UnderlineStyle != next.UnderlineStyle {
				ulStyle := next.UnderlineStyle
				switch vx.caps.styledUnderlines {
				case true:
					_, _ = vx.tw.WriteString(tparm(ulStyleSet, ulStyle))
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
				_, _ = vx.tw.WriteString(tparm(osc8, linkPs, link))
			}

			cursor = next.Style

			if next.Width == 0 {
				next.Width = vx.characterWidth(next.Grapheme)
			}

			switch next.Width {
			case 0:
				_, _ = vx.tw.WriteString(" ")
			default:
				_, _ = vx.tw.WriteString(next.Grapheme)
			}
			skip := vx.advance(next)
			for i := 1; i < skip+1; i += 1 {
				if col+i >= len(vx.screenNext.buf[row]) {
					break
				}
				// null out any cells we end up skipping
				vx.screenLast.buf[row][col+i] = Cell{}
			}
			col += skip
		}
	}
	if cursor.Hyperlink != "" {
		_, _ = vx.tw.WriteString(tparm(osc8, "", ""))
	}
	if vx.cursorNext.visible && !vx.cursorLast.visible {
		_, _ = vx.tw.WriteString(vx.showCursor())
	}
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
		switch seq.Final {
		case 'c':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				for _, ps := range seq.Parameters {
					switch ps[0] {
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
			if atomicLoad(&vx.reqCursorPos) {
				atomicStore(&vx.reqCursorPos, false)
				if len(seq.Parameters) != 2 {
					log.Error("not enough DSRCPR params")
					return
				}
				vx.chCursorPos <- [2]int{
					seq.Parameters[0][0],
					seq.Parameters[1][0],
				}
				return
			}
		case 'S':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				if len(seq.Parameters) < 3 {
					break
				}
				switch seq.Parameters[0][0] {
				case 2:
					if seq.Parameters[1][0] == 0 {
						vx.PostEventBlocking(capabilitySixel{})
					}
				}
				return
			}
		case 'n':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				if len(seq.Parameters) != 2 {
					break
				}
				switch seq.Parameters[0][0] {
				case colorThemeResp: // 997
					m := ColorThemeMode(seq.Parameters[1][0])
					vx.PostEventBlocking(ColorThemeUpdate{
						Mode: m,
					})
				}
				return
			}
		case 'y':
			// DECRPM - DEC Report Mode
			if len(seq.Parameters) < 1 {
				log.Error("not enough DECRPM params")
				return
			}
			switch seq.Parameters[0][0] {
			case 2026:
				if len(seq.Parameters) < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					vx.PostEventBlocking(synchronizedUpdates{})
				}
			case 2027:
				if len(seq.Parameters) < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					vx.PostEventBlocking(unicodeCoreCap{})
				}
			case 2031:
				if len(seq.Parameters) < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					vx.PostEventBlocking(notifyColorChange{})
				}
			}
			return
		case 'u':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				vx.PostEventBlocking(kittyKeyboard{})
				return
			}
		case '~':
			if len(seq.Intermediate) == 0 {
				if len(seq.Parameters) == 0 {
					log.Error("[CSI] unknown sequence with final '~'")
					return
				}
				switch seq.Parameters[0][0] {
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
			mouse, ok := parseMouseEvent(seq)
			if ok {
				vx.PostEventBlocking(mouse)
			}
			return
		case 't':
			if len(seq.Parameters) < 3 {
				log.Error("[CSI] unknown sequence: %s", seq)
				return
			}
			// CSI <type> ; <height> ; <width> t
			typ := seq.Parameters[0][0]
			h := seq.Parameters[1][0]
			w := seq.Parameters[2][0]
			switch typ {
			case 4:
				vx.nextSize.XPixel = w
				vx.nextSize.YPixel = h
				if !vx.caps.reportSizePixels {
					// Gate on this so we only report this
					// once at startup
					vx.PostEventBlocking(textAreaPix{})
					return
				}
			case 8:
				vx.nextSize.Cols = w
				vx.nextSize.Rows = h
				if !vx.caps.reportSizeChars {
					// Gate on this so we only report this
					// once at startup. This also means we
					// can set the size directly and won't
					// have race conditions
					vx.PostEventBlocking(textAreaChar{})
					return
				}
				vx.chSizeDone <- true
			case 48:
				// CSI <type> ; <height> ; <width> ; <height_pix> ; <width_pix> t
				switch len(seq.Parameters) {
				case 5:
					atomicStore(&vx.resize, true)
					vx.nextSize.Cols = w
					vx.nextSize.Rows = h
					vx.nextSize.YPixel = seq.Parameters[3][0]
					vx.nextSize.XPixel = seq.Parameters[4][0]
					if !vx.caps.inBandResize {
						vx.PostEventBlocking(inBandResizeEvents{})
					}
					vx.Resize()
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
		switch seq.Final {
		case 'r':
			if len(seq.Intermediate) < 1 {
				return
			}
			switch seq.Intermediate[0] {
			case '+':
				// XTGETTCAP response
				if len(seq.Parameters) < 1 {
					return
				}
				if seq.Parameters[0] == 0 {
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
			}
		case '|':
			if len(seq.Intermediate) < 1 {
				return
			}
			switch seq.Intermediate[0] {
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
		if strings.HasPrefix(string(seq.Payload), "11") {
			// If we are here and don't know that the host terminal
			// supports the sequence, it means we are handling the response
			// to the initial query and thus nobody is expecting its actual
			// content. In this case, we don't want to fill the channel buffer
			// as no one will clear it.
			if vx.CanReportBackgroundColor() {
				vx.chBg <- string(seq.Payload)
			}
			vx.PostEventBlocking(capabilityOsc11{})
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

// QueryBackground queries the host terminal for background color and returns
// it as an instance of vaxis.Color. If the host terminal doesn't support this,
// Color(0) is returned instead. Make sure not to run this in the same
// goroutine as Vaxis runs in or deadlock will occur.
func (vx *Vaxis) QueryBackground() Color {
	if !vx.CanReportBackgroundColor() {
		return Color(0)
	}
	vx.tw.WriteString(osc11)
	resp := <-vx.chBg
	var r, g, b int
	_, err := fmt.Sscanf(resp, "11;rgb:%x/%x/%x", &r, &g, &b)
	if err != nil {
		log.Error("QueryBackground: failed to parse the OSC 11 response: %s", err)
		return Color(0)
	}
	// The returned value can in principle be 16 bits per channel, however
	// we are not aware of any terminal that would do this, foot for
	// instance just repeats the same 8 bits twice. Hence we only take the
	// lower 8 bits.
	return RGBColor(uint8(r), uint8(g), uint8(b))
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

	_, _ = vx.tw.WriteString(decrqm(synchronizedUpdate))
	_, _ = vx.tw.WriteString(decrqm(unicodeCore))
	_, _ = vx.tw.WriteString(decrqm(colorThemeUpdates))
	// We blindly enable in band resize. We get a response immediately if it
	// is supported
	_, _ = vx.tw.WriteString(decset(inBandResize))
	_, _ = vx.tw.WriteString(xtversion)
	_, _ = vx.tw.WriteString(kittyKBQuery)
	_, _ = vx.tw.WriteString(kittyGquery)
	_, _ = vx.tw.WriteString(xtsmSixelGeom)
	// Can the terminal report it's own size?
	_, _ = vx.tw.WriteString(textAreaSize)

	// Query some terminfo capabilities
	// Just another way to see if we have RGB support
	_, _ = vx.tw.WriteString(xtgettcap("RGB"))
	// Does the terminal respond to OSC 11 queries?
	_, _ = vx.tw.WriteString(osc11)
	// Back up the current app ID
	_, _ = vx.tw.WriteString(getAppID)
	// We request Smulx to check for styled underlines. Technically, Smulx
	// only means the terminal supports different underline types (curly,
	// dashed, etc), but we'll assume the terminal also suppports underline
	// colors (CSI 58 : ...)
	_, _ = vx.tw.WriteString(xtgettcap("Smulx"))
	// Need to send tertiary for VTE based terminals. These don't respond to
	// XTGETTCAP
	_, _ = vx.tw.WriteString(tertiaryAttributes)
	// Send Device Attributes is last. Everything responds, and when we get
	// a response we'll return from init
	_, _ = vx.tw.WriteString(primaryAttributes)
	_, _ = vx.tw.Flush()
}

// enableModes enables all the modes we want
func (vx *Vaxis) enableModes() {
	// kitty keyboard
	if vx.caps.kittyKeyboard {
		_, _ = vx.tw.WriteString(tparm(kittyKBEnable, vx.kittyFlags))
	}
	// sixel scrolling
	if vx.caps.sixels {
		_, _ = vx.tw.WriteString(decset(sixelScrolling))
	}
	// Mode 2027, unicode segmentation (for correct emoji/wc widths)
	if vx.caps.unicodeCore {
		_, _ = vx.tw.WriteString(decset(unicodeCore))
	}

	// Mode 2031: color scheme updates
	if vx.caps.colorThemeUpdates {
		_, _ = vx.tw.WriteString(decset(colorThemeUpdates))
		// Let's query the current mode also
		_, _ = vx.tw.WriteString(tparm(dsr, colorThemeReq))
	}
	if vx.caps.inBandResize {
		_, _ = vx.tw.WriteString(decset(inBandResize))
	}

	// TODO: query for bracketed paste support?
	_, _ = vx.tw.WriteString(decset(bracketedPaste)) // bracketed paste
	_, _ = vx.tw.WriteString(decset(cursorKeys))     // application cursor keys
	_, _ = vx.tw.WriteString(applicationMode)        // application cursor keys mode

	// TODO: Query for mouse modes or just hope for the best? In the
	// meantime, we enable button events, then all events. Terminals which
	// support both will enable the latter. Terminals which support only the
	// first will enable button events, then ignore the all events mode.
	if !vx.disableMouse {
		_, _ = vx.tw.WriteString(decset(mouseButtonEvents))
		_, _ = vx.tw.WriteString(decset(mouseAllEvents))
		_, _ = vx.tw.WriteString(decset(mouseFocusEvents))
		_, _ = vx.tw.WriteString(decset(mouseSGR))
	}
	_, _ = vx.tw.Flush()
}

func (vx *Vaxis) disableModes() {
	_, _ = vx.tw.WriteString(sgrReset)               // reset fg, bg, attrs
	_, _ = vx.tw.WriteString(decrst(bracketedPaste)) // bracketed paste
	if vx.caps.kittyKeyboard {
		_, _ = vx.tw.WriteString(kittyKBPop) // kitty keyboard
	}
	_, _ = vx.tw.WriteString(decrst(cursorKeys))
	_, _ = vx.tw.WriteString(numericMode)
	if !vx.disableMouse {
		_, _ = vx.tw.WriteString(decrst(mouseButtonEvents))
		_, _ = vx.tw.WriteString(decrst(mouseAllEvents))
		_, _ = vx.tw.WriteString(decrst(mouseFocusEvents))
		_, _ = vx.tw.WriteString(decrst(mouseSGR))
	}
	if vx.caps.sixels {
		_, _ = vx.tw.WriteString(decrst(sixelScrolling))
	}
	if vx.caps.unicodeCore {
		_, _ = vx.tw.WriteString(decrst(unicodeCore))
	}
	if vx.caps.colorThemeUpdates {
		_, _ = vx.tw.WriteString(decrst(colorThemeUpdates))
	}
	if vx.caps.osc176 {
		_, _ = vx.tw.WriteString(tparm(setAppID, vx.appIDLast))
	}
	if vx.caps.inBandResize {
		_, _ = vx.tw.WriteString(decrst(inBandResize))
	}
	// Most terminals default to "text" mouse shape
	_, _ = vx.tw.WriteString(tparm(mouseShape, MouseShapeTextInput))
	_, _ = vx.tw.Flush()
}

func (vx *Vaxis) enterAltScreen() {
	vx.tw.vx.refresh = true
	_, _ = vx.tw.WriteString(decset(alternateScreen))
	_, _ = vx.tw.WriteString(decrst(cursorVisibility))
	_, _ = vx.tw.Flush()
}

func (vx *Vaxis) exitAltScreen() {
	vx.HideCursor()
	_, _ = vx.tw.WriteString(decset(cursorVisibility))
	_, _ = vx.tw.WriteString(decrst(alternateScreen))
	_, _ = vx.tw.Flush()
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
	io.WriteString(vx.console, primaryAttributes)
	vx.parser.WaitClose()

	vx.disableModes()
	vx.exitAltScreen()
	signal.Stop(vx.chSigKill)
	signal.Stop(vx.chSigWinSz)
	vx.console.Reset()
	return nil
}

// openTty opens the /dev/tty device, makes it raw, and starts an input parser
func (vx *Vaxis) openTty(tgts []*os.File) error {
	for _, s := range tgts {
		if c, err := console.ConsoleFromFile(s); err == nil {
			vx.console = c
			break
		}
	}
	if vx.console == nil {
		return console.ErrNotAConsole
	}
	err := vx.console.SetRaw()
	if err != nil {
		return err
	}
	vx.tw = newWriter(vx)
	vx.parser = ansi.NewParser(vx.console)
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
					vx.parser.Finish(seq)
				}
			case <-vx.chSigWinSz:
				atomicStore(&vx.resize, true)
				vx.PostEventBlocking(Redraw{})
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
	tgts := []*os.File{os.Stderr, os.Stdout, os.Stdin}
	if vx.withTty != "" {
		f, err := os.OpenFile(vx.withTty, os.O_RDWR, 0)
		if err != nil {
			return err
		}
		tgts = []*os.File{f}
	}
	err := vx.openTty(tgts)
	if err != nil {
		return err
	}
	vx.enterAltScreen()
	vx.enableModes()
	vx.setupSignals()
	atomicStore(&vx.resize, true)
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
	buf.WriteString(decset(cursorVisibility))
	return buf.String()
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func (vx *Vaxis) CursorPosition() (row int, col int) {
	// DSRCPR - reports cursor position
	atomicStore(&vx.reqCursorPos, true)
	_, _ = io.WriteString(vx.console, dsrcpr)
	timeout := time.NewTimer(50 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warn("CursorPosition timed out")
		atomicStore(&vx.reqCursorPos, false)
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
	if vx.cursorNext.style == CursorDefault {
		// Cursor block is the default
		return tparm(cursorStyleSet, int(CursorBlock))
	}
	return tparm(cursorStyleSet, int(vx.cursorNext.style))
}

// ClipboardPush copies the provided string to the system clipboard
func (vx *Vaxis) ClipboardPush(s string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	_, _ = io.WriteString(vx.console, tparm(osc52put, b64))
}

// ClipboardPop requests the content from the system clipboard. ClipboardPop works by
// requesting the data from the underlying terminal, which responds back with
// the data. Depending on usage, this could take some time. Callers can provide
// a context to set a deadline for this function to return. An error will be
// returned if the context is cancelled.
func (vx *Vaxis) ClipboardPop(ctx context.Context) (string, error) {
	_, _ = io.WriteString(vx.console, osc52pop)
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
		_, _ = io.WriteString(vx.console, tparm(osc9notify, body))
		return
	}
	_, _ = io.WriteString(vx.console, tparm(osc777notify, title, body))
}

// SetTitle sets the terminal's title via OSC 2
func (vx *Vaxis) SetTitle(s string) {
	_, _ = io.WriteString(vx.console, tparm(setTitle, s))
}

// SetAppID sets the terminal's application ID via OSC 176
func (vx *Vaxis) SetAppID(s string) {
	_, _ = io.WriteString(vx.console, tparm(setAppID, s))
}

// Bell sends a BEL control signal to the terminal
func (vx *Vaxis) Bell() {
	_, _ = vx.console.Write([]byte{0x07})
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
	if vx.caps.unicodeCore {
		return gwidth(s, unicodeStd)
	}
	if vx.caps.noZWJ {
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

func (vx *Vaxis) CanKittyGraphics() bool {
	return vx.caps.kittyGraphics
}

func (vx *Vaxis) CanSixel() bool {
	return vx.caps.sixels
}

func (vx *Vaxis) CanReportBackgroundColor() bool {
	return vx.caps.osc11
}

func (vx *Vaxis) CanDisplayGraphics() bool {
	return vx.caps.sixels || vx.caps.kittyGraphics
}

func (vx *Vaxis) CanSetAppID() bool {
	return vx.caps.osc176
}

func (vx *Vaxis) nextGraphicID() uint64 {
	vx.graphicsIDNext += 1
	return vx.graphicsIDNext
}

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
