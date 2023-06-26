package rtk

import (
	"bytes"
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
)

// Converts a string into a slice of EGCs suitable to assign to terminal cells
func EGCs(s string) []string {
	egcs := []string{}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		egcs = append(egcs, g.Str())
	}
	return egcs
}

func Run(model Model) error {
	// Setup terminal
	err := setupTermInfo()
	if err != nil {
		return err
	}
	tty = os.Stdout
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
	PostMsg(Init{})
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
		case sendMsg:
			msg.model.Update(msg.msg)
			model.Draw(Window{})
		case funcMsg:
			msg.fn()
			model.Draw(Window{})
		case partialDrawMsg:
			msg.model.Draw(msg.window)
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
	tty.WriteString(cvvis) // show the cursor
	tty.WriteString(sgr0)  // reset fg, bg, attrs
	tty.WriteString(clear)

	// Disable any modes we enabled
	tty.WriteString(bd)      // bracketed paste
	tty.WriteString(kkbpPop) // kitty keyboard
	tty.WriteString(rmkx)    // application cursor keys
	tty.WriteString(resetMouse)

	exitAltScreen()

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
					renderBuf.WriteString(civis)
				}
				if capabilities.synchronizedUpdate {
					renderBuf.WriteString(sumSet)
				}
			}
			lastRender.buf[row][col] = next
			if reposition {
				renderBuf.WriteString(tparm(cup, row, col))
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
					renderBuf.WriteString(fgop)
				case 1:
					renderBuf.WriteString(tparm(setaf, int(ps[0])))
				case 3:
					out := tparm(setrgbf, int(ps[0]), int(ps[1]), int(ps[2]))
					renderBuf.WriteString(out)
				}
			}

			if bg != next.Background {
				bg = next.Background
				ps := bg.Params()
				switch len(ps) {
				case 0:
					renderBuf.WriteString(bgop)
				case 1:
					renderBuf.WriteString(tparm(setab, int(ps[0])))
				case 3:
					out := tparm(setrgbb, int(ps[0]), int(ps[1]), int(ps[2]))
					renderBuf.WriteString(out)
				}
			}

			if attr != next.Attribute {
				// find the ones that have changed
				dAttr := attr ^ next.Attribute
				// If the bit is changed and in next, it was
				// turned on
				on := dAttr & next.Attribute

				if on&AttrBold != 0 {
					renderBuf.WriteString(bold)
				}
				if on&AttrDim != 0 {
					renderBuf.WriteString(dim)
				}
				if on&AttrItalic != 0 {
					renderBuf.WriteString(sitm)
				}
				if on&AttrUnderline != 0 {
					renderBuf.WriteString(smul)
				}
				if on&AttrBlink != 0 {
					renderBuf.WriteString(blink)
				}
				if on&AttrReverse != 0 {
					renderBuf.WriteString(rev)
				}
				if on&AttrInvisible != 0 {
					renderBuf.WriteString(invis)
				}
				if on&AttrStrikethrough != 0 {
					renderBuf.WriteString(smxx)
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
						renderBuf.WriteString(dim)
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					renderBuf.WriteString(boldDimReset)
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						renderBuf.WriteString(bold)
					}
				}
				if off&AttrItalic != 0 {
					renderBuf.WriteString(ritm)
				}
				if off&AttrUnderline != 0 {
					renderBuf.WriteString(rmul)
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					renderBuf.WriteString(endBlink)
				}
				if off&AttrReverse != 0 {
					renderBuf.WriteString(rmso)
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					renderBuf.WriteString(endInvis)
				}
				if off&AttrStrikethrough != 0 {
					renderBuf.WriteString(rmxx)
				}
				attr = next.Attribute
			}
			renderBuf.WriteString(next.Character)
		}
	}
	if renderBuf.Len() != 0 {
		renderBuf.WriteString(sgr0)
		if capabilities.synchronizedUpdate {
			renderBuf.WriteString(sumReset)
		}
	}
	if cursor.visible {
		renderBuf.WriteString(showCursor())
	}
	if !cursor.visible {
		renderBuf.WriteString(civis)
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
		PostMsg(Key{Codepoint: rune(seq)})
	case ansi.C0:
		key := Key{}
		switch rune(seq) {
		case 0x08:
			key.Codepoint = KeyBackspace
		case 0x09:
			key.Codepoint = KeyTab
		case 0x0D:
			key.Codepoint = KeyEnter
		case 0x1B:
			key.Codepoint = KeyEsc
		default:
			switch {
			case rune(seq) == 0x00:
				key.Codepoint = '@'
			case rune(seq) < 0x1A:
				// normalize these to lowercase runes
				key.Codepoint = rune(seq) + 0x60
			case rune(seq) < 0x20:
				key.Codepoint = rune(seq) + 0x40
			}
		}
		key.Modifiers = ModCtrl
		PostMsg(key)
	case ansi.ESC:
		PostMsg(Key{
			Codepoint: seq.Final,
			Modifiers: ModAlt,
		})
	case ansi.SS3:
		Logger.Debug("SS3!!!!")
		lookup := fmt.Sprintf("\x1bO%c", seq)
		key, ok := keyMap[lookup]
		if ok {
			PostMsg(key)
			return
		}
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
				case 201:
					inPaste = false
					PostMsg(Paste(pasteBuf.String()))
					pasteBuf.Reset()
				}
			}
		case 'M', 'm':
			mouse, ok := parseMouseEvent(seq)
			if ok {
				PostMsg(mouse)
			}
		}

		switch capabilities.kittyKeyboard {
		case true:
			key := parseKittyKbp(seq)
			if key.Codepoint != 0 {
				PostMsg(key)
			}
		default:
			// Lookup the key from terminfo
			params := []string{}
			for _, ps := range seq.Parameters {
				if len(ps) > 1 {
					Logger.Debug("Unknown sequence", "sequence", seq)
				}
				params = append(params, fmt.Sprintf("%d", ps[0]))
			}
			lookup := fmt.Sprintf("\x1b[%s%c", strings.Join(params, ";"), seq.Final)
			Logger.Info("LOOKING UP %s%c", strings.TrimPrefix(lookup, "\x1b"))
			key, ok := keyMap[lookup]
			if !ok {
				return
			}
			PostMsg(key)

		}
	}
}

func sendQueries() {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		Logger.Info("RGB color supported")
		capabilities.rgb = true
	}
	enterAltScreen()
	tty.WriteString(civis)
	tty.WriteString(xtversion)
	tty.WriteString(kkbpQuery)
	tty.WriteString(sumQuery)

	// Enable some modes
	tty.WriteString(be)       // bracketed paste
	tty.WriteString(smkx)     // application cursor keys
	tty.WriteString(setMouse) // mouse

	tty.WriteString(clear)
}

// Terminal controls

// Enter the alternate screen (for fullscreen applications)
func enterAltScreen() {
	Logger.Debug("Entering alt screen")
	tty.WriteString(smcup)
}

func exitAltScreen() {
	Logger.Debug("Exiting alt screen")
	tty.WriteString(rmcup)
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
	buf.WriteString(tparm(cup, cursor.row, cursor.col))
	buf.WriteString(cvvis)
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
		return se
	}
	return tparm(ss, int(cursor.style))
}
