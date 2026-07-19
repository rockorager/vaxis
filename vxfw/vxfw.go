package vxfw

import (
	"image"
	"math"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/log"
)

type Widget interface {
	Draw(DrawContext) (Surface, error)
}

// EventCapturer is a Widget which can capture events before they are delivered
// to the focused widget. To capture an event, the EventCapturer must be an
// ancestor of the target widget
//
// The event may represent a [Command] sent from propagation or postage.
//
// This phase is commonly useful for receiving any events such as custom binds
// that must be handled before being sent to any other widget (eg. [QuitCmd])
type EventCapturer interface {
	CaptureEvent(vaxis.Event) (Command, error)
}

// EventPhase is the phase of the event during the event handling
// process.
type EventPhase uint8

const (
	// Walking the focus path, all widgets in the path to the focused
	// widget will be called with [CaptureEvent], before the focused
	// widget is called with this phase
	TargetPhase EventPhase = iota

	// If the target widget has not consumed the event with [ConsumeEventCmd],
	// the event will be sent to widgets in the focus path with this phase,
	// walking backwards from the focused widget that might consume it instead
	//
	// This phase should be used for implementing any event handling that does
	// not require immediate propagation, such as implementing custom binds after
	// a focused widget (eg. textfield) had not consumed it
	BubblePhase
)

// EventHandler is a Widget which can handle events directly when focused.
// It's a separate interface to simplify creating custom [Widget]s that do not
// require capturing events
//
// The event may represent a [Command] sent from propagation or postage.
type EventHandler interface {
	HandleEvent(vaxis.Event, EventPhase) (Command, error)
}

// Unwrapper is a indicator of a Widget that wraps around a single widget and
// doesn't expect to handle or forward any events on its own. It's main
// usecase is to help resolve the correct focus order of the widget, used in
// widgets that just show a single widget with decoration
type Unwrapper interface {
	Unwrap() Widget
}

type (
	// Sent as the first event to the root widget.
	Init struct{}

	MouseEnter struct{}
	MouseLeave struct{}
)

// Command represents any type returned from event propagation
// or a PostEvent call. See [EventPhase], [EventCapturer] [EventHandler].
type Command any

type (
	// RedrawCmd tells the UI to redraw
	RedrawCmd struct{}
	// RefreshCmd tells the UI to flush a complete redraw
	RefreshCmd struct{}
	// QuitCmd tells the application to exit
	QuitCmd struct{}
	// ConsumeEventCmd indicates that this [EventCapturer] has captured
	// the event successfully, stopping event propagation
	// If returning a batch of events, ensure this is placed last
	ConsumeEventCmd struct{}
	// BatchCmd is a batch of other commands
	BatchCmd []Command
	// FocusWidgetCmd sets the focus to the widget
	FocusWidgetCmd Widget
	// SetMouseShapeCmd sets the mouse shape
	SetMouseShapeCmd vaxis.MouseShape
	// SetTitleCmd sets the title of the terminal
	SetTitleCmd string
	// CopyToClipboard copies the provided string to the host clipboard
	CopyToClipboardCmd string
	// SendNotificationCmd sends a system notification
	SendNotificationCmd struct {
		Title string
		Body  string
	}
	// DebugCmd tells the runtime to print the Surface tree each render
	DebugCmd struct{}
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

// WithConstraints returns a new DrawContext with the supplied min and max size
// Use [DrawContext.WithMin] or [DrawContext.WithMax] to change one constraint
func (ctx DrawContext) WithConstraints(min, max Size) DrawContext {
	return DrawContext{
		Min: min, Max: max,
		Characters: ctx.Characters,
	}
}

// WithMin returns a new DrawContext with the minimum size set to min
func (ctx DrawContext) WithMin(min Size) DrawContext {
	return ctx.WithConstraints(min, ctx.Max)
}

// WithMax returns a new DrawContext with the maximum size set to max
func (ctx DrawContext) WithMax(max Size) DrawContext {
	return ctx.WithConstraints(ctx.Min, max)
}

type Size struct {
	Width  uint16
	Height uint16
}

func (s Size) HasUnboundedWidth() bool {
	return s.Width == math.MaxUint16
}

func (s Size) HasUnboundedHeight() bool {
	return s.Height == math.MaxUint16
}

type Surface struct {
	Size     Size
	Widget   Widget
	Cursor   *CursorState
	Buffer   []vaxis.Cell
	Children []SubSurface
	Render   func(vaxis.Window)
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
		Buffer: make([]vaxis.Cell, height*width),
	}
}

func (s *Surface) AddChild(col int, row int, child Surface) {
	ss := NewSubSurface(col, row, child)
	s.Children = append(s.Children, ss)
}

func (s *Surface) WriteCell(col uint16, row uint16, cell vaxis.Cell) {
	if col >= s.Size.Width ||
		row >= s.Size.Height {
		return
	}
	i := (row * s.Size.Width) + col
	if int(i) >= len(s.Buffer) {
		return
	}
	s.Buffer[i] = cell
}

// FillStyle sets style on all cells in s
func (s *Surface) FillStyle(style vaxis.Style) {
	for i := range s.Buffer {
		s.Buffer[i].Style = style
	}
}

// FillCharacter writes ch to all cells in s
func (s *Surface) FillCharacter(ch vaxis.Character) {
	for i := range s.Buffer {
		s.Buffer[i].Character = ch
	}
}

// Fill writes cell to all cells in s
func (s *Surface) Fill(cell vaxis.Cell) {
	for i := range s.Buffer {
		s.Buffer[i] = cell
	}
}

func (s Surface) render(win vaxis.Window, focused Widget) {
	// Render ourself first
	for i, cell := range s.Buffer {
		row := i / int(s.Size.Width)
		col := i % int(s.Size.Width)
		win.SetCell(col, row, cell)
	}

	if s.Render != nil {
		s.Render(win)
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
		// clip the child window to the minimum of the parent surface or the child surface this
		// effectively forces clipping at the layout level
		w := math.Min(float64(child.Surface.Size.Width), float64(int(s.Size.Width)-child.Origin.Col))
		h := math.Min(float64(child.Surface.Size.Height), float64(int(s.Size.Height)-child.Origin.Row))
		childWin := win.New(
			int(child.Origin.Col),
			int(child.Origin.Row),
			int(w),
			int(h),
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

func (ss *SubSurface) containsPoint(col int, row int) bool {
	return col >= ss.Origin.Col &&
		col < (ss.Origin.Col+int(ss.Surface.Size.Width)) &&
		row >= ss.Origin.Row &&
		row < (ss.Origin.Row+int(ss.Surface.Size.Height))
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
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	// Target phase
	cmd, err := tryHandleEvent(f.focused, ev, TargetPhase)
	if err != nil {
		return err
	}
	if err := app.handleCommand(cmd); err != nil {
		return err
	}
	if app.consumeEvent {
		app.consumeEvent = false
		return nil
	}

	// Bubble phase. We don't bubble to the focused widget (which is the last one in the list).
	// Hence, - 2
	for i := len(f.path) - 2; i >= 0; i -= 1 {
		w := f.path[i]
		cmd, err = tryHandleEvent(w, ev, BubblePhase)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	return nil
}

func (f *focusHandler) updatePath(app *App, root Surface) {
	path, ok := f.childHasFocus(root, f.path[:0])
	if !ok {
		// Resolve focus to the missing widget's parent
		parent := f.root
		if n := len(f.path); n >= 2 {
			parent = f.path[n-2]
		}
		_ = f.focusWidget(app, parent)

		path, ok = f.childHasFocus(root, f.path[:0])
	}
	if !ok {
		// Best effort refocus
		_ = f.focusWidget(app, f.root)
	}

	if f.root != root.Widget || len(path) == 0 {
		// Make sure that we always add the original root widget as the
		// last node. We will reverse the list, making this widget the
		// first one with the opportunity to capture events
		path = append(path, f.root)
	}

	// Reverse the list since it is ordered target to root, and we want the
	// opposite
	for i := 0; i < len(path)/2; i++ {
		path[i], path[len(path)-1-i] = path[len(path)-1-i], path[i]
	}

	f.path = path
}

func (f *focusHandler) childHasFocus(s Surface, path []Widget) ([]Widget, bool) {
	if s.Widget == f.focused {
		return append(path, s.Widget), true
	}

	for _, c := range s.Children {
		p, ok := f.childHasFocus(c.Surface, path)
		if !ok {
			continue
		}
		// A widget that reuses its child's own Widget will be
		// unused in the focus path, as no events will be passed to it.
		if len(p) > 0 && p[len(p)-1] == s.Widget {
			return p, true
		}
		return append(p, s.Widget), true
	}

	return path, false
}

func (f *focusHandler) focusWidget(app *App, w Widget) error {
	for {
		u, ok := w.(Unwrapper)
		if !ok {
			break
		}
		w = u.Unwrap()
	}
	if f.focused == w {
		return nil
	}

	cmd, err := tryHandleEvent(f.focused, vaxis.FocusOut{}, TargetPhase)
	if err != nil {
		return err
	}
	if err := app.handleCommand(cmd); err != nil {
		return err
	}

	// Change the focused widget before we send the focus in event. If the
	// newly focused widget changes focus again, we need to set this before
	// the handleCommand call
	f.focused = w
	cmd, err = tryHandleEvent(w, vaxis.FocusIn{}, TargetPhase)
	if err != nil {
		return err
	}
	return app.handleCommand(cmd)
}

// tryHandleEvent calls HandleEvent on w, if w is an [EventHandler]
// If w is not an EventHandler, tryHandleEvent returns nil, nil.
// Otherwise, tryHandleEvent returns the [Command] and error from HandleEvent.
func tryHandleEvent(w Widget, event vaxis.Event, phase EventPhase) (Command, error) {
	eh, ok := w.(EventHandler)
	if !ok {
		return nil, nil
	}

	return eh.HandleEvent(event, phase)
}

type App struct {
	vx *vaxis.Vaxis

	redraw       bool
	refresh      bool
	shouldQuit   bool
	consumeEvent bool
	debug        bool

	charCache map[string]int
	fh        focusHandler
}

func NewApp(opts vaxis.Options) (*App, error) {
	vx, err := vaxis.New(opts)
	if err != nil {
		return nil, err
	}
	app := &App{
		vx:        vx,
		charCache: make(map[string]int, 256),
	}
	return app, nil
}

func (a *App) Suspend() error {
	return a.vx.Suspend()
}

func (a *App) Resume() error {
	return a.vx.Resume()
}

// NewImage creates a new image using the highest quality renderer the terminal
// is capable of. The returned image can be displayed with the vxfw/image
// widget.
func (a *App) NewImage(img image.Image) (vaxis.Image, error) {
	return a.vx.NewImage(img)
}

// Run the application
func (a *App) Run(w Widget) error {
	defer a.vx.Close()

	// Initialize the focus handler. Our root, focused, and first node of
	// the path is the root widget at init
	a.fh = focusHandler{
		root:    w,
		focused: w,
		path:    []Widget{w},
	}

	err := a.fh.handleEvent(a, Init{})
	if err != nil {
		return err
	}

	s, err := a.layout(w)
	if err != nil {
		return err
	}

	mh := mouseHandler{
		lastFrame: s,
	}

	// This is the main event loop. We first wait for events with an 8ms
	// timeout. If we have an event, we handle it immediately and process
	// any commands it returns.
	//
	// Then we check if we should quit
	for {
		select {
		case ev := <-a.vx.Events():
			switch ev := ev.(type) {
			case vaxis.Resize:
				a.vx.Resize(ev)
				// Trigger a redraw on resize
				a.redraw = true
			case vaxis.Mouse:
				err := mh.handleEvent(a, ev)
				if err != nil {
					return err
				}
			case vaxis.FocusIn:
				cmd, err := tryHandleEvent(w, MouseEnter{}, TargetPhase)
				if err != nil {
					return err
				}
				err = a.handleCommand(cmd)
				if err != nil {
					return err
				}
			case vaxis.FocusOut:
				mh.mouse = nil
				err := mh.mouseExit(a)
				if err != nil {
					return err
				}
			case vaxis.Key:
				err := a.fh.handleEvent(a, ev)
				if err != nil {
					return err
				}
			case vaxis.Redraw:
				a.redraw = true
			default:
				// Anything else we let the application or widgets handle.
				err = a.handleCommand(ev)
				if err != nil {
					return err
				}
			}
			if a.shouldQuit {
				return nil
			}
		case <-time.After(8 * time.Millisecond):
			if !a.redraw {
				continue
			}
			a.redraw = false

			s, err := a.layout(w)
			if err != nil {
				return err
			}

			// Update mouse
			err = mh.update(a, s)
			if err != nil {
				return err
			}

			// mh.update can trigger a redraw based on mouse enter /
			// mouse exit events. check and redo the layout if
			// needed
			if a.redraw {
				a.redraw = false
				s, err = a.layout(w)
				if err != nil {
					return err
				}
			}

			win := a.vx.Window()
			win.Clear()
			a.vx.HideCursor()
			s.render(win, a.fh.focused)

			switch a.refresh {
			case true:
				a.vx.Refresh()
				a.refresh = false
			case false:
				a.vx.Render()
			}

			if a.debug {
				debugPrintWidget(s, 0, a.fh.focused)
				a.debug = false
			}

			// Update focus handler
			a.fh.updatePath(a, s)
			// Update the mouse last frame
			mh.lastFrame = s
		}
	}
}

func (a *App) layout(root Widget) (Surface, error) {
	win := a.vx.Window()
	min := Size{Width: 0, Height: 0}
	max := Size{
		Width:  uint16(win.Width),
		Height: uint16(win.Height),
	}
	return root.Draw(DrawContext{
		Min:        min,
		Max:        max,
		Characters: a.Characters,
	})
}

func (a *App) handleCommand(cmd Command) error {
	switch cmd := cmd.(type) {
	case nil:
	case BatchCmd:
		for _, c := range cmd {
			if err := a.handleCommand(c); err != nil {
				return err
			}
		}
	case []Command:
		for _, c := range cmd {
			if err := a.handleCommand(c); err != nil {
				return err
			}
		}
	case RedrawCmd:
		a.redraw = true
	case RefreshCmd:
		a.refresh = true
	case QuitCmd:
		a.shouldQuit = true
	case ConsumeEventCmd:
		a.consumeEvent = true
	case FocusWidgetCmd:
		return a.fh.focusWidget(a, cmd)
	case SetMouseShapeCmd:
		a.vx.SetMouseShape(vaxis.MouseShape(cmd))
	case SetTitleCmd:
		a.vx.SetTitle(string(cmd))
	case CopyToClipboardCmd:
		a.vx.ClipboardPush(string(cmd))
	case SendNotificationCmd:
		a.vx.Notify(cmd.Title, cmd.Body)
	case DebugCmd:
		a.debug = true
		a.redraw = true
	default:
		return a.fh.handleEvent(a, cmd)
	}
	return nil
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

type hitResult struct {
	col uint16
	row uint16
	w   Widget
}

type mouseHandler struct {
	lastFrame Surface
	lastHits  []hitResult
	mouse     *vaxis.Mouse
}

func (m *mouseHandler) handleEvent(app *App, ev vaxis.Mouse) error {
	m.mouse = &ev
	// Always do an update
	err := m.update(app, m.lastFrame)
	if err != nil {
		return err
	}

	if len(m.lastHits) == 0 {
		return nil
	}

	// Handle the mouse event
	app.consumeEvent = false

	// Capture phase
	for _, h := range m.lastHits {
		c, ok := h.w.(EventCapturer)
		if !ok {
			continue
		}
		cmd, err := c.CaptureEvent(ev)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	target := m.lastHits[len(m.lastHits)-1]

	// Target phase
	cmd, err := tryHandleEvent(target.w, ev, TargetPhase)
	if err != nil {
		return err
	}
	if err := app.handleCommand(cmd); err != nil {
		return err
	}
	if app.consumeEvent {
		app.consumeEvent = false
		return nil
	}

	// Bubble phase. We don't bubble to the focused widget (which is the
	// last one in the list). Hence, - 2
	for i := len(m.lastHits) - 2; i >= 0; i -= 1 {
		h := m.lastHits[i]
		cmd, err := tryHandleEvent(h.w, ev, BubblePhase)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
		if app.consumeEvent {
			app.consumeEvent = false
			return nil
		}
	}

	return nil
}

// update hit tests s. It delivers mouse leave and mouse enter events to all
// relevant widgets which are different between the last hit list. The new
// hitlist is saved to mouseHandler
func (m *mouseHandler) update(app *App, s Surface) error {
	// Nothing to do if we don't have a mouse event
	if m.mouse == nil {
		return nil
	}

	hits := []hitResult{}

	ss := NewSubSurface(0, 0, s)
	if ss.containsPoint(m.mouse.Col, m.mouse.Row) {
		hits = hitTest(s, hits, uint16(m.mouse.Col), uint16(m.mouse.Row))
	}

	// Handle mouse exit events. These are widgets in lastHits but not in
	// hits
outer_exit:
	for _, h1 := range m.lastHits {
		for _, h2 := range hits {
			if h1 == h2 {
				continue outer_exit
			}
		}
		// h1 was not found in the new hitlist send it a mouse leave
		// event
		cmd, err := tryHandleEvent(h1.w, MouseLeave{}, TargetPhase)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
	}

	// Handle mouse enter events. These are widgets in hits but not in
	// lastHits
outer_enter:
	for _, h1 := range hits {
		for _, h2 := range m.lastHits {
			if h1 == h2 {
				continue outer_enter
			}
		}
		// h1 was not found in the old hitlist send it a mouse enter
		// event
		cmd, err := tryHandleEvent(h1.w, MouseEnter{}, TargetPhase)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
	}

	// Save this list as our current hit list
	m.lastHits = hits

	return nil
}

// mouseExit send a mouseLeave event to each widget in the last hit list
func (m *mouseHandler) mouseExit(app *App) error {
	for _, h := range m.lastHits {
		cmd, err := tryHandleEvent(h.w, MouseLeave{}, TargetPhase)
		if err != nil {
			return err
		}
		if err := app.handleCommand(cmd); err != nil {
			return err
		}
	}
	// Clear the last hit list
	m.lastHits = []hitResult{}
	return nil
}

func hitTest(s Surface, hits []hitResult, col uint16, row uint16) []hitResult {
	r := hitResult{
		col: col,
		row: row,
		w:   s.Widget,
	}
	hits = append(hits, r)
	for _, ss := range s.Children {
		if !ss.containsPoint(int(col), int(row)) {
			continue
		}
		local_col := col - uint16(ss.Origin.Col)
		local_row := row - uint16(ss.Origin.Row)
		hits = hitTest(ss.Surface, hits, local_col, local_row)
	}

	return hits
}

func debugPrintWidget(s Surface, indent int, focused Widget) {
	if s.Widget == focused {
		log.Info("\x1b[31m%s%T\x1b[m", strings.Repeat(" ", indent*4), s.Widget)
	} else {
		log.Info("%s%T", strings.Repeat(" ", indent*4), s.Widget)
	}
	for _, ch := range s.Children {
		debugPrintWidget(ch.Surface, indent+1, focused)
	}
}

func ConsumeAndRedraw() BatchCmd {
	return []Command{
		RedrawCmd{},
		ConsumeEventCmd{},
	}
}
