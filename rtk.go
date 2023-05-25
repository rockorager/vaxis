package rtk

import (
	"bytes"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/rtk/ansi"
	"git.sr.ht/~rockorager/rtk/log"
	"github.com/rivo/uniseg"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
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

// RTK is the surface associated with the terminal screen. It will always
// have an offset of 0,0 and a size equal to the size of the terminal screen
type RTK struct {
	// std is the buffered state of the stdSurface. Applications write cells
	// to this Surface, which is then rendered
	std *stdSurface
	// model is the last rendered state of the stdSurface
	model *stdSurface
	// The terminfo entry
	info *terminfo

	// Statistics
	// elapsed time spent rendering
	elapsed time.Duration
	// number of renders we have done
	renders uint64

	// queues
	msgs *queue[Msg]

	mu sync.Mutex
	// buffer to collect our output from flush
	outBuf *bytes.Buffer
	// input parser
	parser *ansi.Parser
	quit   chan struct{}
	// ss3 is true if we have received an \EO sequence. We have to buffer
	// this specific one for keyboard input of some keys
	ss3 bool
	// dsrcpr is true if we have requested a cursor position report
	dsrcpr   bool
	chDSRCPR chan int
	// refresh is true if we are redrawing the entire screen, ignoring
	// incremental renders
	refresh bool
	// saved state, restored on Close
	saved   *term.State
	signals chan os.Signal
	// the underlying tty
	tty *os.File

	caps struct {
		// Synchronized Update Mode
		SUM bool
	}
}

func New() (*RTK, error) {
	rtk := &RTK{
		msgs:    newQueue[Msg](),
		outBuf:  &bytes.Buffer{},
		parser:  ansi.NewParser(os.Stdout),
		tty:     os.Stdout,
		quit:    make(chan struct{}),
		signals: make(chan os.Signal),
	}
	rtk.std = newStdSurface(rtk)
	rtk.model = newStdSurface(rtk)
	info, err := infocmp(os.Getenv("TERM"))
	if err != nil {
		return nil, err
	}
	rtk.info = info
	log.Tracef("terminfo entry found for TERM=%s", rtk.info.Names)

	rtk.saved, err = term.MakeRaw(int(rtk.tty.Fd()))

	signal.Notify(rtk.signals,
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

	size, err := unix.IoctlGetWinsize(int(rtk.tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return nil, err
	}
	rtk.std.resize(int(size.Col), int(size.Row))
	rtk.model.resize(int(size.Col), int(size.Row))
	rtk.PostMsg(Init{})

	go func() {
		for {
			select {
			case seq := <-rtk.parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					log.Tracef(seq.String())
					return
				default:
					rtk.handleSequence(seq)
				}
			case sig := <-rtk.signals:
				switch sig {
				case syscall.SIGWINCH, syscall.SIGCONT:
					rtk.mu.Lock()
					size, err := unix.IoctlGetWinsize(int(rtk.tty.Fd()), unix.TIOCGWINSZ)
					rtk.mu.Unlock()
					if err != nil {
						log.Error(err)
					}
					rtk.std.resize(int(size.Col), int(size.Row))
					rtk.model.resize(int(size.Col), int(size.Row))
				default:
					log.Debugf("Signal caught: %s. Closing", sig)
					rtk.Close()
					return
				}
			case <-rtk.quit:
				return
			}
		}
	}()

	rtk.sendQueries()
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		rtk.info.Strings["setfrgb"] = "\x1b[38;2;%p1%d;%p2%d;%p3%dm"
		rtk.info.Strings["setbrgb"] = "\x1b[48;2;%p1%d;%p2%d;%p3%dm"
		rtk.info.Strings["setfbrgb"] = "\x1b[38;2;%p1%d;%p2%d;%p3%d;48;2;%p4%d;%p5%d;%p6%dm"
	}
	return rtk, nil
}

func (rtk *RTK) Close() {
	rtk.PostMsg(Quit{})
	close(rtk.quit)
	term.Restore(int(rtk.tty.Fd()), rtk.saved)
	log.Infof("Renders = %v", rtk.renders)
	log.Infof("Time/render = %s", rtk.elapsed/time.Duration(rtk.renders))
}

func (rtk *RTK) StdSurface() Surface {
	return rtk.std
}

// Msgs returns the channel of Msgs
func (rtk *RTK) Msgs() chan Msg {
	return rtk.msgs.ch
}

func (rtk *RTK) PostMsg(msg Msg) {
	rtk.msgs.push(msg)
}

// Render the surface's content to the terminal
func (rtk *RTK) Render() {
	start := time.Now()
	rtk.mu.Lock()
	defer rtk.mu.Unlock()
	defer rtk.outBuf.Reset()
	out := rtk.render()
	rtk.tty.WriteString(out)
	rtk.elapsed += time.Since(start)
	rtk.renders += 1
	rtk.refresh = false
}

// Refresh forces a full render of the entire screen
func (rtk *RTK) Refresh() {
	rtk.refresh = true
	rtk.Render()
}

func (rtk *RTK) render() string {
	var (
		cup        = rtk.info.Strings["cup"]
		reposition = true
		fg         Color
		bg         Color
		attr       AttributeMask
	)
	rtk.std.mu.Lock()
	defer rtk.std.mu.Unlock()
	for row := range rtk.std.buf {
		for col := range rtk.std.buf[row] {
			next := rtk.std.buf[row][col]
			if next == rtk.model.buf[row][col] {
				reposition = true
				continue
			}
			rtk.model.buf[row][col] = next
			if reposition {
				rtk.outBuf.WriteString(tparm(cup, row, col))
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
					rtk.outBuf.WriteString("\x1b[39m")
				case 1:
					setaf := rtk.info.Strings["setaf"]
					rtk.outBuf.WriteString(tparm(setaf, int(ps[0])))
				case 3:
					setfrgb := rtk.info.Strings["setfrgb"]
					out := tparm(setfrgb, int(ps[0]), int(ps[1]), int(ps[2]))
					rtk.outBuf.WriteString(out)
				}
			}

			if bg != next.Background {
				bg = next.Background
				ps := bg.Params()
				switch len(ps) {
				case 0:
					rtk.outBuf.WriteString("\x1b[49m")
				case 1:
					setab := rtk.info.Strings["setab"]
					rtk.outBuf.WriteString(tparm(setab, int(ps[0])))
				case 3:
					setbrgb := rtk.info.Strings["setbrgb"]
					out := tparm(setbrgb, int(ps[0]), int(ps[1]), int(ps[2]))
					rtk.outBuf.WriteString(out)
				}
			}

			if attr != next.Attribute {
				// find the ones that have changed
				dAttr := attr ^ next.Attribute
				// If the bit is changed and in next, it was
				// turned on
				on := dAttr & next.Attribute

				if on&AttrBold != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["bold"])
				}
				if on&AttrDim != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["dim"])
				}
				if on&AttrItalic != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["sitm"])
				}
				if on&AttrUnderline != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["smul"])
				}
				if on&AttrBlink != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["blink"])
				}
				if on&AttrReverse != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["rev"])
				}
				if on&AttrInvisible != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["invis"])
				}
				if on&AttrStrikethrough != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["smxx"])
				}

				// If the bit is changed and is in previous, it
				// was turned off
				off := dAttr & attr
				if off&AttrBold != 0 {
					// Normal intensity isn't in terminfo
					rtk.outBuf.WriteString("\x1B[22m")
					// Normal intensity turns off dim. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrDim != 0 {
						rtk.outBuf.WriteString(rtk.info.Strings["dim"])
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					rtk.outBuf.WriteString("\x1B[22m")
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						rtk.outBuf.WriteString(rtk.info.Strings["bold"])
					}
				}
				if off&AttrItalic != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["ritm"])
				}
				if off&AttrUnderline != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["rmul"])
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					rtk.outBuf.WriteString("\x1B[25m")
				}
				if off&AttrReverse != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["rmso"])
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					rtk.outBuf.WriteString("\x1B[28m")
				}
				if off&AttrStrikethrough != 0 {
					rtk.outBuf.WriteString(rtk.info.Strings["rmxx"])
				}
				attr = next.Attribute
			}
			rtk.outBuf.WriteString(next.EGC)
		}
	}
	if rtk.outBuf.Len() != 0 {
		rtk.outBuf.WriteString(rtk.info.Strings["sgr0"])
	}
	return rtk.outBuf.String()
}

func (rtk *RTK) handleSequence(seq ansi.Sequence) {
	log.Tracef("%s", seq)
	switch seq := seq.(type) {
	case ansi.Print:
		var key Key
		switch {
		case rtk.ss3:
			rtk.ss3 = false
			// TODO
			// key.codepoint = ??
		default:
			key.Codepoint = rune(seq)
		}
		rtk.PostMsg(key)
	case ansi.C0:
		key := Key{Codepoint: rune(seq)}
		rtk.PostMsg(key)
	case ansi.ESC:
		if seq.Final == 'O' {
			rtk.ss3 = true
		}
	case ansi.CSI:
		switch seq.Final {
		case 'R':
			// This could be an F1 key, we need to buffer if we have
			// requested a DSRCPR
			if rtk.dsrcpr {
				rtk.dsrcpr = false
				if len(seq.Parameters) != 2 {
					log.Errorf("not enough DSRCPR params")
					return
				}
				rtk.chDSRCPR <- seq.Parameters[0]
				rtk.chDSRCPR <- seq.Parameters[1]
				return
			}
		case 'y':
			// DECRPM - DEC Report Mode
			if len(seq.Parameters) < 1 {
				log.Errorf("not enough DECRPM params")
				return
			}
			switch seq.Parameters[0] {
			case 2026:
				if len(seq.Parameters) < 2 {
					log.Errorf("not enough DECRPM params")
					return
				}
				switch seq.Parameters[1] {
				case 1, 2:
					log.Debugf("Synchronized Update Mode supported")
					rtk.caps.SUM = true
				}
			}
		}
	default:
	}
}

func (rtk *RTK) sendQueries() {
	// XTVERSION
	xtversion := "\x1b[>0q"
	rtk.tty.WriteString(xtversion)

	// Kitty keyboard protocol
	kittyKBD := "\x1b[?u"
	rtk.tty.WriteString(kittyKBD)

	// Synchronized Update Mode
	sumquery := "\x1b[?2026$p"
	rtk.tty.WriteString(sumquery)
}

// Terminal controls

// Enter the alternate screen (for fullscreen applications)
func (rtk *RTK) EnterAltScreen() {
	smcup := rtk.info.Strings["smcup"]
	rtk.tty.WriteString(smcup)
}

func (rtk *RTK) ExitAltScreen() {
	smcup := rtk.info.Strings["rmcup"]
	rtk.tty.WriteString(smcup)
}

// Clear the screen. Issues an actual 'clear' to the controlling terminal, and
// clears the model
func (rtk *RTK) Clear() {
	Clear(rtk.model)
	clear := rtk.info.Strings["clear"]
	rtk.tty.WriteString(clear)
}

func (rtk *RTK) HideCursor() {
	civis := rtk.info.Strings["civis"]
	rtk.tty.WriteString(civis)
}

func (rtk *RTK) ShowCursor(col int, row int) {
	cup := rtk.info.Strings["cup"]
	rtk.tty.WriteString(tparm(cup, row, col))
	cvvis := rtk.info.Strings["cvvis"]
	rtk.tty.WriteString(cvvis)
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func (rtk *RTK) CursorPosition() (col int, row int) {
	// DSRCPR - reports cursor position
	dsrcpr := "\x1b[6n"
	rtk.dsrcpr = true
	rtk.chDSRCPR = make(chan int)
	rtk.tty.WriteString(dsrcpr)
	row = <-rtk.chDSRCPR
	col = <-rtk.chDSRCPR
	close(rtk.chDSRCPR)
	log.Debugf("row=%d col=%d", row, col)
	return col - 1, row - 1
}
