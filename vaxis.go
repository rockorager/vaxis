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
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

var (
	log = slog.New(slog.NewTextHandler(io.Discard, nil))

	// msgs is the main event loop Msg queue
	msgs *queue[Msg]
	// chQuit is a channel to signal to running goroutines that we are
	// quitting
	chQuit chan struct{}
	// inPaste signals that we are within a bracketed paste
	inPaste    bool
	osc52Paste chan string
	// pasteBuf buffers bracketed paste text
	pasteBuf *bytes.Buffer
	// Have we requested a cursor position?
	cursorPositionRequested  bool
	chCursorPositionReport   chan int
	deviceAttributesReceived chan struct{}
	initialized              bool
	// Disambiguate, report all keys as escapes, report associated text
	kittyKBFlags = 25

	// Rendering variables
	// w is our buffered, synchronized-update writer
	w *writer
	// refresh signals if we should do a delta render or full render
	refresh bool
	// stdScreen is the stdScreen Surface
	stdScreen *screen
	// lastRender stores the state of our last render. This is used to
	// optimize what we update on the next render
	lastRender *screen

	// stdout is the terminal we are talking with
	stdout *os.File
	stdin  *os.File
	state  *term.State

	capabilities struct {
		synchronizedUpdate bool
		rgb                bool
		kittyGraphics      bool
		kittyKeyboard      bool
		styledUnderlines   bool
		sixels             bool
		// unicode refers to rendering complex unicode graphemes
		// properly.
		unicode bool
	}
	winsize Resize

	lastGraphicPlacements map[int]*placement
	nextGraphicPlacements map[int]*placement
	lastMouseShape        MouseShape
	nextMouseShape        MouseShape

	nextCursor cursorState
	lastCursor cursorState
	// Statistics
	renders int
	elapsed time.Duration

	framerate uint
)

type cursorState struct {
	row     int
	col     int
	style   CursorStyle
	visible bool
}

// Converts a string into a slice of Characters suitable to assign to terminal cells
func Characters(s string) []string {
	egcs := []string{}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		egcs = append(egcs, g.Str())
	}
	return egcs
}

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
	// Framerate is the framerate (in frames per second) to render at in the
	// event loop. Default is 120 FPS. If using the PollMsg for a custom
	// event loop, this value is unused
	Framerate uint
}

// Init sets up all internal data structures, queries the terminal for feature
// support, and enters the alternate screen.
//
// Terminals *must* respond to Primary Device Attributes queries for this
// function to return error free. A timeout is implemented, and if a terminal
// does not respond in less than 3 seconds an error will be returned.
func Init(opts Options) error {
	// Let's give some deadline for our queries responding. If they don't,
	// it means the terminal doesn't respond to Primary Device Attributes
	// and that is a problem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stdout = os.Stdout
	stdin = os.Stdin
	var err error
	state, err = term.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	parser := ansi.NewParser(stdin)
	if opts.Logger != nil {
		log = opts.Logger
	}
	if opts.ReportKeyboardEvents {
		kittyKBFlags += 2
	}
	switch opts.Framerate {
	case 0:
		framerate = 120
	default:
		framerate = opts.Framerate
	}

	// Rendering
	w = newWriter()
	lastRender = newScreen()
	stdScreen = newScreen()

	// pasteBuf buffers bracketed paste
	pasteBuf = &bytes.Buffer{}
	osc52Paste = make(chan string)

	nextGraphicPlacements = make(map[int]*placement)
	lastGraphicPlacements = make(map[int]*placement)

	// Setup internals and signal handling
	msgs = newQueue[Msg]()
	chQuit = make(chan struct{})
	chCursorPositionReport = make(chan int)
	PostMsg(InitMsg{})

	chSIGWINCH := make(chan os.Signal, 1)
	reportWinsize()
	stdScreen.resize(winsize.Cols, winsize.Rows)
	lastRender.resize(winsize.Cols, winsize.Rows)
	deviceAttributesReceived = make(chan struct{})

	go func() {
		for {
			select {
			case seq := <-parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					return
				default:
					handleSequence(seq)
				}
			case <-chSIGWINCH:
				reportWinsize()
			case <-chQuit:
				return
			}
		}
	}()
	signal.Notify(chSIGWINCH, syscall.SIGWINCH)
	sendQueries()
	select {
	case <-ctx.Done():
		close(deviceAttributesReceived)
		initialized = true
		return context.Canceled
	case <-deviceAttributesReceived:
		close(deviceAttributesReceived)
		initialized = true
	}

	// Disable features based on options. We've already received all of our
	// queries so this still has effect
	if opts.DisableKittyKeyboard {
		capabilities.kittyKeyboard = false
	}
	return nil
}

// Run operates an event loop for the provided Model. Users of the Run loop
// don't need to explicitly render, the loop will render every event
func Run(model Model) error {
	dur := time.Duration((float64(1) / float64(framerate)) * float64(time.Second))
	tick := time.NewTicker(dur)
	updated := false
	for {
		select {
		case <-chQuit:
			return nil
		case <-tick.C:
			if !updated {
				continue
			}
			model.Draw(Window{})
			Render()
			updated = false
		case msg := <-msgs.ch:
			updated = true
			switch msg := msg.(type) {
			case Resize:
				stdScreen.resize(msg.Cols, msg.Rows)
				lastRender.resize(msg.Cols, msg.Rows)
			case SendMsg:
				msg.Model.Update(msg.Msg)
			case FuncMsg:
				msg.Func()
			case DrawModelMsg:
				// msg.Model.Draw(msg.Window)
			default:
				model.Update(msg)
			}
		}
	}
}

// Close shuts down the event loops and returns the terminal to it's original
// state
func Close() {
	close(chQuit)
	_, _ = w.WriteString(decset(cursorVisibility)) // show the cursor
	_, _ = w.WriteString(sgrReset)                 // reset fg, bg, attrs
	_, _ = w.WriteString(clear)

	// Disable any modes we enabled
	_, _ = w.WriteString(decrst(bracketedPaste)) // bracketed paste
	_, _ = w.WriteString(kittyKBPop)             // kitty keyboard
	_, _ = w.WriteString(decrst(cursorKeys))
	_, _ = w.WriteString(numericMode)
	_, _ = w.WriteString(decrst(mouseAllEvents))
	_, _ = w.WriteString(decrst(mouseFocusEvents))
	_, _ = w.WriteString(decrst(mouseSGR))
	_, _ = w.WriteString(decrst(sixelScrolling))
	_, _ = w.WriteString(tparm(mouseShape, MouseShapeDefault))

	_, _ = w.WriteString(decrst(alternateScreen))
	_, _ = w.Flush()

	_ = term.Restore(int(os.Stdout.Fd()), state)

	log.Info("Renders", "val", renders)
	if renders != 0 {
		log.Info("Time/render", "val", elapsed/time.Duration(renders))
	}
}

// Render renders the model's content to the terminal
func Render() {
	start := time.Now()
	// defer renderBuf.Reset()
	render()
	n, _ := w.Flush()
	log.Debug("flushed", "bytes", n)
	// updating cursor state has to be after Flush, we check state change in
	// flush.
	lastCursor = nextCursor
	elapsed += time.Since(start)
	renders += 1
	refresh = false
}

// Refresh forces a full render of the entire screen. Traditionally, this should
// be bound to Ctrl+l
func Refresh() {
	refresh = true
	Render()
}

func render() {
	var (
		reposition = true
		fg         Color
		bg         Color
		ul         Color
		ulStyle    UnderlineStyle
		attr       AttributeMask
		link       string
		linkID     string
	)
	// Delete any placements we don't have this round
	for id, p := range lastGraphicPlacements {
		if _, ok := nextGraphicPlacements[id]; ok && !refresh {
			continue
		}
		_, _ = w.WriteString(p.delete())
		delete(lastGraphicPlacements, id)
	}
	// draw new placements
	for id, p := range nextGraphicPlacements {
		p.lockRegion()
		if _, ok := lastGraphicPlacements[id]; ok {
			continue
		}
		_, _ = w.WriteString(tparm(cup, p.row+1, p.col+1))
		_, _ = w.WriteString(p.draw())
		lastGraphicPlacements[id] = p
	}
	if lastMouseShape != nextMouseShape {
		_, _ = w.WriteString(tparm(mouseShape, nextMouseShape))
		lastMouseShape = nextMouseShape
	}
	for row := range stdScreen.buf {
		for col := 0; col < len(stdScreen.buf[row]); col += 1 {
			next := stdScreen.buf[row][col]
			if next.sixel {
				lastRender.buf[row][col].sixel = true
				reposition = true
				continue
			}
			if next == lastRender.buf[row][col] && !refresh {
				reposition = true
				// Advance the column by the width of this
				// character
				skip := advance(next.Character)
				for i := 1; i < skip+1; i += 1 {
					if col+i >= len(stdScreen.buf[row]) {
						break
					}
					// null out any cells we end up skipping
					lastRender.buf[row][col+i] = Cell{}
				}
				col += skip
				continue
			}
			lastRender.buf[row][col] = next
			if reposition {
				_, _ = w.WriteString(tparm(cup, row+1, col+1))
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

			if fg != next.Foreground {
				fg = next.Foreground
				ps := fg.Params()
				if !capabilities.rgb {
					ps = fg.AsIndex().Params()
				}
				switch len(ps) {
				case 0:
					_, _ = w.WriteString(fgReset)
				case 1:
					switch {
					case ps[0] < 8:
						w.Printf(fgSet, ps[0])
					case ps[0] < 16:
						w.Printf(fgBrightSet, ps[0]-8)
					default:
						w.Printf(fgIndexSet, ps[0])
					}
				case 3:
					w.Printf(fgRGBSet, ps[0], ps[1], ps[2])
				}
			}

			if bg != next.Background {
				bg = next.Background
				ps := bg.Params()
				if !capabilities.rgb {
					ps = bg.AsIndex().Params()
				}
				switch len(ps) {
				case 0:
					_, _ = w.WriteString(bgReset)
				case 1:
					switch {
					case ps[0] < 8:
						w.Printf(bgSet, ps[0])
					case ps[0] < 16:
						w.Printf(bgBrightSet, ps[0]-8)
					default:
						w.Printf(bgIndexSet, ps[0])
					}
				case 3:
					w.Printf(bgRGBSet, ps[0], ps[1], ps[2])
				}
			}

			if capabilities.styledUnderlines {
				if ul != next.UnderlineColor {
					ul = next.UnderlineColor
					ps := ul.Params()
					if !capabilities.rgb {
						ps = ul.AsIndex().Params()
					}
					switch len(ps) {
					case 0:
						_, _ = w.WriteString(ulColorReset)
					case 1:
						_, _ = w.Printf(ulIndexSet, ps[0])
					case 3:
						_, _ = w.Printf(ulRGBSet, ps[0], ps[1], ps[2])
					}
				}
			}

			if attr != next.Attribute {
				// find the ones that have changed
				dAttr := attr ^ next.Attribute
				// If the bit is changed and in next, it was
				// turned on
				on := dAttr & next.Attribute

				if on&AttrBold != 0 {
					_, _ = w.WriteString(boldSet)
				}
				if on&AttrDim != 0 {
					_, _ = w.WriteString(dimSet)
				}
				if on&AttrItalic != 0 {
					_, _ = w.WriteString(italicSet)
				}
				if on&AttrBlink != 0 {
					_, _ = w.WriteString(blinkSet)
				}
				if on&AttrReverse != 0 {
					_, _ = w.WriteString(reverseSet)
				}
				if on&AttrInvisible != 0 {
					_, _ = w.WriteString(hiddenSet)
				}
				if on&AttrStrikethrough != 0 {
					_, _ = w.WriteString(strikethroughSet)
				}

				// If the bit is changed and is in previous, it
				// was turned off
				off := dAttr & attr
				if off&AttrBold != 0 {
					// Normal intensity isn't in terminfo
					_, _ = w.WriteString(boldDimReset)
					// Normal intensity turns off dim. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrDim != 0 {
						_, _ = w.WriteString(dimSet)
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					_, _ = w.WriteString(boldDimReset)
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						_, _ = w.WriteString(boldSet)
					}
				}
				if off&AttrItalic != 0 {
					_, _ = w.WriteString(italicReset)
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					_, _ = w.WriteString(blinkReset)
				}
				if off&AttrReverse != 0 {
					_, _ = w.WriteString(reverseReset)
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					_, _ = w.WriteString(hiddenReset)
				}
				if off&AttrStrikethrough != 0 {
					_, _ = w.WriteString(strikethroughReset)
				}
				attr = next.Attribute
			}

			if ulStyle != next.UnderlineStyle {
				ulStyle = next.UnderlineStyle
				switch capabilities.styledUnderlines {
				case true:
					_, _ = w.WriteString(tparm(ulStyleSet, ulStyle))
				case false:
					switch ulStyle {
					case UnderlineOff:
						_, _ = w.WriteString(underlineReset)
					default:
						// Fallback to single underlines
						_, _ = w.WriteString(underlineSet)
					}
				}
			}

			if link != next.Hyperlink || linkID != next.HyperlinkID {
				link = next.Hyperlink
				linkID = next.HyperlinkID
				switch {
				case link == "" && linkID == "":
					_, _ = w.WriteString(osc8End)
				case linkID == "":
					_, _ = w.WriteString(tparm(osc8, link))
				default:
					_, _ = w.WriteString(tparm(osc8WithID, link, linkID))
				}
			}
			switch next.Character {
			case "\x00":
				_, _ = w.WriteString(" ")
			default:
				_, _ = w.WriteString(next.Character)
			}
			// Advance the column by the width of this
			// character
			skip := advance(next.Character)
			for i := 1; i < skip+1; i += 1 {
				if col+i >= len(stdScreen.buf[row]) {
					break
				}
				// null out any cells we end up skipping
				lastRender.buf[row][col+i] = Cell{}
			}
			col += skip
		}
	}
	if nextCursor.visible && !lastCursor.visible {
		_, _ = w.WriteString(showCursor())
	}
}

func handleSequence(seq ansi.Sequence) {
	log.Debug("[stdin]", "sequence", seq)
	switch seq := seq.(type) {
	case ansi.Print:
		if inPaste {
			pasteBuf.WriteRune(rune(seq))
			return
		}
		PostMsg(decodeXterm(seq))
	case ansi.C0:
		PostMsg(decodeXterm(seq))
	case ansi.ESC:
		PostMsg(decodeXterm(seq))
	case ansi.SS3:
		PostMsg(decodeXterm(seq))
	case ansi.CSI:
		switch seq.Final {
		case 'c':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				for _, ps := range seq.Parameters {
					switch ps[0] {
					case 4:
						capabilities.sixels = true
						if graphicsProtocol < sixelGraphics {
							graphicsProtocol = sixelGraphics
						}
						log.Info("Sixels supported")
					}
				}
				if !initialized {
					deviceAttributesReceived <- struct{}{}
				}
			}
		case 'I':
			PostMsg(FocusIn{})
		case 'O':
			PostMsg(FocusOut{})
		case 'R':
			// KeyF1 or DSRCPR
			// This could be an F1 key, we need to buffer if we have
			// requested a DSRCPR (cursor position report)
			//
			// Kitty keyboard protocol disambiguates this scenario,
			// hopefully people are using that
			if cursorPositionRequested {
				cursorPositionRequested = false
				if len(seq.Parameters) != 2 {
					log.Error("not enough DSRCPR params")
					return
				}
				chCursorPositionReport <- seq.Parameters[0][0]
				chCursorPositionReport <- seq.Parameters[1][0]
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
						capabilities.sixels = true
						if graphicsProtocol < sixelGraphics {
							graphicsProtocol = sixelGraphics
						}
						log.Info("Sixels supported")
					}
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
					log.Info("Synchronized Update Mode supported")
					capabilities.synchronizedUpdate = true
				}
			}
			return
		case 'u':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				capabilities.kittyKeyboard = true
				log.Info("Kitty Keyboard Protocol supported")
				stdout.WriteString(tparm(kittyKBEnable, kittyKBFlags))
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
					inPaste = true
					return
				case 201:
					inPaste = false
					PostMsg(PasteMsg(pasteBuf.String()))
					pasteBuf.Reset()
					return
				}
			}
		case 'M', 'm':
			mouse, ok := parseMouseEvent(seq)
			if ok {
				PostMsg(mouse)
			}
			return
		}

		switch capabilities.kittyKeyboard {
		case true:
			key := parseKittyKbp(seq)
			if key.Codepoint != 0 {
				PostMsg(key)
			}
		default:
			PostMsg(decodeXterm(seq))
		}
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
					capabilities.styledUnderlines = true
					log.Info("Styled underlines supported")
				case hexEncode("RGB"):
					if !capabilities.rgb {
						capabilities.rgb = true
						log.Info("RGB Color supported")
					}
				}
			}
		case '|':
			if len(seq.Intermediate) < 1 {
				return
			}
			switch seq.Intermediate[0] {
			case '!':
				if string(seq.Data) == hexEncode("~VTE") {
					log.Info("Styled underlines supported")
					capabilities.styledUnderlines = true
				}
			}
		}
	case ansi.APC:
		if len(seq.Data) == 0 {
			return
		}
		if strings.HasPrefix(seq.Data, "G") {
			if capabilities.kittyGraphics {
				return
			}
			log.Info("Kitty graphics supported")
			capabilities.kittyGraphics = true
			if graphicsProtocol < kittyGraphics {
				graphicsProtocol = kittyGraphics
			}
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
			case osc52Paste <- string(b):
			case <-ctx.Done():
			}
		}
	}
}

func sendQueries() {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		log.Info("RGB color supported")
		capabilities.rgb = true
	}

	// We enter the alt screen without our buffered writer to prevent our
	// unicode query from bleeding onto the main terminal
	stdout.WriteString(decset(alternateScreen))
	_, _ = w.WriteString(decset(sixelScrolling))
	_, _ = w.WriteString(decrst(cursorVisibility))
	_, _ = w.WriteString(xtversion)
	_, _ = w.WriteString(kittyKBQuery)
	_, _ = w.WriteString(kittyGquery)
	_, _ = w.WriteString(sumQuery)

	_, _ = w.WriteString(xtsmSixelGeom)

	capabilities.unicode = queryUnicodeSupport()

	// Enable some modes
	_, _ = w.WriteString(decset(bracketedPaste)) // bracketed paste
	_, _ = w.WriteString(decset(cursorKeys))     // application cursor keys
	_, _ = w.WriteString(applicationMode)        // application cursor keys mode
	_, _ = w.WriteString(decset(mouseAllEvents))
	_, _ = w.WriteString(decset(mouseFocusEvents))
	_, _ = w.WriteString(decset(mouseSGR))
	_, _ = w.WriteString(clear)

	// Query some terminfo capabilities
	// Just another way to see if we have RGB support
	_, _ = w.WriteString(xtgettcap("RGB"))
	// We request Smulx to check for styled underlines. Technically, Smulx
	// only means the terminal supports different underline types (curly,
	// dashed, etc), but we'll assume the terminal also suppports underline
	// colors (CSI 58 : ...)
	_, _ = w.WriteString(xtgettcap("Smulx"))
	// Need to send tertiary for VTE based terminals. These don't respond to
	// XTGETTCAP
	_, _ = w.WriteString(tertiaryAttributes)
	// Send Device Attributes is last. Everything responds, and when we get
	// a response we'll return from init
	_, _ = w.WriteString(primaryAttributes)
	_, _ = w.Flush()
}

// HideCursor hides the hardware cursor
func HideCursor() {
	nextCursor.visible = false
}

// ShowCursor shows the cursor at the given colxrow, with the given style. The
// passed column and row are 0-indexed and global
func ShowCursor(col int, row int, style CursorStyle) {
	nextCursor.style = style
	nextCursor.col = col
	nextCursor.row = row
	nextCursor.visible = true
}

func showCursor() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(cursorStyle())
	buf.WriteString(tparm(cup, nextCursor.row+1, nextCursor.col+1))
	buf.WriteString(decset(cursorVisibility))
	return buf.String()
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func CursorPosition() (col int, row int) {
	// DSRCPR - reports cursor position
	cursorPositionRequested = true
	stdout.WriteString(dsrcpr)
	timeout := time.NewTimer(50 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warn("CursorPosition timed out")
		cursorPositionRequested = false
		return -1, -1
	case row = <-chCursorPositionReport:
		// if we get one, we'll get another
		col = <-chCursorPositionReport
		return col - 1, row - 1
	}
}

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

func cursorStyle() string {
	if nextCursor.style == CursorDefault {
		return cursorStyleReset
	}
	return tparm(cursorStyleSet, int(nextCursor.style))
}

// ClipboardPush copies the provided string to the system clipboard
func ClipboardPush(s string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	stdout.WriteString(tparm(osc52put, b64))
}

// ClipboardPop requests the content from the system clipboard. ClipboardPop works by
// requesting the data from the underlying terminal, which responds back with
// the data. Depending on usage, this could take some time. Callers can provide
// a context to set a deadline for this function to return. An error will be
// returned if the context is cancelled.
func ClipboardPop(ctx context.Context) (string, error) {
	stdout.WriteString(osc52pop)
	select {
	case str := <-osc52Paste:
		return str, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Notify (attempts) to send a system notification. If title is the empty
// string, OSC9 will be used - otherwise osc777 is used
func Notify(title string, body string) {
	if title == "" {
		stdout.WriteString(tparm(osc9notify, body))
		return
	}
	stdout.WriteString(tparm(osc777notify, title, body))
}

// SetTitle sets the terminal's title via OSC 2
func SetTitle(s string) {
	stdout.WriteString(tparm(setTitle, s))
}

// Bell sends a BEL control signal to the terminal
func Bell() {
	stdout.WriteString("\a")
}

// advance returns the extra amount to advance the column by when rendering
func advance(ch string) int {
	w := RenderedWidth(ch) - 1
	if w < 0 {
		return 0
	}
	return w
}

// Determines if our terminal knows about unicode. The test string should
// produce an emoji that is ~1.5 cells wide. Terminals that properly render this
// will report that their cursor has moved forward by 2 total cells. Terminals
// that don't render this properly will report (probably) 4 cells of movement
// (one for each emoji in the ZWJ sequence)
func queryUnicodeSupport() bool {
	stdout.WriteString(tparm(cup, 1, 1))
	test := "👩‍🚀"
	originX, _ := CursorPosition()
	if originX < 0 {
		return false
	}
	stdout.WriteString(test)
	newX, _ := CursorPosition()
	if newX-originX > 2 {
		return false
	}
	log.Info("Unicode supported")
	return true
}

// RenderedWidth returns the rendered width of the provided string. The result
// is dependent on if your terminal can support unicode properly. RenderedWidth
// must be called after Init to ensure Unicode support has been detected.
//
// This is best effort. It will usually be correct, and in the few cases it's
// wrong will end up wrong in the nicer-rendering way (complex emojis will have
// extra space after them. This is preferable to messing up the internal model)
func RenderedWidth(s string) int {
	if capabilities.unicode {
		return uniseg.StringWidth(s)
	}
	w := 0
	for _, r := range s {
		// Why runewidth here? uniseg differs from wcwidth a bit but is
		// more accurate for terminals which support unicode. We use
		// uniseg there, and runewidth here
		w += runewidth.RuneWidth(r)
	}
	return w
}

func SetMouseShape(shape MouseShape) {
	nextMouseShape = shape
}

// reportWinsize posts a Resize Msg
func reportWinsize() {
	ws, err := unix.IoctlGetWinsize(int(stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		log.Error("couldn't get winsize", "error", err)
		return
	}
	winsize = Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}
	PostMsg(winsize)
}
