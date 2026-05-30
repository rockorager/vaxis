package ui

import "time"

type textEditorInsertMode int

const (
	textEditorSingleLine textEditorInsertMode = iota
	textEditorMultiline
)

const textEditorMultiClickInterval = 500 * time.Millisecond

type textEditorState struct {
	node         FocusNode
	buffer       TextBuffer
	selecting    bool
	now          func() time.Time
	lastClick    time.Time
	lastClickRow int
	lastClickCol int
	clickCount   int
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

func (s *textEditorState) DefaultActions(opts textEditorHandleOptions, child Widget) Widget {
	h := s.eventHandler(opts)
	return DefaultActions{
		Bindings: map[IntentType]ActionFunc{
			MoveCaretIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.moveCaret(ctx, intent)
			},
			DeleteTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.deleteText(ctx, intent)
			},
			InsertTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.insertText(ctx, intent)
			},
			InsertLineBreakIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.insertLineBreak(ctx, intent)
			},
			SelectAllTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.selectAll(ctx, intent)
			},
			CopySelectionTextIntentType: func(ctx EventContext, intent Intent) EventResult {
				return h.copySelection(ctx, intent)
			},
		},
		Child: child,
	}
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

func (s *textEditorState) SetCursorOffset(offset int) {
	s.buffer.SetCursorOffset(offset)
}

func (s *textEditorState) SetSelection(selection TextSelection) bool {
	return s.buffer.SetSelection(selection)
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
	return s.eventHandler(opts).HandleEvent(ctx, ev)
}

func (s *textEditorState) eventHandler(opts textEditorHandleOptions) textEditorEventHandler {
	return textEditorEventHandler{
		buffer:           &s.buffer,
		selecting:        &s.selecting,
		clickCount:       s.mouseClickCount,
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
	}
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

func (s *textEditorState) mouseClickCount(mouse Mouse) int {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if s.clickCount == 0 || mouse.Row != s.lastClickRow || mouse.Col != s.lastClickCol || now.Sub(s.lastClick) > textEditorMultiClickInterval {
		s.clickCount = 1
	} else {
		s.clickCount++
	}
	s.lastClick = now
	s.lastClickRow = mouse.Row
	s.lastClickCol = mouse.Col
	return s.clickCount
}

type textEditorEventHandler struct {
	buffer           *TextBuffer
	selecting        *bool
	clickCount       func(Mouse) int
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
	switch {
	case key.MatchString("Ctrl+a"):
		return ctx.Invoke(SelectAllTextIntent{})
	case key.MatchString("Ctrl+c"):
		return ctx.Invoke(CopySelectionTextIntent{})
	case key.MatchString("Ctrl+Shift+Left"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLeft, Unit: TextMotionWord, ExtendSelection: true})
	case key.MatchString("Ctrl+Shift+Right"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionRight, Unit: TextMotionWord, ExtendSelection: true})
	case key.MatchString("Ctrl+Left"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLeft, Unit: TextMotionWord})
	case key.MatchString("Ctrl+Right"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionRight, Unit: TextMotionWord})
	case key.MatchString("Shift+Left"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLeft, Unit: TextMotionCharacter, ExtendSelection: true})
	case key.MatchString("Shift+Right"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionRight, Unit: TextMotionCharacter, ExtendSelection: true})
	case key.MatchString("Shift+Up"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionUp, ExtendSelection: true})
	case key.MatchString("Shift+Down"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionDown, ExtendSelection: true})
	case key.MatchString("Shift+Home"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLineStart, ExtendSelection: true})
	case key.MatchString("Shift+End"):
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLineEnd, ExtendSelection: true})
	case key.Keycode == KeyLeft:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLeft, Unit: TextMotionCharacter})
	case key.Keycode == KeyRight:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionRight, Unit: TextMotionCharacter})
	case key.Keycode == KeyUp:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionUp})
	case key.Keycode == KeyDown:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionDown})
	case key.Keycode == KeyHome:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLineStart})
	case key.Keycode == KeyEnd:
		return ctx.Invoke(MoveCaretIntent{Motion: TextMotionLineEnd})
	case key.Keycode == KeyBackspace:
		if key.MatchString("Ctrl+Backspace") {
			return ctx.Invoke(DeleteTextIntent{Direction: TextDeleteBackward, Unit: TextMotionWord})
		}
		return ctx.Invoke(DeleteTextIntent{Direction: TextDeleteBackward, Unit: TextMotionCharacter})
	case key.Keycode == KeyDelete:
		if key.MatchString("Ctrl+Delete") {
			return ctx.Invoke(DeleteTextIntent{Direction: TextDeleteForward, Unit: TextMotionWord})
		}
		return ctx.Invoke(DeleteTextIntent{Direction: TextDeleteForward, Unit: TextMotionCharacter})
	case key.MatchString("Enter"):
		return ctx.Invoke(InsertLineBreakIntent{})
	case key.Text != "":
		return ctx.Invoke(InsertTextIntent{Text: key.Text})
	default:
		return EventIgnored
	}
}

func (h textEditorEventHandler) moveCaret(ctx EventContext, intent Intent) EventResult {
	move, ok := intent.(MoveCaretIntent)
	if !ok {
		return EventIgnored
	}
	handled := false
	switch move.Motion {
	case TextMotionLeft:
		if move.ExtendSelection {
			if move.Unit == TextMotionWord {
				handled = h.buffer.ExtendWordLeft()
			} else {
				handled = h.buffer.ExtendLeft()
			}
		} else if move.Unit == TextMotionWord {
			handled = h.buffer.MoveWordLeft()
		} else {
			handled = h.buffer.MoveLeft()
		}
	case TextMotionRight:
		if move.ExtendSelection {
			if move.Unit == TextMotionWord {
				handled = h.buffer.ExtendWordRight()
			} else {
				handled = h.buffer.ExtendRight()
			}
		} else if move.Unit == TextMotionWord {
			handled = h.buffer.MoveWordRight()
		} else {
			handled = h.buffer.MoveRight()
		}
	case TextMotionUp:
		handled = h.moveVertical(move.ExtendSelection, true)
	case TextMotionDown:
		handled = h.moveVertical(move.ExtendSelection, false)
	case TextMotionLineStart:
		if move.ExtendSelection {
			handled = h.buffer.ExtendHome()
		} else {
			handled = h.buffer.MoveHome()
		}
	case TextMotionLineEnd:
		if move.ExtendSelection {
			handled = h.buffer.ExtendEnd()
		} else {
			handled = h.buffer.MoveEnd()
		}
	default:
		return EventIgnored
	}
	return h.finishUnchanged(handled)
}

func (h textEditorEventHandler) moveVertical(extend, up bool) bool {
	switch {
	case extend && up && h.extendUp != nil:
		return h.extendUp()
	case extend && !up && h.extendDown != nil:
		return h.extendDown()
	case !extend && up && h.moveUp != nil:
		return h.moveUp()
	case !extend && !up && h.moveDown != nil:
		return h.moveDown()
	default:
		return false
	}
}

func (h textEditorEventHandler) deleteText(ctx EventContext, intent Intent) EventResult {
	deleteIntent, ok := intent.(DeleteTextIntent)
	if !ok {
		return EventIgnored
	}
	changed := false
	switch deleteIntent.Direction {
	case TextDeleteBackward:
		if deleteIntent.Unit == TextMotionWord {
			changed = h.buffer.DeleteWordBackward()
		} else {
			changed = h.buffer.DeleteBackward()
		}
	case TextDeleteForward:
		if deleteIntent.Unit == TextMotionWord {
			changed = h.buffer.DeleteWordForward()
		} else {
			changed = h.buffer.DeleteForward()
		}
	default:
		return EventIgnored
	}
	return h.finishChanged(ctx, changed)
}

func (h textEditorEventHandler) insertText(ctx EventContext, intent Intent) EventResult {
	insert, ok := intent.(InsertTextIntent)
	if !ok {
		return EventIgnored
	}
	changed := false
	if h.insertMode == textEditorMultiline {
		changed = h.buffer.Insert(insert.Text)
	} else {
		changed = h.buffer.InsertSingleLine(insert.Text)
	}
	return h.finishChanged(ctx, changed)
}

func (h textEditorEventHandler) insertLineBreak(ctx EventContext, intent Intent) EventResult {
	if _, ok := intent.(InsertLineBreakIntent); !ok {
		return EventIgnored
	}
	if h.insertMode == textEditorMultiline {
		return h.finishChanged(ctx, h.buffer.Insert("\n"))
	}
	if h.submit != nil {
		h.submit(ctx, h.buffer.Text())
	}
	return EventHandled
}

func (h textEditorEventHandler) selectAll(ctx EventContext, intent Intent) EventResult {
	if _, ok := intent.(SelectAllTextIntent); !ok {
		return EventIgnored
	}
	return h.finishUnchanged(h.buffer.SelectAll())
}

func (h textEditorEventHandler) copySelection(ctx EventContext, intent Intent) EventResult {
	copyIntent, ok := intent.(CopySelectionTextIntent)
	if !ok {
		return EventIgnored
	}
	if h.buffer.HasSelection() {
		text := h.buffer.SelectedText()
		ctx.Copy(text)
		if text != "" && copyIntent.OnCopied != nil {
			copyIntent.OnCopied(text)
		}
	}
	return EventHandled
}

func (h textEditorEventHandler) finishChanged(ctx EventContext, changed bool) EventResult {
	if changed {
		h.change(ctx)
		return EventHandled
	}
	return EventHandled
}

func (h textEditorEventHandler) finishUnchanged(handled bool) EventResult {
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
		clickCount := 1
		if h.clickCount != nil {
			clickCount = h.clickCount(mouse)
		}
		switch {
		case clickCount >= 3:
			*h.selecting = false
			h.buffer.SelectLineAt(pos)
		case clickCount == 2:
			*h.selecting = false
			h.buffer.SelectWordAt(pos)
		default:
			*h.selecting = true
			h.buffer.CollapseSelection(pos)
		}
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
