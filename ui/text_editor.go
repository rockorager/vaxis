package ui

type textEditorInsertMode int

const (
	textEditorSingleLine textEditorInsertMode = iota
	textEditorMultiline
)

type textEditorState struct {
	node      FocusNode
	buffer    TextBuffer
	selecting bool
}

type textEditorHandleOptions struct {
	insertMode       textEditorInsertMode
	markNeedsBuild   func()
	onChanged        TextChangedCallback
	submit           func(EventContext, string)
	positionForMouse func(Mouse) (TextPosition, bool)
	moveUp           func() bool
	moveDown         func() bool
	extendUp         func() bool
	extendDown       func() bool
}

func (s *textEditorState) SyncValue(value string) {
	if s.buffer.Text() != value {
		s.buffer.SetText(value)
	}
}

func (s *textEditorState) SetFocusChange(fn func()) {
	s.node.onChange = fn
}

func (s *textEditorState) Focus(child Widget) Widget {
	return Focus(&s.node, child)
}

func (s *textEditorState) Text() string {
	return s.buffer.Text()
}

func (s *textEditorState) Len() int {
	return s.buffer.Len()
}

func (s *textEditorState) CursorOffset() int {
	return s.buffer.CursorOffset()
}

func (s *textEditorState) Selection() TextSelection {
	return s.buffer.Selection()
}

func (s *textEditorState) HasFocus() bool {
	return s.node.HasFocus()
}

func (s *textEditorState) PositionForOffset(offset int) TextPosition {
	return s.buffer.positionForOffset(offset)
}

func (s *textEditorState) MoveVisualUp(layout TextLayout) bool {
	if len(layout.Lines) > 0 {
		return s.buffer.MoveVisualUp(layout)
	}
	return s.buffer.MoveLineUp()
}

func (s *textEditorState) MoveVisualDown(layout TextLayout) bool {
	if len(layout.Lines) > 0 {
		return s.buffer.MoveVisualDown(layout)
	}
	return s.buffer.MoveLineDown()
}

func (s *textEditorState) ExtendVisualUp(layout TextLayout) bool {
	if len(layout.Lines) > 0 {
		return s.buffer.ExtendVisualUp(layout)
	}
	return s.buffer.ExtendLineUp()
}

func (s *textEditorState) ExtendVisualDown(layout TextLayout) bool {
	if len(layout.Lines) > 0 {
		return s.buffer.ExtendVisualDown(layout)
	}
	return s.buffer.ExtendLineDown()
}

func (s *textEditorState) HandleEvent(ctx EventContext, ev Event, opts textEditorHandleOptions) EventResult {
	return textEditorEventHandler{
		buffer:           &s.buffer,
		selecting:        &s.selecting,
		insertMode:       opts.insertMode,
		requestFocus:     s.node.RequestFocus,
		markNeedsBuild:   opts.markNeedsBuild,
		change:           s.change(opts.onChanged, opts.markNeedsBuild),
		submit:           opts.submit,
		positionForMouse: opts.positionForMouse,
		moveUp:           opts.moveUp,
		moveDown:         opts.moveDown,
		extendUp:         opts.extendUp,
		extendDown:       opts.extendDown,
	}.HandleEvent(ctx, ev)
}

func (s *textEditorState) change(onChanged TextChangedCallback, markNeedsBuild func()) func(EventContext) {
	return func(ctx EventContext) {
		if onChanged != nil {
			onChanged(ctx, s.buffer.Text())
			return
		}
		markNeedsBuild()
	}
}

type textEditorEventHandler struct {
	buffer           *TextBuffer
	selecting        *bool
	insertMode       textEditorInsertMode
	requestFocus     func()
	markNeedsBuild   func()
	change           func(EventContext)
	submit           func(EventContext, string)
	positionForMouse func(Mouse) (TextPosition, bool)
	moveUp           func() bool
	moveDown         func() bool
	extendUp         func() bool
	extendDown       func() bool
}

func (h textEditorEventHandler) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		return h.handleKey(ctx, ev)
	case Mouse:
		return h.handleMouse(ev)
	default:
		return EventIgnored
	}
}

func (h textEditorEventHandler) handleKey(ctx EventContext, key Key) EventResult {
	if keyIsRelease(key) {
		return EventIgnored
	}
	changed := false
	handled := true
	switch {
	case key.MatchString("Ctrl+a"):
		handled = h.buffer.SelectAll()
	case key.MatchString("Ctrl+c"):
		if h.buffer.HasSelection() {
			ctx.Copy(h.buffer.SelectedText())
		}
	case key.MatchString("Ctrl+Shift+Left"):
		handled = h.buffer.ExtendWordLeft()
	case key.MatchString("Ctrl+Shift+Right"):
		handled = h.buffer.ExtendWordRight()
	case key.MatchString("Ctrl+Left"):
		handled = h.buffer.MoveWordLeft()
	case key.MatchString("Ctrl+Right"):
		handled = h.buffer.MoveWordRight()
	case key.MatchString("Shift+Left"):
		handled = h.buffer.ExtendLeft()
	case key.MatchString("Shift+Right"):
		handled = h.buffer.ExtendRight()
	case key.MatchString("Shift+Up"):
		if h.extendUp == nil {
			return EventIgnored
		}
		handled = h.extendUp()
	case key.MatchString("Shift+Down"):
		if h.extendDown == nil {
			return EventIgnored
		}
		handled = h.extendDown()
	case key.MatchString("Shift+Home"):
		handled = h.buffer.ExtendHome()
	case key.MatchString("Shift+End"):
		handled = h.buffer.ExtendEnd()
	case key.Keycode == KeyLeft:
		handled = h.buffer.MoveLeft()
	case key.Keycode == KeyRight:
		handled = h.buffer.MoveRight()
	case key.Keycode == KeyUp:
		if h.moveUp == nil {
			return EventIgnored
		}
		handled = h.moveUp()
	case key.Keycode == KeyDown:
		if h.moveDown == nil {
			return EventIgnored
		}
		handled = h.moveDown()
	case key.Keycode == KeyHome:
		handled = h.buffer.MoveHome()
	case key.Keycode == KeyEnd:
		handled = h.buffer.MoveEnd()
	case key.Keycode == KeyBackspace:
		if key.MatchString("Ctrl+Backspace") {
			changed = h.buffer.DeleteWordBackward()
		} else {
			changed = h.buffer.DeleteBackward()
		}
	case key.Keycode == KeyDelete:
		if key.MatchString("Ctrl+Delete") {
			changed = h.buffer.DeleteWordForward()
		} else {
			changed = h.buffer.DeleteForward()
		}
	case key.MatchString("Enter"):
		if h.insertMode == textEditorMultiline {
			changed = h.buffer.Insert("\n")
		} else if h.submit != nil {
			h.submit(ctx, h.buffer.Text())
			return EventHandled
		} else {
			return EventHandled
		}
	case key.Text != "":
		if h.insertMode == textEditorMultiline {
			changed = h.buffer.Insert(key.Text)
		} else {
			changed = h.buffer.InsertSingleLine(key.Text)
		}
	default:
		return EventIgnored
	}
	if changed {
		h.change(ctx)
		return EventHandled
	}
	if handled {
		h.markNeedsBuild()
		return EventHandled
	}
	return EventHandled
}

func (h textEditorEventHandler) handleMouse(mouse Mouse) EventResult {
	if h.positionForMouse == nil {
		return EventIgnored
	}
	if mouse.Button != MouseLeftButton {
		if mouse.EventType == EventRelease {
			*h.selecting = false
			return EventHandled
		}
		return EventIgnored
	}
	pos, ok := h.positionForMouse(mouse)
	if !ok {
		return EventIgnored
	}
	switch mouse.EventType {
	case EventPress:
		h.requestFocus()
		*h.selecting = true
		h.buffer.CollapseSelection(pos)
		h.markNeedsBuild()
		return EventHandled
	case EventMotion:
		if !*h.selecting {
			return EventIgnored
		}
		h.buffer.ExtendSelection(pos)
		h.markNeedsBuild()
		return EventHandled
	case EventRelease:
		if !*h.selecting {
			return EventIgnored
		}
		*h.selecting = false
		h.buffer.ExtendSelection(pos)
		h.markNeedsBuild()
		return EventHandled
	default:
		return EventIgnored
	}
}
