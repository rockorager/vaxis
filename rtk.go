package rtk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/rtk/ansi"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	// Logger is a slog.Logger that rtk will dump logs to. rtk uses stdlib
	// levels for logging, and a trace level at -8
	// TODO make the trace level
	Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	// async is an asynchronous queue, provided as a helper for applications
	async *queue[Msg]
	// msgs is the main event loop Msg queue
	msgs *queue[Msg]
	// chQuit is a channel to signal to running goroutines that we are
	// quitting
	chQuit chan struct{}
	// inPaste signals that we are within a bracketed paste
	inPaste bool
	// pasteBuf buffers bracketed paste text
	pasteBuf *bytes.Buffer
	// Have we requested a cursor position?
	cursorPositionRequested bool
	chCursorPositionReport  chan int

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

	// tty is the terminal we are talking with
	tty        *os.File
	savedState *term.State

	capabilities struct {
		synchronizedUpdate bool
		rgb                bool
		kittyKeyboard      bool
		styledUnderlines   bool
	}

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

func Init(ctx context.Context) error {
	var err error
	tty, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
	parser := ansi.NewParser(tty)
	savedState, err = term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return err
	}

	// Rendering
	renderBuf = &bytes.Buffer{}
	lastRender = newScreen()
	stdScreen = newScreen()

	// pasteBuf buffers bracketed paste
	pasteBuf = &bytes.Buffer{}

	// Setup internals and signal handling
	msgs = newQueue[Msg]()
	chQuit = make(chan struct{})
	chOSSignals := make(chan os.Signal)
	chCursorPositionReport = make(chan int)
	signal.Notify(chOSSignals,
		syscall.SIGWINCH, // triggers Resize
		syscall.SIGCONT,  // triggers Resize
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGTERM,
	)
	PostMsg(InitMsg{})
	resize(int(tty.Fd()))
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
			case sig := <-chOSSignals:
				switch sig {
				case syscall.SIGWINCH, syscall.SIGCONT:
					resize(int(tty.Fd()))
				default:
					Logger.Debug("Signal caught", "signal", sig)
					quit()
					return
				}
			case <-chQuit:
				return
			}
		}
	}()
	return nil
}

func Run(model Model) error {
	for msg := range msgs.ch {
		if msg == nil {
			continue
		}
		switch msg := msg.(type) {
		case QuitMsg:
			close(chQuit)
			model.Update(msg)
			quit()
			return nil
		case Resize:
			stdScreen.resize(msg.Cols, msg.Rows)
			lastRender.resize(msg.Cols, msg.Rows)
			model.Update(msg)
			model.Draw(Window{})
		case SendMsg:
			msg.Model.Update(msg.Msg)
			model.Draw(Window{})
		case FuncMsg:
			msg.Func()
			model.Draw(Window{})
		case DrawModelMsg:
			msg.Model.Draw(msg.Window)
		default:
			model.Update(msg)
			model.Draw(Window{})
		}
		Render()
	}
	return nil
}

// resize posts a Resize Msg
func resize(fd int) error {
	size, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return err
	}
	PostMsg(Resize{
		Cols: int(size.Col),
		Rows: int(size.Row),
	})
	return nil
}

func Quit() {
	PostMsg(QuitMsg{})
}

func quit() {
	tty.WriteString(decset(cursorVisibility)) // show the cursor
	tty.WriteString(sgrReset)                 // reset fg, bg, attrs
	tty.WriteString(clear)

	// Disable any modes we enabled
	tty.WriteString(decrst(bracketedPaste)) // bracketed paste
	tty.WriteString(kkbpPop)                // kitty keyboard
	tty.WriteString(decrst(cursorKeys))
	tty.WriteString(numericMode)
	tty.WriteString(decrst(mouseAllEvents))
	tty.WriteString(decrst(mouseFocusEvents))
	tty.WriteString(decrst(mouseSGR))

	tty.WriteString(decrst(alternateScreen))

	term.Restore(int(tty.Fd()), savedState)

	Logger.Info("Renders", "val", renders)
	if renders != 0 {
		Logger.Info("Time/render", "val", elapsed/time.Duration(renders))
	}
}

// Render the surface's content to the terminal
func Render() {
	start := time.Now()
	defer renderBuf.Reset()
	out := render()
	if out != "" {
		tty.WriteString(out)
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
	)
	for row := range stdScreen.buf {
		for col := range stdScreen.buf[row] {
			next := stdScreen.buf[row][col]
			if next == lastRender.buf[row][col] && !refresh {
				reposition = true
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

			renderBuf.WriteString(next.Character)
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
	Logger.Debug("[stdin]", "sequence", seq)
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
					Logger.Error("not enough DSRCPR params")
					return
				}
				chCursorPositionReport <- seq.Parameters[0][0]
				chCursorPositionReport <- seq.Parameters[1][0]
				return
			}
		case 'y':
			// DECRPM - DEC Report Mode
			if len(seq.Parameters) < 1 {
				Logger.Error("not enough DECRPM params")
				return
			}
			switch seq.Parameters[0][0] {
			case 2026:
				if len(seq.Parameters) < 2 {
					Logger.Error("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1][0] {
				case 1, 2:
					Logger.Info("Synchronized Update Mode supported")
					capabilities.synchronizedUpdate = true
				}
			}
			return
		case 'u':
			if len(seq.Intermediate) == 1 && seq.Intermediate[0] == '?' {
				capabilities.kittyKeyboard = true
				Logger.Info("Kitty Keyboard Protocol supported")
				tty.WriteString(kkbpEnable)
				return
			}
		case '~':
			if len(seq.Intermediate) == 0 {
				if len(seq.Parameters) == 0 {
					Logger.Error("[CSI] unknown sequence with final '~'")
					return
				}
				switch seq.Parameters[0][0] {
				case 200:
					inPaste = true
					return
				case 201:
					inPaste = false
					PostMsg(Paste(pasteBuf.String()))
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
					Logger.Error("error parsing XTGETTCAP", "value", string(seq.Data))
				}
				switch vals[0] {
				case hexEncode("Smulx"):
					capabilities.styledUnderlines = true
					Logger.Info("Styled underlines supported")
				case hexEncode("RGB"):
					if !capabilities.rgb {
						capabilities.rgb = true
						Logger.Info("RGB Color supported")
					}
				}
			}
		}
	}
}

func sendQueries() {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		Logger.Info("RGB color supported")
		capabilities.rgb = true
	}

	tty.WriteString(decset(alternateScreen))
	tty.WriteString(decrst(cursorVisibility))
	tty.WriteString(xtversion)
	tty.WriteString(kkbpQuery)
	tty.WriteString(sumQuery)

	// Enable some modes
	tty.WriteString(decset(bracketedPaste)) // bracketed paste
	tty.WriteString(decset(cursorKeys))     // application cursor keys
	tty.WriteString(applicationMode)        // application cursor keys mode
	tty.WriteString(decset(mouseAllEvents))
	tty.WriteString(decset(mouseFocusEvents))
	tty.WriteString(decset(mouseSGR))
	tty.WriteString(clear)

	// Query some terminfo capabilities
	tty.WriteString(xtgettcap("RGB"))
	tty.WriteString(xtgettcap("Smulx"))
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
	tty.WriteString(dsrcpr)
	timeout := time.NewTimer(10 * time.Millisecond)
	select {
	case <-timeout.C:
		Logger.Warn("CursorPosition timed out")
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
