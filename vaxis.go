package vaxis

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis/ansi"
	"github.com/mattn/go-tty"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
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

	// renderBuf buffers the output that we'll send to the tty
	renderBuf *bytes.Buffer
	// refresh signals if we should do a delta render or full render
	refresh bool
	// stdScreen is the stdScreen Surface
	stdScreen *screen
	// lastRender stores the state of our last render. This is used to
	// optimize what we update on the next render
	lastRender *screen

	// device is our tty
	device *tty.TTY
	// ttyOut is the terminal we are talking with
	ttyOut *os.File
	// savedState *term.State

	capabilities struct {
		synchronizedUpdate bool
		rgb                bool
		kittyGraphics      bool
		kittyKeyboard      bool
		styledUnderlines   bool
		sixels             bool
	}
	winsize Resize

	lastGraphicPlacements map[int]*placement
	nextGraphicPlacements map[int]*placement

	cursor struct {
		row     int
		col     int
		style   CursorStyle
		visible bool
	}
	// Statistics
	renders int
	elapsed time.Duration

	imgBuf *bytes.Buffer
)

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
	// By default, if support is detected the protocol will be used. Your
	// application will receive key release events as well as improved key
	// support
	DisableKittyKeyboard bool
	// ReportKeyboardEvents will report key release and key repeat events if
	// KittyKeyboardProtocol is enabled and supported by the terminal
	ReportKeyboardEvents bool
}

func Init(opts Options) error {
	// Let's give some deadline for our queries responding. If they don't,
	// it means the terminal doesn't respond to Primary Device Attributes
	// and that is a problem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	var err error
	device, err = tty.Open()
	if err != nil {
		return err
	}
	ttyOut = device.Output()
	parser := ansi.NewParser(device.Input())
	if opts.Logger != nil {
		log = opts.Logger
	}
	if opts.ReportKeyboardEvents {
		kittyKBFlags += 2
	}

	// Rendering
	renderBuf = &bytes.Buffer{}
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

	chSIGWINCH := device.SIGWINCH()

	reportWinsize()
	deviceAttributesReceived = make(chan struct{})
	sendQueries()

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
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-deviceAttributesReceived:
		close(deviceAttributesReceived)
		initialized = true
		cancel()
	}

	// Disable features based on options. We've already received all of our
	// queries so this still has effect
	if opts.DisableKittyKeyboard {
		capabilities.kittyKeyboard = false
	}
	return nil
}

// PollMsg returns the next Msg. When a QuitMsg is received, all input processing
// will cease.
func PollMsg() Msg {
	var m Msg
	for msg := range msgs.ch {
		if msg == nil {
			continue
		}
		switch msg := msg.(type) {
		case QuitMsg:
			close(chQuit)
			return msg
		case Resize:
			stdScreen.resize(msg.Cols, msg.Rows)
			lastRender.resize(msg.Cols, msg.Rows)
		}
		m = msg
		break
	}
	return m
}

// Msgs provides access to the channel of Msgs
func Msgs() chan Msg {
	return msgs.Chan()
}

// Run operates an event loop for the provided Model. Users of the Run loop
// don't need to explicitly render, the loop will render every event
func Run(model Model) error {
	for msg := range msgs.ch {
		if msg == nil {
			continue
		}
		win := Window{}
		switch msg := msg.(type) {
		case QuitMsg:
			close(chQuit)
			model.Update(msg)
			Close()
			return nil
		case Resize:
			stdScreen.resize(msg.Cols, msg.Rows)
			lastRender.resize(msg.Cols, msg.Rows)
			model.Update(msg)
			model.Draw(win)
		case SendMsg:
			msg.Model.Update(msg.Msg)
			model.Draw(win)
		case FuncMsg:
			msg.Func()
			model.Draw(win)
		case DrawModelMsg:
			msg.Model.Draw(msg.Window)
		default:
			model.Update(msg)
			model.Draw(win)
		}
		Render()
	}
	return nil
}

// reportWinsize posts a Resize Msg
func reportWinsize() {
	col, row, x, y, err := device.SizePixel()
	if err != nil {
		log.Error("couldn't get winsize", "error", err)
		return
	}
	winsize = Resize{
		Cols:   col,
		Rows:   row,
		XPixel: x,
		YPixel: y,
	}
	PostMsg(winsize)
}

func Quit() {
	PostMsg(QuitMsg{})
}

func Close() {
	ttyOut.WriteString(decset(cursorVisibility)) // show the cursor
	ttyOut.WriteString(sgrReset)                 // reset fg, bg, attrs
	ttyOut.WriteString(clear)

	// Disable any modes we enabled
	ttyOut.WriteString(decrst(bracketedPaste)) // bracketed paste
	ttyOut.WriteString(kittyKBPop)             // kitty keyboard
	ttyOut.WriteString(decrst(cursorKeys))
	ttyOut.WriteString(numericMode)
	ttyOut.WriteString(decrst(mouseAllEvents))
	ttyOut.WriteString(decrst(mouseFocusEvents))
	ttyOut.WriteString(decrst(mouseSGR))

	ttyOut.WriteString(decrst(alternateScreen))

	device.Close()

	log.Info("Renders", "val", renders)
	if renders != 0 {
		log.Info("Time/render", "val", elapsed/time.Duration(renders))
	}
}

// Render the surface's content to the terminal
func Render() {
	start := time.Now()
	defer renderBuf.Reset()
	out := render()
	if out != "" {
		ttyOut.WriteString(out)
	}
	elapsed += time.Since(start)
	renders += 1
	refresh = false
}

// Refresh forces a full render of the entire screen
func Refresh() {
	refresh = true
	Render()
}

func render() string {
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
		if renderBuf.Len() == 0 {
			if cursor.visible {
				// Hide cursor if it's visible
				renderBuf.WriteString(decrst(cursorVisibility))
			}
			if capabilities.synchronizedUpdate {
				renderBuf.WriteString(decset(synchronizedUpdate))
			}
		}
		renderBuf.WriteString(p.delete())
		delete(lastGraphicPlacements, id)
	}
	// draw new placements
	for id, p := range nextGraphicPlacements {
		p.lockRegion()
		if _, ok := lastGraphicPlacements[id]; ok {
			continue
		}
		if renderBuf.Len() == 0 {
			if cursor.visible {
				// Hide cursor if it's visible
				renderBuf.WriteString(decrst(cursorVisibility))
			}
			if capabilities.synchronizedUpdate {
				renderBuf.WriteString(decset(synchronizedUpdate))
			}
		}
		renderBuf.WriteString(tparm(cup, p.row+1, p.col+1))
		renderBuf.WriteString(p.draw())
		lastGraphicPlacements[id] = p
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
				col += advance(next.Character)
				continue
			}
			if renderBuf.Len() == 0 {
				if cursor.visible {
					// Hide cursor if it's visible
					renderBuf.WriteString(decrst(cursorVisibility))
				}
				if capabilities.synchronizedUpdate {
					renderBuf.WriteString(decset(synchronizedUpdate))
				}
			}
			lastRender.buf[row][col] = next
			if reposition {
				renderBuf.WriteString(tparm(cup, row+1, col+1))
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
				switch len(ps) {
				case 0:
					renderBuf.WriteString(fgReset)
				case 1:
					switch {
					case ps[0] < 8:
						renderBuf.WriteString(fmt.Sprintf(fgSet, ps[0]))
					case ps[0] < 16:
						renderBuf.WriteString(fmt.Sprintf(fgBrightSet, ps[0]-8))
					default:
						renderBuf.WriteString(fmt.Sprintf(fgIndexSet, ps[0]))
					}
				case 3:
					out := fmt.Sprintf(fgRGBSet, ps[0], ps[1], ps[2])
					out = strings.TrimPrefix(out, "\x1b")
					renderBuf.WriteString(fmt.Sprintf(fgRGBSet, ps[0], ps[1], ps[2]))
				}
			}

			if bg != next.Background {
				bg = next.Background
				ps := bg.Params()
				switch len(ps) {
				case 0:
					renderBuf.WriteString(bgReset)
				case 1:
					switch {
					case ps[0] < 8:
						renderBuf.WriteString(fmt.Sprintf(bgSet, ps[0]))
					case ps[0] < 16:
						renderBuf.WriteString(fmt.Sprintf(bgBrightSet, ps[0]-8))
					default:
						renderBuf.WriteString(fmt.Sprintf(bgIndexSet, ps[0]))
					}
				case 3:
					renderBuf.WriteString(fmt.Sprintf(bgRGBSet, ps[0], ps[1], ps[2]))
				}
			}

			if capabilities.styledUnderlines {
				if ul != next.Underline {
					ul = next.Underline
					ps := bg.Params()
					switch len(ps) {
					case 0:
						renderBuf.WriteString(ulColorReset)
					case 1:
						renderBuf.WriteString(fmt.Sprintf(ulIndexSet, ps[0]))
					case 3:
						renderBuf.WriteString(fmt.Sprintf(ulRGBSet, ps[0], ps[1], ps[2]))
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
					renderBuf.WriteString(boldSet)
				}
				if on&AttrDim != 0 {
					renderBuf.WriteString(dimSet)
				}
				if on&AttrItalic != 0 {
					renderBuf.WriteString(italicSet)
				}
				if on&AttrBlink != 0 {
					renderBuf.WriteString(blinkSet)
				}
				if on&AttrReverse != 0 {
					renderBuf.WriteString(reverseSet)
				}
				if on&AttrInvisible != 0 {
					renderBuf.WriteString(hiddenSet)
				}
				if on&AttrStrikethrough != 0 {
					renderBuf.WriteString(strikethroughSet)
				}

				// If the bit is changed and is in previous, it
				// was turned off
				off := dAttr & attr
				if off&AttrBold != 0 {
					// Normal intensity isn't in terminfo
					renderBuf.WriteString(boldDimReset)
					// Normal intensity turns off dim. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrDim != 0 {
						renderBuf.WriteString(dimSet)
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					renderBuf.WriteString(boldDimReset)
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						renderBuf.WriteString(boldSet)
					}
				}
				if off&AttrItalic != 0 {
					renderBuf.WriteString(italicReset)
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					renderBuf.WriteString(blinkReset)
				}
				if off&AttrReverse != 0 {
					renderBuf.WriteString(reverseReset)
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					renderBuf.WriteString(hiddenReset)
				}
				if off&AttrStrikethrough != 0 {
					renderBuf.WriteString(strikethroughReset)
				}
				attr = next.Attribute
			}

			if ulStyle != next.UnderlineStyle {
				ulStyle = next.UnderlineStyle
				switch capabilities.styledUnderlines {
				case true:
					renderBuf.WriteString(tparm(ulStyleSet, ulStyle))
				case false:
					switch ulStyle {
					case UnderlineOff:
						renderBuf.WriteString(underlineReset)
					default:
						// Fallback to single underlines
						renderBuf.WriteString(underlineSet)
					}
				}
			}

			if link != next.Hyperlink || linkID != next.HyperlinkID {
				link = next.Hyperlink
				linkID = next.HyperlinkID
				switch {
				case link == "" && linkID == "":
					renderBuf.WriteString(osc8End)
				case linkID == "":
					renderBuf.WriteString(tparm(osc8, link))
				default:
					renderBuf.WriteString(tparm(osc8WithID, link, linkID))
				}
			}
			renderBuf.WriteString(next.Character)
			// Advance the column by the width of this
			// character
			col += advance(next.Character)
		}
	}
	if renderBuf.Len() != 0 {
		renderBuf.WriteString(sgrReset)
		if capabilities.synchronizedUpdate {
			renderBuf.WriteString(decrst(synchronizedUpdate))
		}
	}
	if cursor.visible {
		renderBuf.WriteString(showCursor())
	}
	if !cursor.visible {
		renderBuf.WriteString(decrst(cursorVisibility))
	}
	return renderBuf.String()
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
				ttyOut.WriteString(tparm(kittyKBEnable, kittyKBFlags))
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
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
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

	ttyOut.WriteString(decset(alternateScreen))
	ttyOut.WriteString(decrst(cursorVisibility))
	ttyOut.WriteString(xtversion)
	ttyOut.WriteString(kittyKBQuery)
	ttyOut.WriteString(kittyGquery)
	ttyOut.WriteString(sumQuery)

	ttyOut.WriteString(xtsmSixelGeom)

	// Enable some modes
	ttyOut.WriteString(decset(bracketedPaste)) // bracketed paste
	ttyOut.WriteString(decset(cursorKeys))     // application cursor keys
	ttyOut.WriteString(applicationMode)        // application cursor keys mode
	ttyOut.WriteString(decset(mouseAllEvents))
	ttyOut.WriteString(decset(mouseFocusEvents))
	ttyOut.WriteString(decset(mouseSGR))
	ttyOut.WriteString(clear)

	// Query some terminfo capabilities
	// Just another way to see if we have RGB support
	ttyOut.WriteString(xtgettcap("RGB"))
	// We request Smulx to check for styled underlines. Technically, Smulx
	// only means the terminal supports different underline types (curly,
	// dashed, etc), but we'll assume the terminal also suppports underline
	// colors (CSI 58 : ...)
	ttyOut.WriteString(xtgettcap("Smulx"))
	// Need to send tertiary for VTE based terminals. These don't respond to
	// XTGETTCAP
	ttyOut.WriteString(tertiaryAttributes)
	// Send Device Attributes is last. Everything responds, and when we get
	// a response we'll return from init
	ttyOut.WriteString(primaryAttributes)
}

func HideCursor() {
	cursor.visible = false
}

func ShowCursor(col int, row int, style CursorStyle) {
	cursor.style = style
	cursor.col = col
	cursor.row = row
	cursor.visible = true
}

func showCursor() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(cursorStyle())
	buf.WriteString(tparm(cup, cursor.row+1, cursor.col+1))
	buf.WriteString(decset(cursorVisibility))
	return buf.String()
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func CursorPosition() (col int, row int) {
	// DSRCPR - reports cursor position
	cursorPositionRequested = true
	ttyOut.WriteString(dsrcpr)
	timeout := time.NewTimer(10 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warn("CursorPosition timed out")
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
	if cursor.style == CursorDefault {
		return cursorStyleReset
	}
	return tparm(cursorStyleSet, int(cursor.style))
}

// ClipboardPush copies the provided string to the system clipboard
func ClipboardPush(s string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	ttyOut.WriteString(tparm(osc52put, b64))
}

// ClipboardPop requests the content from the system clipboard. ClipboardPop works by
// requesting the data from the underlying terminal, which responds back with
// the data. Depending on usage, this could take some time. Callers can provide
// a context to set a deadline for this function to return. An error will be
// returned if the context is cancelled.
func ClipboardPop(ctx context.Context) (string, error) {
	ttyOut.WriteString(osc52pop)
	select {
	case str := <-osc52Paste:
		return str, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// advance returns the extra amount to advance the column by when rendering
func advance(ch string) int {
	w := uniseg.StringWidth(ch) - 1
	if w < 0 {
		return 0
	}
	return w
}
