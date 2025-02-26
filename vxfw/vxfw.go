package vxfw

import (
	"math"
	"slices"
	"sort"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/log"
)

type Widget interface {
	HandleEvent(vaxis.Event, EventPhase) (Command, error)
	Draw(DrawContext) (Surface, error)
}

// EventCapturer is a Widget which can capture events before they are delivered
// to the target widget. To capture an event, the EventCapturer must be an
// ancestor of the target widget
type EventCapturer interface {
	CaptureEvent(vaxis.Event) (Command, error)
}

type Event interface{}

type (
	// Sent as the first event to the root widget
	Init struct{}
)

type Command interface{}

type (
	// RedrawCmd tells the UI to redraw
	RedrawCmd struct{}
	// QuitCmd tells the application to exit
	QuitCmd struct{}
	// ConsumeEventCmd tells the application to stop the event propagation
	ConsumeEventCmd struct{}
	// BatchCmd is a batch of other commands
	BatchCmd []Command
	// SyncFuncCmd is a function which will be run in the main goroutine
	SyncFuncCmd func()
	// Sets the focus to this Widget
	FocusWidgetCmd Widget
)

type DrawContext struct {
	// The minimum size the widget must render as
	Min Size
	// The maximum size the widget must render as. A value of math.MaxUint16
	// in either dimension means that dimension has no limit
	Max Size
	// Function to turn a string into a slice of characters. This splits the
	// string into graphemes and measures each grapheme
	Characters func(string) []vaxis.Character
}

type Size struct {
	Width  uint16
	Height uint16
}

func (s Size) UnboundedWidth() bool {
	return s.Width == math.MaxUint16
}

func (s Size) UnboundedHeight() bool {
	return s.Height == math.MaxUint16
}

// EventPhase is the phase of the event during the event handling process.
// Possible values are
//
//	CapturePhase
//	TargetPhase
//	BubblePhase
type EventPhase uint8

const (
	CapturePhase EventPhase = iota
	TargetPhase
	BubblePhase
)

type Surface struct {
	Size     Size
	Widget   Widget
	Cursor   *CursorState
	Buffer   []vaxis.Cell
	Children []SubSurface
}

// Creates a new surface. The resulting surface will have a Buffer with capacity
// large enough for Size
func NewSurface(width uint16, height uint16, w Widget) Surface {
	return Surface{
		Size: Size{
			Width:  width,
			Height: height,
		},
		Widget: w,
		Buffer: make([]vaxis.Cell, 0, height*width),
	}
}

func (s *Surface) AddChild(col int, row int, child Surface) {
	ss := NewSubSurface(col, row, child)
	s.Children = append(s.Children, ss)
}

func (s Surface) render(win vaxis.Window, focused Widget) {
	// Render ourself first
	for i, cell := range s.Buffer {
		row := i / int(s.Size.Width)
		col := i % int(s.Size.Width)
		win.SetCell(col, row, cell)
	}

	// If we have a cursor state and we are the focused widget, draw the
	// cursor
	if s.Cursor != nil && s.Widget == focused {
		win.ShowCursor(
			int(s.Cursor.Col),
			int(s.Cursor.Row),
			s.Cursor.Shape,
		)
	}

	// Sort the Children by z-index
	sort.Slice(s.Children, func(i int, j int) bool {
		return s.Children[i].ZIndex < s.Children[j].ZIndex
	})

	for _, child := range s.Children {
		childWin := win.New(
			int(child.Origin.Col),
			int(child.Origin.Row),
			int(child.Surface.Size.Width),
			int(child.Surface.Size.Height),
		)
		child.Surface.render(childWin, focused)
	}
}

type CursorState struct {
	Row   uint16
	Col   uint16
	Shape vaxis.CursorStyle
}

type SubSurface struct {
	Origin  RelativePoint
	Surface Surface
	ZIndex  int
}

func NewSubSurface(col int, row int, s Surface) SubSurface {
	return SubSurface{
		Origin: RelativePoint{
			Row: row,
			Col: col,
		},
		Surface: s,
		ZIndex:  0,
	}
}

type RelativePoint struct {
	Row int
	Col int
}

type focusHandler struct {
	// Current focused focused
	focused Widget

	// Root widget
	root Widget

	// path is the path to the focused widet
	path []Widget
}

func (f *focusHandler) handleEvent(app *App, ev vaxis.Event) error {
	app.consumeEvent = false

	// Capture phase
	for _, w := range f.path {
		c, ok := w.(EventCapturer)
		if !ok {
			continue
		}
		cmd, err := c.CaptureEvent(ev)
		if err != nil {
			return err
		}
		app.handleCommand(cmd)
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	// Target phase
	cmd, err := f.focused.HandleEvent(ev, TargetPhase)
	if err != nil {
		return err
	}
	app.handleCommand(cmd)
	if app.consumeEvent {
		app.consumeEvent = false
		return nil
	}

	// Bubble phase. We don't bubble to the focused widget (which is the
	// last one in the list). Hence, - 2
	for i := len(f.path) - 2; i >= 0; i -= 1 {
		w := f.path[i]
		cmd, err := w.HandleEvent(ev, BubblePhase)
		if err != nil {
			return err
		}
		app.handleCommand(cmd)
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	return nil
}

func (f *focusHandler) updatePath(root Surface) {
	// Clear the path
	f.path = []Widget{}

	ok := f.childHasFocus(root)
	if !ok {
		panic("focused widget not found in Surface tree")
	}

	if f.root != root.Widget {
		// Make sure that we always add the original root widget as the
		// last node. We will reverse the list, making this widget the
		// first one with the opportunity to capture events
		f.path = append(f.path, f.root)
	}

	// Reverse the list since it is ordered target to root, and we want the
	// opposite
	slices.Reverse(f.path)
}

func (f *focusHandler) childHasFocus(s Surface) bool {
	// If s is our focused widget, we add to path and return true
	if s.Widget == f.focused {
		f.path = append(f.path, s.Widget)
		return true
	}

	// Loop through children to find the focused widget
	for _, c := range s.Children {
		if !f.childHasFocus(c.Surface) {
			continue
		}
		f.path = append(f.path, s.Widget)
		return true
	}

	return false
}

func (f *focusHandler) focusWidget(app *App, w Widget) error {
	if f.focused == w {
		return nil
	}

	cmd, err := f.focused.HandleEvent(vaxis.FocusOut{}, TargetPhase)
	if err != nil {
		return err
	}
	app.handleCommand(cmd)
	cmd, err = w.HandleEvent(vaxis.FocusIn{}, TargetPhase)
	if err != nil {
		return err
	}
	app.handleCommand(cmd)

	return nil
}

type App struct {
	vx *vaxis.Vaxis

	redraw       bool
	shouldQuit   bool
	consumeEvent bool

	charCache map[string]int

	focus focusHandler
}

func NewApp() (*App, error) {
	vx, err := vaxis.New(vaxis.Options{
		CSIuBitMask: vaxis.CSIuDisambiguate |
			vaxis.CSIuReportEvents |
			vaxis.CSIuAlternateKeys |
			vaxis.CSIuAllKeys |
			vaxis.CSIuAlternateKeys,
	})
	if err != nil {
		return nil, err
	}
	app := &App{
		vx:        vx,
		charCache: make(map[string]int, 256),
	}
	return app, nil
}

// Run the application
func (a *App) Run(w Widget) error {
	defer a.vx.Close()

	// Initialize the focus handler. Our root, focused, and first node of
	// the path is the root widget at init
	a.focus = focusHandler{
		root:    w,
		focused: w,
		path:    []Widget{w},
	}

	err := a.focus.handleEvent(a, Init{})
	if err != nil {
		return err
	}

	// This is the main event loop. We first wait for events with an 8ms
	// timeout. If we have an event, we handle it immediately and process
	// any commands it returns.
	//
	// Then we check if we should quit
	for {
		select {
		case ev := <-a.vx.Events():
			err := a.focus.handleEvent(a, ev)
			if err != nil {
				return err
			}
			if a.shouldQuit {
				return nil
			}
		case <-time.After(8 * time.Millisecond):
			if !a.redraw {
				continue
			}
			a.redraw = false

			win := a.vx.Window()
			min := Size{Width: 0, Height: 0}
			max := Size{
				Width:  uint16(win.Width),
				Height: uint16(win.Height),
			}
			s, err := w.Draw(DrawContext{
				Min:        min,
				Max:        max,
				Characters: a.Characters,
			})
			if err != nil {
				return err
			}

			win.Clear()
			s.render(win, a.focus.focused)
			a.vx.Render()

			// Update focus handler
			a.focus.updatePath(s)
		}

	}
}

func (a *App) handleCommand(cmd Command) {
	switch cmd := cmd.(type) {
	case BatchCmd:
		for _, c := range cmd {
			a.handleCommand(c)
		}
	case []Command:
		for _, c := range cmd {
			a.handleCommand(c)
		}
	case RedrawCmd:
		a.redraw = true
	case QuitCmd:
		a.shouldQuit = true
	case ConsumeEventCmd:
		a.consumeEvent = true
	case SyncFuncCmd:
		cmd()
	case FocusWidgetCmd:
		err := a.focus.focusWidget(a, cmd)
		if err != nil {
			log.Error("focusWidget error: %s", err)
			return
		}

	}
}

func (a App) PostEvent(ev vaxis.Event) {
	a.vx.PostEvent(ev)
}

// Characters turns a string into a slice of measured graphemes
func (a *App) Characters(s string) []vaxis.Character {
	chars := vaxis.Characters(s)
	if !a.vx.CanUnicodeCore() {
		// If we don't have unicode core, we need to remeasure
		// everything. We cache the results
		for i := range chars {
			g := chars[i].Grapheme
			w, ok := a.charCache[g]
			if !ok {
				// Put the result in the cache
				w = a.vx.RenderedWidth(g)
				a.charCache[g] = w
			}
			chars[i].Width = w
		}
	}

	return chars
}
