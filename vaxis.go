// Package vaxis is a terminal user interface for modern terminals
package vaxis

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

var log = slog.New(slog.NewTextHandler(io.Discard, nil))

type capabilities struct {
	synchronizedUpdate bool
	unicodeCore        bool
	rgb                bool
	kittyGraphics      bool
	kittyKeyboard      atomic.Bool
	styledUnderlines   bool
	sixels             bool
	colorThemeUpdates  bool
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
	// Logger is an optional slog.Logger that vaxis will log to. vaxis uses
	// stdlib levels for logging
	Logger *slog.Logger
	// DisableKittyKeyboard disables the use of the Kitty Keyboard protocol.
	// By default, if support is detected the protocol will be used.
	DisableKittyKeyboard bool
	// ReportKeyboardEvents will report key release and key repeat events if
	// KittyKeyboardProtocol is enabled and supported by the terminal
	ReportKeyboardEvents bool
}

type Vaxis struct {
	queue            *Queue[Event]
	tty              *os.File
	state            *term.State
	tw               *writer
	screenNext       *screen
	screenLast       *screen
	graphicsNext     map[int]*placement
	graphicsLast     map[int]*placement
	mouseShapeNext   MouseShape
	mouseShapeLast   MouseShape
	pastePending     bool
	chClipboard      chan string
	chSignal         chan os.Signal
	chCursorPos      chan [2]int
	chQuit           chan bool
	winSize          Resize
	caps             capabilities
	graphicsProtocol int
	graphicsIDNext   uint64
	reqCursorPos     atomic.Bool
	charCache        map[string]int
	cursorNext       cursorState
	cursorLast       cursorState
	closed           bool
	refresh          bool
	kittyFlags       int

	renders int
	elapsed time.Duration

	mu sync.Mutex
}

// New creates a new [Vaxis] instance. Calling New will query the underlying
// terminal for supported features and enter the alternate screen
func New(opts Options) (*Vaxis, error) {
	if opts.Logger != nil {
		log = opts.Logger
	}

	// Disambiguate, report alternate keys, report all keys as escapes, report associated text

	// Let's give some deadline for our queries responding. If they don't,
	// it means the terminal doesn't respond to Primary Device Attributes
	// and that is a problem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	vx := &Vaxis{
		kittyFlags: 29,
	}

	if opts.ReportKeyboardEvents {
		vx.kittyFlags += 2
	}

	vx.queue = NewQueue[Event]()
	vx.screenNext = newScreen()
	vx.screenLast = newScreen()
	vx.graphicsNext = make(map[int]*placement)
	vx.graphicsLast = make(map[int]*placement)
	vx.chClipboard = make(chan string)
	vx.chSignal = make(chan os.Signal, 1)
	vx.chCursorPos = make(chan [2]int)
	vx.chQuit = make(chan bool)
	vx.charCache = make(map[string]int, 256)
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
		case ev := <-vx.queue.Chan():
			switch ev.(type) {
			case primaryDeviceAttribute:
				break outer
			case capabilitySixel:
				log.Info("Capability: Sixel graphics")
				vx.caps.sixels = true
				if vx.graphicsProtocol < sixelGraphics {
					vx.graphicsProtocol = sixelGraphics
				}
			case synchronizedUpdates:
				log.Info("Capability: Synchronized updates")
				vx.caps.synchronizedUpdate = true
			case unicodeCoreCap:
				log.Info("Capability: Unicode core")
				vx.caps.unicodeCore = true
			case notifyColorChange:
				log.Info("Capability: Color theme notifications")
				vx.caps.colorThemeUpdates = true
			case kittyKeyboard:
				log.Info("Capability: Kitty keyboard")
				if opts.DisableKittyKeyboard {
					continue
				}
				vx.caps.kittyKeyboard.Store(true)
			case styledUnderlines:
				vx.caps.styledUnderlines = true
				log.Info("Capability: Styled underlines")
			case truecolor:
				vx.caps.rgb = true
				log.Info("Capability: RGB")
			case kittyGraphics:
				log.Info("Capability: Kitty graphics supported")
				vx.caps.kittyGraphics = true
				if vx.graphicsProtocol < kitty {
					vx.graphicsProtocol = kitty
				}
			}
		}
	}

	vx.enterAltScreen()
	vx.enableModes()
	signal.Notify(vx.chSignal,
		syscall.SIGWINCH,
		// kill signals
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGTERM,
	)
	vx.winSize, err = vx.reportWinsize()
	if err != nil {
		return nil, err
	}
	vx.PostEvent(vx.winSize)
	return vx, nil
}

// PostEvent inserts an event into the [Vaxis] event loop
func (vx *Vaxis) PostEvent(ev Event) {
	vx.queue.Push(ev)
}

// SyncFunc queues a function to be called from the main thread. vaxis will call
// the function when the event is received in the main thread either through
// PollEvent or Events. A Redraw event will be sent to the host application
// after the function is completed
func (vx *Vaxis) SyncFunc(fn func()) {
	vx.PostEvent(syncFunc(fn))
}

// PollEvent blocks until there is an Event, and returns that Event
func (vx *Vaxis) PollEvent() Event {
	for {
		select {
		case ev := <-vx.queue.Chan():
			switch e := ev.(type) {
			case Resize:
				vx.mu.Lock()
				vx.screenNext.resize(e.Cols, e.Rows)
				vx.screenLast.resize(e.Cols, e.Rows)
				vx.mu.Unlock()
			case syncFunc:
				e()
				ev = Redraw{}
			}
			return ev
		case <-vx.chQuit:
			return QuitEvent{}
		}
	}
}

// Events returns a channel of events. This should only be called once: it will
// create a channel for you to listen on. Multiple calls will kick off multiple
// goroutines.
//
// Good use:
//
//	for ev := range vx.Events() {
//		// do something
//	}
func (vx *Vaxis) Events() chan Event {
	ch := make(chan Event)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				vx.Close()
			}
		}()
		for {
			ev := vx.PollEvent()
			switch ev.(type) {
			case QuitEvent:
				close(ch)
				return
			default:
				ch <- ev
			}
		}
	}()
	return ch
}

// Close shuts down the event loops and returns the terminal to it's original
// state
func (vx *Vaxis) Close() {
	if vx.closed {
		return
	}
	vx.closed = true
	defer close(vx.chQuit)

	vx.Suspend()

	log.Info("Renders", "val", vx.renders)
	if vx.renders != 0 {
		log.Info("Time/render", "val", vx.elapsed/time.Duration(vx.renders))
	}
	log.Info("Cached characters", "val", len(vx.charCache))
}

// Render renders the model's content to the terminal
func (vx *Vaxis) Render() {
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
	// Delete any placements we don't have this round
	for id, p := range vx.graphicsLast {
		if _, ok := vx.graphicsNext[id]; ok && !vx.refresh {
			continue
		}
		_, _ = vx.tw.WriteString(p.delete())
		delete(vx.graphicsLast, id)
	}
	// draw new placements
	for id, p := range vx.graphicsNext {
		p.lockRegion()
		if _, ok := vx.graphicsLast[id]; ok {
			continue
		}
		_, _ = vx.tw.WriteString(tparm(cup, p.row+1, p.col+1))
		_, _ = vx.tw.WriteString(p.draw())
		vx.graphicsLast[id] = p
	}
	if vx.mouseShapeLast != vx.mouseShapeNext {
		_, _ = vx.tw.WriteString(tparm(mouseShape, vx.mouseShapeNext))
		vx.mouseShapeLast = vx.mouseShapeNext
	}
	for row := range vx.screenNext.buf {
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
				ps := fg.params()
				if !vx.caps.rgb {
					ps = fg.asIndex().params()
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
				ps := bg.params()
				if !vx.caps.rgb {
					ps = bg.asIndex().params()
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
					ps := ul.params()
					if !vx.caps.rgb {
						ps = ul.asIndex().params()
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
	if vx.cursorNext.visible && !vx.cursorLast.visible {
		_, _ = vx.tw.WriteString(vx.showCursor())
	}
}

func (vx *Vaxis) handleSequence(seq ansi.Sequence) {
	log.Debug("[stdin]", "sequence", seq)
	switch seq := seq.(type) {
	case ansi.Print:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEvent(key)
	case ansi.C0:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEvent(key)
	case ansi.ESC:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEvent(key)
	case ansi.SS3:
		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEvent(key)
	case ansi.CSI:
		switch seq.Final {
		case 'c':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				for _, ps := range seq.Parameters {
					switch ps[0] {
					case 4:
						vx.PostEvent(capabilitySixel{})
					}
				}
				vx.PostEvent(primaryDeviceAttribute{})
				return
			}
		case 'I':
			vx.PostEvent(FocusIn{})
			return
		case 'O':
			vx.PostEvent(FocusOut{})
			return
		case 'R':
			// KeyF1 or DSRCPR
			// This could be an F1 key, we need to buffer if we have
			// requested a DSRCPR (cursor position report)
			//
			// Kitty keyboard protocol disambiguates this scenario,
			// hopefully people are using that
			if vx.reqCursorPos.Swap(false) {
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
						vx.PostEvent(capabilitySixel{})
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
					vx.PostEvent(ColorThemeUpdate{
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
					vx.PostEvent(synchronizedUpdates{})
				}
			case 2027:
				if len(seq.Parameters) < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					vx.PostEvent(unicodeCoreCap{})
				}
			case 2031:
				if len(seq.Parameters) < 2 {
					log.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					vx.PostEvent(notifyColorChange{})
				}
			}
			return
		case 'u':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				vx.PostEvent(kittyKeyboard{})
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
					vx.PostEvent(PasteStartEvent{})
					return
				case 201:
					vx.pastePending = false
					vx.PostEvent(PasteEndEvent{})
					return
				}
			}
		case 'M', 'm':
			mouse, ok := parseMouseEvent(seq)
			if ok {
				vx.PostEvent(mouse)
			}
			return
		}

		key := decodeKey(seq)
		if vx.pastePending {
			key.EventType = EventPaste
		}
		vx.PostEvent(key)
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
					log.Error("error parsing XTGETTCAP", "value", string(seq.Data))
				}
				switch vals[0] {
				case hexEncode("Smulx"):
					vx.PostEvent(styledUnderlines{})
				case hexEncode("RGB"):
					vx.PostEvent(truecolor{})
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
					vx.PostEvent(styledUnderlines{})
				}
			}
		}
	case ansi.APC:
		if len(seq.Data) == 0 {
			return
		}
		if strings.HasPrefix(seq.Data, "G") {
			vx.PostEvent(kittyGraphics{})
		}
	case ansi.OSC:
		if strings.HasPrefix(string(seq.Payload), "52") {
			vals := strings.Split(string(seq.Payload), ";")
			if len(vals) != 3 {
				log.Error("invalid OSC 52 payload")
				return
			}
			b, err := base64.StdEncoding.DecodeString(vals[2])
			if err != nil {
				log.Error("couldn't decode OSC 52", "error", err)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			select {
			case vx.chClipboard <- string(b):
			case <-ctx.Done():
			}
		}
	}
}

func (vx *Vaxis) sendQueries() {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		vx.PostEvent(truecolor{})
	}

	_, _ = vx.tw.WriteString(decrqm(synchronizedUpdate))
	_, _ = vx.tw.WriteString(decrqm(unicodeCore))
	_, _ = vx.tw.WriteString(decrqm(colorThemeUpdates))
	_, _ = vx.tw.WriteString(xtversion)
	_, _ = vx.tw.WriteString(kittyKBQuery)
	_, _ = vx.tw.WriteString(kittyGquery)
	_, _ = vx.tw.WriteString(xtsmSixelGeom)

	// Query some terminfo capabilities
	// Just another way to see if we have RGB support
	_, _ = vx.tw.WriteString(xtgettcap("RGB"))
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
	if vx.caps.kittyKeyboard.Load() {
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

	// TODO: query for bracketed paste support?
	_, _ = vx.tw.WriteString(decset(bracketedPaste)) // bracketed paste
	_, _ = vx.tw.WriteString(decset(cursorKeys))     // application cursor keys
	_, _ = vx.tw.WriteString(applicationMode)        // application cursor keys mode
	// TODO: Query for mouse modes or just hope for the best?
	_, _ = vx.tw.WriteString(decset(mouseAllEvents))
	_, _ = vx.tw.WriteString(decset(mouseFocusEvents))
	_, _ = vx.tw.WriteString(decset(mouseSGR))
	_, _ = vx.tw.Flush()
}

func (vx *Vaxis) disableModes() {
	_, _ = vx.tw.WriteString(sgrReset)               // reset fg, bg, attrs
	_, _ = vx.tw.WriteString(decrst(bracketedPaste)) // bracketed paste
	if vx.caps.kittyKeyboard.Load() {
		_, _ = vx.tw.WriteString(kittyKBPop) // kitty keyboard
	}
	_, _ = vx.tw.WriteString(decrst(cursorKeys))
	_, _ = vx.tw.WriteString(numericMode)
	_, _ = vx.tw.WriteString(decrst(mouseAllEvents))
	_, _ = vx.tw.WriteString(decrst(mouseFocusEvents))
	_, _ = vx.tw.WriteString(decrst(mouseSGR))
	if vx.caps.sixels {
		_, _ = vx.tw.WriteString(decrst(sixelScrolling))
	}
	if vx.caps.unicodeCore {
		_, _ = vx.tw.WriteString(decrst(unicodeCore))
	}
	if vx.caps.colorThemeUpdates {
		_, _ = vx.tw.WriteString(decrst(colorThemeUpdates))
	}
	_, _ = vx.tw.WriteString(tparm(mouseShape, MouseShapeDefault))
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
	signal.Stop(vx.chSignal)
	vx.disableModes()
	vx.exitAltScreen()
	_ = term.Restore(int(vx.tty.Fd()), vx.state)
	vx.tty.Close()
	return nil
}

// makeRaw opens the /dev/tty device, makes it raw, and starts an input parser
func (vx *Vaxis) openTty() error {
	var err error
	vx.tty, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return err
	}
	vx.tw = newWriter(vx)
	vx.state, err = term.MakeRaw(int(vx.tty.Fd()))
	if err != nil {
		return err
	}
	parser := ansi.NewParser(vx.tty)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				vx.Close()
			}
		}()
		for {
			select {
			case seq := <-parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					return
				default:
					vx.handleSequence(seq)
				}
			case sig := <-vx.chSignal:
				switch sig {
				case syscall.SIGWINCH, syscall.SIGCONT:
					vx.winSize, err = vx.reportWinsize()
					if err != nil {
						log.Error("reporting window size",
							"error", err)
					}
					vx.PostEvent(vx.winSize)
				default:
					// Anything else is trying to kill the
					// process
					vx.Close()
					return
				}
			case <-vx.chQuit:
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
	vx.enterAltScreen()
	vx.enableModes()
	signal.Notify(vx.chSignal,
		syscall.SIGWINCH,
		// kill signals
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGTERM,
	)
	vx.winSize, err = vx.reportWinsize()
	if err != nil {
		return err
	}
	vx.PostEvent(vx.winSize)
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
func (vx *Vaxis) CursorPosition() (col int, row int) {
	// DSRCPR - reports cursor position
	vx.reqCursorPos.Store(true)
	_, _ = vx.tty.WriteString(dsrcpr)
	timeout := time.NewTimer(50 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warn("CursorPosition timed out")
		vx.reqCursorPos.Store(false)
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
		return cursorStyleReset
	}
	return tparm(cursorStyleSet, int(vx.cursorNext.style))
}

// ClipboardPush copies the provided string to the system clipboard
func (vx *Vaxis) ClipboardPush(s string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	_, _ = vx.tty.WriteString(tparm(osc52put, b64))
}

// ClipboardPop requests the content from the system clipboard. ClipboardPop works by
// requesting the data from the underlying terminal, which responds back with
// the data. Depending on usage, this could take some time. Callers can provide
// a context to set a deadline for this function to return. An error will be
// returned if the context is cancelled.
func (vx *Vaxis) ClipboardPop(ctx context.Context) (string, error) {
	_, _ = vx.tty.WriteString(osc52pop)
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
		_, _ = vx.tty.WriteString(tparm(osc9notify, body))
		return
	}
	_, _ = vx.tty.WriteString(tparm(osc777notify, title, body))
}

// SetTitle sets the terminal's title via OSC 2
func (vx *Vaxis) SetTitle(s string) {
	_, _ = vx.tty.WriteString(tparm(setTitle, s))
}

// Bell sends a BEL control signal to the terminal
func (vx *Vaxis) Bell() {
	_, _ = vx.tty.WriteString("\a")
}

// advance returns the extra amount to advance the column by when rendering
func (vx *Vaxis) advance(cell Cell) int {
	if cell.Width == 0 {
		cell.Width = vx.characterWidth(cell.Grapheme)
	}
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
		return uniseg.StringWidth(s)
	}
	// Why runewidth here? uniseg differs from wcwidth a bit but is
	// more accurate for terminals which support unicode. We use
	// uniseg there, and runewidth here
	return runewidth.StringWidth(s)
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

// reportWinsize
func (vx *Vaxis) reportWinsize() (Resize, error) {
	ws, err := unix.IoctlGetWinsize(int(vx.tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}, nil
}

func (vx *Vaxis) CanKittyGraphics() bool {
	return vx.caps.kittyGraphics
}

func (vx *Vaxis) CanSixel() bool {
	return vx.caps.sixels
}

func (vx *Vaxis) CanDisplayGraphics() bool {
	return vx.caps.sixels || vx.caps.kittyGraphics
}
