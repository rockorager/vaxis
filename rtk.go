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
		// RGB support was detected in some way
		RGB bool
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

	err := setupTermInfo()
	if err != nil {
		return nil, err
	}

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
		rtk.caps.RGB = true
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
			if rtk.outBuf.Len() == 0 && rtk.caps.SUM {
				rtk.outBuf.WriteString(sumSet)
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
					rtk.outBuf.WriteString(fgop)
				case 1:
					rtk.outBuf.WriteString(tparm(setaf, int(ps[0])))
				case 3:
					out := tparm(setrgbf, int(ps[0]), int(ps[1]), int(ps[2]))
					rtk.outBuf.WriteString(out)
				}
			}

			if bg != next.Background {
				bg = next.Background
				ps := bg.Params()
				switch len(ps) {
				case 0:
					rtk.outBuf.WriteString(bgop)
				case 1:
					rtk.outBuf.WriteString(tparm(setab, int(ps[0])))
				case 3:
					out := tparm(setrgbb, int(ps[0]), int(ps[1]), int(ps[2]))
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
					rtk.outBuf.WriteString(bold)
				}
				if on&AttrDim != 0 {
					rtk.outBuf.WriteString(dim)
				}
				if on&AttrItalic != 0 {
					rtk.outBuf.WriteString(sitm)
				}
				if on&AttrUnderline != 0 {
					rtk.outBuf.WriteString(smul)
				}
				if on&AttrBlink != 0 {
					rtk.outBuf.WriteString(blink)
				}
				if on&AttrReverse != 0 {
					rtk.outBuf.WriteString(rev)
				}
				if on&AttrInvisible != 0 {
					rtk.outBuf.WriteString(invis)
				}
				if on&AttrStrikethrough != 0 {
					rtk.outBuf.WriteString(smxx)
				}

				// If the bit is changed and is in previous, it
				// was turned off
				off := dAttr & attr
				if off&AttrBold != 0 {
					// Normal intensity isn't in terminfo
					rtk.outBuf.WriteString(boldDimReset)
					// Normal intensity turns off dim. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrDim != 0 {
						rtk.outBuf.WriteString(dim)
					}
				}
				if off&AttrDim != 0 {
					// Normal intensity isn't in terminfo
					rtk.outBuf.WriteString(boldDimReset)
					// Normal intensity turns off bold. If it
					// should be on, let's turn it back on
					if next.Attribute&AttrBold != 0 {
						rtk.outBuf.WriteString(bold)
					}
				}
				if off&AttrItalic != 0 {
					rtk.outBuf.WriteString(ritm)
				}
				if off&AttrUnderline != 0 {
					rtk.outBuf.WriteString(rmul)
				}
				if off&AttrBlink != 0 {
					// turn off blink isn't in terminfo
					rtk.outBuf.WriteString(endBlink)
				}
				if off&AttrReverse != 0 {
					rtk.outBuf.WriteString(rmso)
				}
				if off&AttrInvisible != 0 {
					// turn off invisible isn't in terminfo
					rtk.outBuf.WriteString(endInvis)
				}
				if off&AttrStrikethrough != 0 {
					rtk.outBuf.WriteString(rmxx)
				}
				attr = next.Attribute
			}
			rtk.outBuf.WriteString(next.EGC)
		}
	}
	if rtk.outBuf.Len() != 0 {
		rtk.outBuf.WriteString(sgr0)
		if rtk.caps.SUM {
			rtk.outBuf.WriteString(sumReset)
		}
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
	rtk.tty.WriteString(xtversion)
	rtk.tty.WriteString(kkbpQuery)
	rtk.tty.WriteString(sumQuery)
}

// Terminal controls

// Enter the alternate screen (for fullscreen applications)
func (rtk *RTK) EnterAltScreen() {
	rtk.tty.WriteString(smcup)
}

func (rtk *RTK) ExitAltScreen() {
	rtk.tty.WriteString(rmcup)
}

// Clear the screen. Issues an actual 'clear' to the controlling terminal, and
// clears the model
func (rtk *RTK) Clear() {
	Clear(rtk.model)
	rtk.tty.WriteString(clear)
}

func (rtk *RTK) HideCursor() {
	rtk.tty.WriteString(civis)
}

func (rtk *RTK) ShowCursor(col int, row int) {
	rtk.tty.WriteString(tparm(cup, row, col))
	rtk.tty.WriteString(cvvis)
}

// Reports the current cursor position. 0,0 is the upper left corner. Reports
// -1,-1 if the query times out or fails
func (rtk *RTK) CursorPosition() (col int, row int) {
	// DSRCPR - reports cursor position
	rtk.dsrcpr = true
	rtk.chDSRCPR = make(chan int)
	defer close(rtk.chDSRCPR)
	rtk.tty.WriteString(dsrcpr)
	timeout := time.NewTimer(10 * time.Millisecond)
	select {
	case <-timeout.C:
		log.Warnf("CursorPosition timed out")
		return -1, -1
	case row = <-rtk.chDSRCPR:
		// if we get one, we'll get another
		col = <-rtk.chDSRCPR
		return col - 1, row - 1
	}
}
