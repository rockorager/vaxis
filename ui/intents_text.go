package ui

// TextMotion identifies a caret movement direction.
type TextMotion int

const (
	// TextMotionLeft moves toward the previous character or word.
	TextMotionLeft TextMotion = iota
	// TextMotionRight moves toward the next character or word.
	TextMotionRight
	// TextMotionUp moves to the previous visual line.
	TextMotionUp
	// TextMotionDown moves to the next visual line.
	TextMotionDown
	// TextMotionLineStart moves to the start of the current line.
	TextMotionLineStart
	// TextMotionLineEnd moves to the end of the current line.
	TextMotionLineEnd
)

// TextMotionUnit identifies the granularity for text movement or deletion.
type TextMotionUnit int

const (
	// TextMotionCharacter moves or deletes one user-perceived character.
	TextMotionCharacter TextMotionUnit = iota
	// TextMotionWord moves or deletes one word.
	TextMotionWord
)

// TextDeleteDirection identifies deletion before or after the caret.
type TextDeleteDirection int

const (
	// TextDeleteBackward deletes before the caret.
	TextDeleteBackward TextDeleteDirection = iota
	// TextDeleteForward deletes after the caret.
	TextDeleteForward
)

const (
	// MoveCaretIntentType moves or extends the text selection.
	MoveCaretIntentType IntentType = "vaxis.text.move-caret"
	// DeleteTextIntentType deletes text near the caret.
	DeleteTextIntentType IntentType = "vaxis.text.delete"
	// InsertTextIntentType inserts text at the caret.
	InsertTextIntentType IntentType = "vaxis.text.insert"
	// InsertLineBreakIntentType inserts a line break or submits single-line text.
	InsertLineBreakIntentType IntentType = "vaxis.text.insert-line-break"
	// SelectAllTextIntentType selects all text.
	SelectAllTextIntentType IntentType = "vaxis.text.select-all"
	// CopySelectionTextIntentType copies the current text selection.
	CopySelectionTextIntentType IntentType = "vaxis.text.copy-selection"
)

// MoveCaretIntent moves the caret or extends the selection.
type MoveCaretIntent struct {
	Motion          TextMotion
	Unit            TextMotionUnit
	ExtendSelection bool
}

func (MoveCaretIntent) IntentType() IntentType {
	return MoveCaretIntentType
}

// DeleteTextIntent deletes text near the caret.
type DeleteTextIntent struct {
	Direction TextDeleteDirection
	Unit      TextMotionUnit
}

func (DeleteTextIntent) IntentType() IntentType {
	return DeleteTextIntentType
}

// InsertTextIntent inserts text at the caret.
type InsertTextIntent struct {
	Text string
}

func (InsertTextIntent) IntentType() IntentType {
	return InsertTextIntentType
}

// InsertLineBreakIntent inserts a line break or submits single-line text.
type InsertLineBreakIntent struct{}

func (InsertLineBreakIntent) IntentType() IntentType {
	return InsertLineBreakIntentType
}

// SelectAllTextIntent selects all text.
type SelectAllTextIntent struct{}

func (SelectAllTextIntent) IntentType() IntentType {
	return SelectAllTextIntentType
}

// CopySelectionTextIntent copies the current text selection.
type CopySelectionTextIntent struct {
	// OnCopied is called with the copied text after a non-empty selection is
	// placed on the clipboard.
	OnCopied func(string)
}

func (CopySelectionTextIntent) IntentType() IntentType {
	return CopySelectionTextIntentType
}
