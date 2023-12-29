package term

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
	"github.com/creack/pty"
	"github.com/rivo/uniseg"
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
	// environment. If not set, xterm-256color will be used
	TERM string

	mu sync.Mutex

	activeScreen  [][]cell
	altScreen     [][]cell
	primaryScreen [][]cell

	charsets charsets
	cursor   cursor
	margin   margin
	mode     mode
	sShift   charset
	tabStop  []column
	// lastCol is a flag indicating we printed in the last col
	lastCol bool

	primaryState cursorState
	altState     cursorState

	window *vaxis.Window

	cmd    *exec.Cmd
	dirty  int32
	parser *ansi.Parser
	pty    *os.File
	rows   int
	cols   int

	eventHandler func(vaxis.Event)
	events       chan vaxis.Event
	focused      int32
}

type cursorState struct {
	charsets charsets
	cursor   cursor
	decawm   bool
	decom    bool
}

type margin struct {
	top    row
	bottom row
	left   column
	right  column
}

func New() *Model {
	m := &Model{
		OSC8: true,
		charsets: charsets{
			designations: map[charsetDesignator]charset{
				g0: ascii,
				g1: ascii,
				g2: ascii,
				g3: ascii,
			},
		},
		mode: mode{
			decawm:  true,
			dectcem: true,
		},
		primaryState: cursorState{
			charsets: charsets{
				designations: map[charsetDesignator]charset{
					g0: ascii,
					g1: ascii,
					g2: ascii,
					g3: ascii,
				},
			},
			decawm: true,
		},
		altState: cursorState{
			charsets: charsets{
				designations: map[charsetDesignator]charset{
					g0: ascii,
					g1: ascii,
					g2: ascii,
					g3: ascii,
				},
			},
			decawm: true,
		},
		eventHandler: func(ev vaxis.Event) {},
		// Buffering to 2 events. If there is ever a case where one
		// sequence can trigger two events, this should be increased
		events: make(chan vaxis.Event, 2),
	}
	m.setDefaultTabStops()
	return m
}

// Start starts the terminal with the specified command. Start returns when the
// command has been successfully started.
func (vt *Model) Start(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("no command to run")
	}
	vt.cmd = cmd

	if vt.TERM == "" {
		vt.TERM = "xterm-256color"
	}

	env := os.Environ()
	if cmd.Env != nil {
		env = cmd.Env
	}
	cmd.Env = append(env, "TERM="+vt.TERM)

	// Start the command with a pty.
	var err error
	winsize := pty.Winsize{
		Cols: 80,
		Rows: 24,
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

	vt.Resize(int(winsize.Cols), int(winsize.Rows))
	vt.parser = ansi.NewParser(vt.pty)
	tick := time.NewTicker(8 * time.Millisecond)
	go func() {
		defer vt.recover()
		for {
			select {
			case seq := <-vt.parser.Next():
				switch seq := seq.(type) {
				case ansi.EOF:
					err := cmd.Wait()
					vt.eventHandler(EventClosed{
						Term:  vt,
						Error: err,
					})
					return
				default:
					vt.update(seq)
				}
			case ev := <-vt.events:
				vt.eventHandler(ev)
			case <-tick.C:
				if atomicLoad(&vt.dirty) {
					vt.eventHandler(vaxis.Redraw{})
				}
			}
		}
	}()
	return nil
}

// Update is called from the host application. This is user input
func (vt *Model) Update(msg vaxis.Event) {
	defer atomicStore(&vt.dirty, true)
	switch msg := msg.(type) {
	case vaxis.Key:
		str := encodeXterm(msg, vt.mode.deckpam, vt.mode.decckm)
		vt.pty.WriteString(str)
	case vaxis.PasteStartEvent:
		if vt.mode.paste {
			vt.pty.WriteString("\x1B[200~")
			return
		}
	case vaxis.PasteEndEvent:
		if vt.mode.paste {
			vt.pty.WriteString("\x1B[201~")
			return
		}
	case vaxis.Mouse:
		mouse := vt.handleMouse(msg)
		vt.pty.WriteString(mouse)
		return
	}
}

// update is called from the PTY routine...this is updating the internal model
// based on the underlying process
func (vt *Model) update(seq ansi.Sequence) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	defer atomicStore(&vt.dirty, true)
	switch seq := seq.(type) {
	case ansi.Print:
		vt.print(string(seq))
	case ansi.C0:
		vt.c0(rune(seq))
	case ansi.ESC:
		esc := append(seq.Intermediate, seq.Final)
		vt.esc(string(esc))
	case ansi.CSI:
		csi := append(seq.Intermediate, seq.Final)
		vt.csi(string(csi), seq.Parameters)
	case ansi.OSC:
		vt.osc(string(seq.Payload))
	case ansi.DCS:
	}
}

func (vt *Model) String() string {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	str := strings.Builder{}
	for row := range vt.activeScreen {
		for col := range vt.activeScreen[row] {
			_, _ = str.WriteString(vt.activeScreen[row][col].rune())
		}
		if row < vt.height()-1 {
			str.WriteRune('\n')
		}
	}
	return str.String()
}

func (vt *Model) postEvent(ev vaxis.Event) {
	vt.events <- ev
}

func (vt *Model) Attach(fn func(ev vaxis.Event)) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.eventHandler = fn
}

func (vt *Model) Detach() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.eventHandler = func(ev vaxis.Event) {}
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
	primary := vt.primaryScreen
	vt.altScreen = make([][]cell, h)
	vt.primaryScreen = make([][]cell, h)
	for i := range vt.altScreen {
		vt.altScreen[i] = make([]cell, w)
		vt.primaryScreen[i] = make([]cell, w)
	}
	last := vt.cursor.row
	vt.margin.bottom = row(h) - 1
	vt.margin.right = column(w) - 1
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false
	vt.activeScreen = vt.primaryScreen

	// transfer primary to new, skipping the last row
	for row := 0; row < len(primary); row += 1 {
		if row == int(last) {
			break
		}
		wrapped := false
		for col := 0; col < len(primary[0]); col += 1 {
			cell := primary[row][col]
			vt.cursor.attrs = cell.attrs
			vt.print(cell.content)
			wrapped = cell.wrapped
		}
		if !wrapped {
			vt.nel()
		}
	}
	switch vt.mode.smcup {
	case true:
		vt.activeScreen = vt.primaryScreen
	default:
		vt.activeScreen = vt.altScreen
	}

	_ = pty.Setsize(vt.pty, &pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	})
}

func (vt *Model) width() int {
	if len(vt.activeScreen) > 0 {
		return len(vt.activeScreen[0])
	}
	return 0
}

func (vt *Model) height() int {
	return len(vt.activeScreen)
}

// print sets the current cell contents to the given rune. The attributes will
// be copied from the current cursor attributes
func (vt *Model) print(r string) {
	// TODO fix this for change to string
	// if vt.charsets.designations[vt.charsets.selected] == decSpecialAndLineDrawing {
	// 	shifted, ok := decSpecial[r]
	// 	if ok {
	// 		r = shifted
	// 	}
	// }

	// If we are single-shifted, move the previous charset into the current
	if vt.charsets.singleShift {
		vt.charsets.selected = vt.charsets.saved
	}

	if vt.cursor.col == vt.margin.right && vt.lastCol {
		col := vt.cursor.col
		rw := vt.cursor.row
		vt.activeScreen[rw][col].wrapped = true
		vt.nel()
	}

	col := vt.cursor.col
	rw := vt.cursor.row
	w := uniseg.StringWidth(r)

	if vt.mode.irm {
		line := vt.activeScreen[rw]
		for i := vt.margin.right; i > col; i -= 1 {
			line[i] = line[i-column(w)]
		}
	}
	if col > column(vt.width())-1 {
		col = column(vt.width()) - 1
	}
	if rw > row(vt.height()-1) {
		rw = row(vt.height() - 1)
	}

	if w == 0 {
		if col-1 < 0 {
			return
		}
		// vt.activeScreen[rw][col-1].combining = append(vt.activeScreen[rw][col-1].combining, r)
		return
	}
	cell := cell{
		content: r,
		width:   w,
		fg:      vt.cursor.fg,
		bg:      vt.cursor.bg,
		attrs:   vt.cursor.attrs,
	}

	vt.activeScreen[rw][col] = cell

	// Set trailing cells to a space if wide rune
	for i := column(1); i < column(w); i += 1 {
		if col+i > vt.margin.right {
			break
		}
		vt.activeScreen[rw][col+i].content = " "
		vt.activeScreen[rw][col+i].attrs = vt.cursor.attrs
	}

	switch {
	case vt.mode.decawm && col == vt.margin.right:
		vt.lastCol = true
	case col == vt.margin.right:
		// don't move the cursor
	default:
		vt.cursor.col += column(w)
	}
}

// scrollUp shifts all text upward by n rows. Semantically, this is backwards -
// usually scroll up would mean you shift rows down
func (vt *Model) scrollUp(n int) {
	for row := range vt.activeScreen {
		if row > int(vt.margin.bottom) {
			continue
		}
		if row < int(vt.margin.top) {
			continue
		}
		if row+n > int(vt.margin.bottom) {
			for col := vt.margin.left; col <= vt.margin.right; col += 1 {
				vt.activeScreen[row][col].erase(vt.cursor.bg)
			}
			continue
		}
		copy(vt.activeScreen[row], vt.activeScreen[row+n])
	}
}

// scrollDown shifts all lines down by n rows.
func (vt *Model) scrollDown(n int) {
	for r := vt.margin.bottom; r >= vt.margin.top; r -= 1 {
		if r-row(n) < vt.margin.top {
			for col := vt.margin.left; col <= vt.margin.right; col += 1 {
				vt.activeScreen[r][col].erase(vt.cursor.bg)
			}
			continue
		}
		copy(vt.activeScreen[r], vt.activeScreen[r-row(n)])
	}
}

func (vt *Model) Close() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if vt.cmd != nil && vt.cmd.Process != nil {
		vt.cmd.Process.Kill()
		vt.cmd.Wait()
	}
	vt.pty.Close()
}

func (vt *Model) Draw(win vaxis.Window) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	defer atomicStore(&vt.dirty, false)
	width, height := win.Size()
	if int(width) != vt.width() || int(height) != vt.height() {
		win.Width = width
		win.Height = height
		vt.Resize(width, height)
	}
	vt.window = &win
	for row := 0; row < vt.height(); row += 1 {
		for col := 0; col < vt.width(); {
			cell := vt.activeScreen[row][col]
			w := cell.width

			if cell.content == "" {
				cell.content = " "
			}
			vcell := vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: cell.content,
					Width:    cell.width,
				},
				Style: vaxis.Style{
					Foreground:      cell.fg,
					Background:      cell.bg,
					Attribute:       cell.attrs,
					Hyperlink:       cell.url,
					HyperlinkParams: "id=%s" + cell.urlId,
				},
			}

			win.SetCell(col, row, vcell)
			if w == 0 {
				w = 1
			}
			col += w
		}
	}
	if vt.mode.dectcem && atomicLoad(&vt.focused) {
		win.ShowCursor(int(vt.cursor.col), int(vt.cursor.row), vt.cursor.style)
	}
	// for _, s := range buf.getVisibleSixels() {
	// 	fmt.Printf("\033[%d;%dH", s.Sixel.Y, s.Sixel.X)
	// 	// DECSIXEL Introducer(\033P0;0;8q) + DECGRA ("1;1): Set Raster Attributes
	// 	os.Stdout.Write([]byte{0x1b, 0x50, 0x30, 0x3b, 0x30, 0x3b, 0x38, 0x71, 0x22, 0x31, 0x3b, 0x31})
	// 	os.Stdout.Write(s.Sixel.Data)
	// 	// string terminator(ST)
	// 	os.Stdout.Write([]byte{0x1b, 0x5c})
	// }
}

func (vt *Model) Focus() {
	atomicStore(&vt.focused, true)
}

func (vt *Model) Blur() {
	atomicStore(&vt.focused, false)
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
