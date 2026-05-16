# Term Widget Mouse Selection Plan

This document lays out the full implementation path for native mouse text
selection in `widgets/term`. The goal is to make the terminal widget behave
like a terminal emulator: users can select visible terminal text with the mouse,
copy it programmatically, and still allow child applications to capture mouse
events when they request terminal mouse reporting.

Ghostty is the main behavioral reference. The important idea to carry over is
not its exact data structures, but its separation of concerns:

- selection is terminal screen state, not only a render-time decoration;
- mouse gesture state is separate from persisted selection state;
- app mouse reporting takes precedence except when Shift is being used as the
  selection override;
- rendering applies a selection overlay over the existing cells;
- copying uses the selection bounds and screen wrapping metadata, not the
  rendered cell stream alone.

## Current Vaxis State

Mouse input currently flows through `Model.Update` in this order:

1. prompt click handling;
2. viewport wheel scrolling;
3. terminal mouse reporting to the child PTY.

Selection does not exist yet. `snapshotDraw` walks `visibleLine`, renders each
cell, and has no selection overlay. Scrollback is page-backed, but visible rows
can be mapped to either scrollback rows or active-screen rows via
`scrollOffset`.

The implementation should preserve existing behavior unless a selection gesture
is explicitly active.

## End State

The completed feature should support:

- left-drag cell selection across visible scrollback and active-screen content;
- Shift-left-drag selection override while child mouse reporting is active,
  unless `XTSHIFTESCAPE` / `mouseShiftCapture` says Shift should be reported to
  the child;
- selection rendering for normal and reverse-video screens;
- selection text extraction with soft-wrap unwrapping;
- clear semantics for input, output, screen changes, and alternate screen;
- tests for the selection geometry, render overlay, text extraction, and
  mouse-reporting interactions.

Optional follow-up features are double-click word selection, triple-click line
selection, rectangular selection, copy-on-select integration, and automatic
scrolling while dragging outside the viewport. The implementation should leave
room for them, but the core data model should not depend on them existing.

## Phase 1: Selection Coordinates

Add a coordinate type that can address both scrollback and active-screen rows.
Do not store only viewport row/column; viewport coordinates become stale as soon
as the user scrolls.

Proposed model:

```go
type selectionPoint struct {
	sourceRow int
	col       int
}

type selectionRange struct {
	start     selectionPoint
	end       selectionPoint
	rectangle bool
}
```

`sourceRow` should use the same conceptual space as `visibleLine`:

- `0..scrollbackLen-1` addresses scrollback;
- `scrollbackLen..scrollbackLen+height-1` addresses active-screen rows.

Add helpers:

- `viewportSourceRow(viewportRow int) (int, bool)`;
- `sourceRowLine(sourceRow int) ([]cell, screenRow, bool)`;
- `selectionRange.ordered()`;
- `selectionRange.contains(sourceRow, col int) bool`;
- `selectionRange.rowSpan(sourceRow int, width int) (left, right int, ok bool)`.

Start with normal linear selection. Keep `rectangle` in the type so rectangular
selection can use the same rendering and text extraction APIs later.

Tests:

- viewport row maps to active rows at `scrollOffset == 0`;
- viewport row maps to scrollback rows when scrolled up;
- ordering works for forward and reverse drags;
- row spans are correct for single-line and multi-line selections.

## Phase 2: Persisted Selection State

Add selection state to `Model`.

Proposed fields:

```go
selection      *selectionRange
selectionMouse mouseSelectionState
```

Proposed gesture state:

```go
type mouseSelectionState struct {
	active       bool
	start       selectionPoint
	startCol    int
	startXPixel int
	clicks      int
	lastClickAt time.Time
	lastClickRow int
	lastClickCol int
}
```

The first implementation can ignore pixel thresholds if the host does not
provide cell-relative x positions. Since `vaxis.Mouse` has `XPixel`, use it when
available and fall back to whole-cell inclusion when not. Keep the threshold
logic isolated so it can be refined without changing input handling.

Add methods:

- `clearSelection()`;
- `setSelection(*selectionRange)`;
- `hasSelection() bool`;
- `selectionActive() bool`.

Selection should be tied to the active screen. If the widget switches into the
alternate screen, clear or suspend primary selection. The conservative first
implementation should clear on alternate-screen entry and on RIS/full reset.

Tests:

- setting selection marks the model dirty;
- clearing selection marks the model dirty only when there was a selection;
- alternate-screen entry clears selection;
- RIS clears selection.

## Phase 3: Render Overlay

Apply the selection during `snapshotDraw`, before `renderCell`.

Rules:

- selected cells should use reverse video initially (`AttrReverse` toggle);
- `renderCell` must still apply `DECSCNM`, so selection and reverse-video mode
  compose predictably;
- wide-cell heads should be selected as a single drawn cell, matching the
  current render loop;
- wide-cell tails are not drawn today, so containment only needs to affect the
  head cell in the first pass.

Implementation shape:

```go
rendered := cell.Cell
if vt.selectionContains(sourceRow, col) {
	rendered.Attribute ^= vaxis.AttrReverse
}
snapshot.cells = append(snapshot.cells, positionedCell{
	col: col,
	row: r,
	cell: vt.renderCell(rendered),
})
```

This requires `snapshotDraw` to know the source row corresponding to each
viewport row. Add a helper that returns both source row and line rather than
duplicating `visibleLine` logic.

Tests:

- selected cells render with reverse attribute;
- unselected cells do not change;
- reverse-video screen mode composes with selection;
- selection over scrolled-back content renders in the visible viewport.

## Phase 4: Mouse Gesture Handling

Add a selection handler before prompt-click and PTY mouse reporting can consume
left-drag selection gestures.

Suggested order for `vaxis.Mouse` in `Update`:

1. selection mouse handling;
2. prompt click handling;
3. viewport wheel scrolling;
4. terminal mouse reporting.

The selection handler should return whether it consumed the event.

Consumption rules:

- if no mouse reporting is active, left press/motion/release can select;
- if mouse reporting is active and Shift is pressed and `mouseShiftCapture` is
  false, left press/motion/release can select;
- if mouse reporting is active and Shift is not acting as an override, clear
  selection and forward the mouse event to the child;
- wheel events keep current viewport behavior when mouse reporting is off;
- prompt click should not fire on a release that completed or updated a
  selection.

Initial drag behavior:

- left press stores the source point and clears any existing selection;
- left motion with left button pressed updates `selection`;
- left release ends the gesture and keeps the final selection;
- a click with no drag clears existing selection but does not create an empty
  selection.

Threshold behavior:

- if pixel information is available, use Ghostty's 60% cell-width threshold to
  decide endpoint inclusion;
- if pixel information is unavailable, include the start and end cells once the
  pointer crosses into another cell.

Tests:

- left drag creates a selection when mouse reporting is off;
- reverse drag creates a reversed selection that renders correctly;
- click without drag clears previous selection;
- Shift-left drag creates a selection while mouse reporting is active and
  `mouseShiftCapture` is false;
- Shift-left drag is forwarded when `mouseShiftCapture` is true;
- non-Shift mouse reporting clears selection and writes the mouse report.

## Phase 5: Selection Text Extraction

Expose an API for retrieving selected text. Keep this independent from
clipboard integration.

Proposed API:

```go
func (vt *Model) Selection() string
func (vt *Model) HasSelection() bool
func (vt *Model) ClearSelection()
```

Text extraction rules:

- use the selected source rows, not currently visible viewport rows;
- trim trailing empty cells on hard line endings;
- unwrap soft-wrapped rows using `screenRow.wrapped` / `wrapContinuation`;
- preserve newlines across hard line breaks;
- handle reverse selections by using ordered bounds;
- for wide cells, append the grapheme once from the head cell and ignore tails;
- for empty cells inside a rectangular selection later, preserve spaces.

The initial implementation should support normal linear selections only. The
public API can still be future-compatible with rectangular selections because
the internal `selectionRange` already carries that flag.

Tests:

- single-line selection returns exact text;
- multi-line hard break includes newline;
- soft-wrapped selection unwraps without newline;
- selection through scrollback returns text from history;
- wide graphemes are not duplicated;
- empty selected cells become spaces only where needed to preserve selected
  columns.

## Phase 6: Screen Mutation Semantics

Decide how selection survives terminal output. Ghostty tracks pins through
screen mutation; vaxis does not yet have tracked pins, so start conservative and
then improve.

First implementation:

- clear selection on resize/reflow;
- clear selection on full screen clear, RIS, alternate-screen switch, and
  scrollback clear;
- clear selection when active-screen scrollback capture would move selected
  active rows into history, unless we explicitly remap source rows in that same
  patch.

Better follow-up:

- when scrolling captures rows into scrollback, shift selected source rows in
  the same coordinate space;
- when scrollback trims old rows, clamp or clear selections that leave history;
- preserve selection through height-only resize where source rows still exist;
- remap selection through reflow using the existing reflow source mapping
  helpers.

Tests for first implementation:

- printing over selected content clears selection only if that is the chosen
  conservative rule;
- resize clears selection;
- scrollback clear clears selection;
- alternate-screen entry and exit leave no stale selection.

Tests for improved implementation:

- selection moves with scrollback capture;
- selection clears when trimmed out of scrollback;
- selection survives simple viewport scrolling;
- selection remaps through resize reflow.

## Phase 7: Double Click, Triple Click, And Word/Line Helpers

After base drag selection and copy extraction work, add click-count behavior.

Double click:

- select the word under the pointer;
- word boundaries should be configurable eventually, but start with whitespace
  versus non-whitespace;
- dragging after a double click extends by whole words.

Triple click:

- select the logical line under the pointer;
- include soft-wrapped physical rows;
- trim leading/trailing whitespace for non-empty lines;
- dragging after a triple click extends by whole logical lines.

Tests:

- double click selects a word;
- double-click drag extends by words in both directions;
- triple click selects across soft wraps;
- triple-click drag extends by logical lines;
- click count resets after timeout or cell-distance threshold.

## Phase 8: Rectangle Selection

Add rectangular selection as an optional mode after normal selection is stable.

Suggested modifier:

- use Ctrl+Alt on non-Darwin and Alt on Darwin if platform information is
  available to this package;
- if platform-specific behavior is undesirable for the library, expose an
  option or use Ctrl+Alt consistently.

Behavior:

- rectangular selection uses fixed column spans across each selected row;
- endpoint threshold logic should consider columns rather than linear cell
  order;
- text extraction preserves rectangular spaces and newlines per visual row.

Tests:

- rectangle selection renders only selected columns;
- reverse rectangle drag works;
- rectangular extraction preserves row shape;
- rectangle mode still works while Shift overrides mouse reporting.

## Phase 9: Clipboard And Host Integration

Keep clipboard integration outside the first core patch unless there is already
a host clipboard abstraction available to the term widget.

Possible additions:

- `CopySelection()` helper returning text and clearing nothing;
- optional copy-on-select command or event;
- event emitted when selection changes, so an application can mirror it to a
  clipboard;
- middle-click paste from selection if the host app exposes a selection
  clipboard concept.

Tests:

- selection-change event fires on set and clear;
- copy API does not mutate selection;
- copy-on-select, if added, fires only on release rather than every motion.

## Phase 10: Manual Verification

Use `_examples/term` or a focused example command to verify:

- shell prompt click still moves the cursor when no selection drag occurred;
- `less`, `vim`, and similar mouse-reporting apps receive mouse events normally;
- Shift-drag selects over `vim` unless Shift capture is enabled;
- wheel scrolling still scrolls viewport when mouse reporting is off;
- alternate-screen wheel behavior still sends cursor keys when applicable;
- selected text copies correctly from scrollback and active content;
- visual selection remains correct with wide characters and reverse-video mode.

## Suggested Commit Sequence

1. Add selection coordinate types and tests.
2. Add persisted selection state and clear/set helpers.
3. Add render overlay for selected cells.
4. Add basic left-drag selection input.
5. Add selection text extraction API.
6. Wire conservative clear semantics into screen resets, alternate screen, and
   resize.
7. Add Shift override behavior around mouse reporting.
8. Add double-click word selection.
9. Add triple-click line selection.
10. Add rectangle selection.
11. Add selection-change/copy integration if desired.

Each commit should keep the existing mouse reporting tests passing. New tests
should live mostly in `selection_test.go`, with targeted additions to
`mouse_test.go`, `viewport_test.go`, and `resize_test.go` where behavior crosses
those boundaries.
