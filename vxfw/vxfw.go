package vxfw

import (
	"math"
	"sort"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

type Widget interface {
	HandleEvent(vaxis.Event, EventPhase) (Command, error)
	Draw(DrawContext) (Surface, error)
}

type Command interface{}

type (
	RedrawCmd struct{}
	QuitCmd   struct{}
	BatchCmd  []Command
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
	// Current focused widget
	widget Widget
}

type App struct {
	vx         *vaxis.Vaxis
	redraw     bool
	shouldQuit bool

	charCache map[string]int

	focus focusHandler
}

func NewApp() (*App, error) {
	vx, err := vaxis.New(vaxis.Options{})
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

	// Set root as the current focus
	a.focus.widget = w

	var lastRedraw time.Time
	// This is the main event loop. We first wait for events with an 8ms
	// timeout. If we have an event, we handle it immediately and process
	// any commands it returns.
	//
	// Then we check if we should quit
	//
	// Then, if we need a redraw and it has been more than 8ms since our
	// last redraw, we redraw.
	for {
		select {
		case ev := <-a.vx.Events():
			cmd, err := w.HandleEvent(ev, TargetPhase)
			if err != nil {
				return err
			}
			a.handleCommand(cmd)
		case <-time.After(8 * time.Millisecond):
		}

		if a.shouldQuit {
			return nil
		}

		now := time.Now()
		if !a.redraw && now.Sub(lastRedraw) < 8*time.Millisecond {
			continue
		}

		lastRedraw = now
		a.redraw = false

		win := a.vx.Window()
		min := Size{Width: 0, Height: 0}
		max := Size{Width: uint16(win.Width), Height: uint16(win.Height)}
		s, err := w.Draw(DrawContext{
			Min:        min,
			Max:        max,
			Characters: a.Characters,
		})
		if err != nil {
			return err
		}

		s.render(win, a.focus.widget)
		a.vx.Render()
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
