package term

import (
	"os/exec"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

// Terminal adapts Model to the ui widget framework.
//
// If Model is nil, Terminal creates and owns a model using Options and starts
// Command when it first receives a non-zero layout size. If Model is non-nil,
// the caller owns the model and Terminal only adapts layout, painting, focus,
// and events.
type Terminal struct {
	// Model is an optional pre-created terminal model.
	Model *Model
	// Command is started in a PTY when Model is nil.
	Command *exec.Cmd
	// Options are passed to New when Model is nil.
	Options []Option
	// AutoFocus requests focus for the terminal after it is built.
	AutoFocus bool
	// OnEvent receives terminal-generated events on the UI event loop.
	OnEvent func(ui.EventContext, ui.Event) ui.EventResult
}

func (w Terminal) CreateState() ui.State {
	return &terminalState{}
}

type terminalState struct {
	ui.StateBase
	model       *Model
	owned       bool
	started     bool
	focused     bool
	autoFocused bool
	mouseShape  ui.MouseShape
}

func (s *terminalState) InitState() {
	s.configureModel()
}

func (s *terminalState) DidUpdateWidget(old ui.Widget) {
	oldModel := old.(Terminal).Model
	nextModel := s.Widget().(Terminal).Model
	if oldModel == nextModel {
		return
	}
	if s.owned && s.model != nil {
		s.model.Close()
	}
	s.model = nil
	s.owned = false
	s.started = false
	s.configureModel()
}

func (s *terminalState) Dispose() {
	if s.model != nil {
		s.model.Detach()
		if s.owned {
			s.model.Close()
		}
	}
}

func (s *terminalState) Build(ctx ui.BuildContext) ui.Widget {
	child := terminalRender{model: s.model, onResize: s.resize, onFocus: s.setFocused, mouseShape: s.mouseShape}
	if s.Widget().(Terminal).AutoFocus && !s.autoFocused {
		childWidget := ui.FocusScope{AutoFocus: true, Child: child}
		return childWidget
	}
	return child
}

func (s *terminalState) configureModel() {
	w := s.Widget().(Terminal)
	if w.Model != nil {
		s.model = w.Model
		s.owned = false
	} else {
		s.model = New(w.Options...)
		s.owned = true
	}
	s.mouseShape = vaxis.MouseShapeTextInput
	runtime := s.Context().Runtime()
	s.model.Attach(func(ev vaxis.Event) {
		runtime.Dispatch(func() { s.handleModelEvent(ev) })
	})
}

// AttachUI attaches a UI-native terminal event handler.
func (vt *Model) AttachUI(fn func(ui.Event)) {
	if fn == nil {
		vt.Attach(nil)
		return
	}
	vt.Attach(func(ev vaxis.Event) { fn(ev) })
}

func (s *terminalState) handleModelEvent(ev ui.Event) {
	if s.model == nil {
		return
	}
	ctx := s.Context().EventContext()
	switch ev := ev.(type) {
	case vaxis.Redraw:
		s.MarkNeedsBuild()
	case EventNotify:
		ctx.Notify(ev.Title, ev.Body)
	case EventTitle:
		ctx.SetTitle(string(ev))
	case EventMouseShape:
		s.mouseShape = ev.Shape
		s.MarkNeedsBuild()
	}
	if cb := s.Widget().(Terminal).OnEvent; cb != nil {
		cb(ctx, ev)
	}
}

func (s *terminalState) resize(size ui.Size) {
	if s.model == nil || size.Width <= 0 || size.Height <= 0 {
		return
	}
	if s.owned && !s.started {
		cmd := s.Widget().(Terminal).Command
		if cmd != nil {
			if err := s.model.StartWithSize(cmd, size.Width, size.Height); err != nil {
				s.handleModelEvent(EventClosed{Term: s.model, Error: err})
			}
		}
		s.started = true
		return
	}
	s.model.Resize(size.Width, size.Height)
}

func (s *terminalState) setFocused(focused bool) {
	if s.model == nil || s.focused == focused {
		return
	}
	s.focused = focused
	if focused {
		s.autoFocused = true
	}
	if focused {
		s.model.Focus()
	} else {
		s.model.Blur()
	}
}

type terminalRender struct {
	model      *Model
	onResize   func(ui.Size)
	onFocus    func(bool)
	mouseShape ui.MouseShape
}

func (w terminalRender) CreateRenderObject(ui.BuildContext) ui.RenderObject {
	return &renderTerminal{model: w.model, onResize: w.onResize, onFocus: w.onFocus, mouseShape: w.mouseShape, focusedIndex: -1}
}

func (w terminalRender) UpdateRenderObject(_ ui.BuildContext, ro ui.RenderObject) {
	r := ro.(*renderTerminal)
	r.model = w.model
	r.onResize = w.onResize
	r.onFocus = w.onFocus
	r.mouseShape = w.mouseShape
	r.MarkNeedsPaint()
}

type renderTerminal struct {
	ui.LeafRenderObject
	model        *Model
	onResize     func(ui.Size)
	onFocus      func(bool)
	mouseShape   ui.MouseShape
	focusedIndex int
}

func (r *renderTerminal) Layout(_ ui.LayoutContext, c ui.Constraints) {
	size := terminalSizeForConstraints(c)
	r.SetSize(size)
	if r.onResize != nil {
		r.onResize(size)
	}
}

func (r *renderTerminal) DryLayout(_ ui.LayoutContext, c ui.Constraints) ui.Size {
	return terminalSizeForConstraints(c)
}

func terminalSizeForConstraints(c ui.Constraints) ui.Size {
	size := ui.Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func (r *renderTerminal) Paint(p *ui.Painter, off ui.Offset) {
	if r.model == nil {
		return
	}
	snapshot := r.model.Snapshot()
	for _, cell := range snapshot.Cells {
		p.DrawCell(ui.Point{X: off.X + cell.Col, Y: off.Y + cell.Row}, cell.Cell)
	}
	if r.focusedIndex >= 0 && snapshot.CursorVisible {
		p.ShowCursor(off.X+snapshot.CursorCol, off.Y+snapshot.CursorRow, snapshot.CursorStyle)
	}
}

func (r *renderTerminal) HitTest(*ui.HitTestResult, ui.Point) bool {
	return true
}

func (r *renderTerminal) HandleEvent(ctx ui.EventContext, ev ui.Event) ui.EventResult {
	if r.model == nil {
		return ui.EventIgnored
	}
	switch ev.(type) {
	case vaxis.Key, vaxis.PasteStartEvent, vaxis.PasteEndEvent:
		if r.focusedIndex < 0 {
			return ui.EventIgnored
		}
		r.model.Update(ev)
		return ui.EventHandled
	case vaxis.Mouse:
		r.model.Update(ev)
		return ui.EventHandled
	case vaxis.ColorThemeUpdate:
		r.model.Update(ev)
		return ui.EventIgnored
	}
	return ui.EventIgnored
}

func (r *renderTerminal) FocusableCount() int {
	return 1
}

func (r *renderTerminal) SetFocusedIndex(index int) {
	focused := index >= 0
	if r.focusedIndex == index {
		return
	}
	r.focusedIndex = index
	if r.onFocus != nil {
		r.onFocus(focused)
	}
}

func (r *renderTerminal) MouseShape(ui.EventContext, ui.Mouse) ui.MouseShape {
	if r.mouseShape == "" {
		return ui.MouseShapeTextInput
	}
	return r.mouseShape
}
