package term

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestOSCTitleEvent(t *testing.T) {
	vt := New()

	vt.osc("2;Hello World")

	ev := <-vt.events
	title, ok := ev.(EventTitle)
	if !ok {
		t.Fatalf("event = %T, want EventTitle", ev)
	}
	if got, want := string(title), "Hello World"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
}

func TestOSCTitleEventEmpty(t *testing.T) {
	vt := New()

	vt.osc("2;")

	ev := <-vt.events
	title, ok := ev.(EventTitle)
	if !ok {
		t.Fatalf("event = %T, want EventTitle", ev)
	}
	if got, want := string(title), ""; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
}

func TestOSCTitleTruncated(t *testing.T) {
	vt := New()
	long := strings.Repeat("a", maxTitleLen+10)

	vt.osc("2;" + long)

	ev := <-vt.events
	title, ok := ev.(EventTitle)
	if !ok {
		t.Fatalf("event = %T, want EventTitle", ev)
	}
	if got, want := len(string(title)), maxTitleLen; got != want {
		t.Fatalf("title length = %d, want %d", got, want)
	}
}

func TestOSCTitleInvalidUTF8Ignored(t *testing.T) {
	vt := New()
	vt.title = "old"

	vt.osc("2;\xff")

	if got, want := vt.title, "old"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for invalid title: %T", ev)
	default:
	}
}

func TestParsedOSCTitleInvalidUTF8Ignored(t *testing.T) {
	vt := New()
	vt.title = "old"

	parseAndApply(t, vt, "\x1b]2;abc\xc0\x1b\\")

	if got, want := vt.title, "old"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for invalid title: %T", ev)
	default:
	}
}

func TestOSC8Hyperlink(t *testing.T) {
	vt := New()

	vt.osc("8;id=foo;https://example.com")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got, want := vt.cursor.HyperlinkParams, "id=foo"; got != want {
		t.Fatalf("hyperlink params = %q, want %q", got, want)
	}
}

func TestOSC8HyperlinkEnd(t *testing.T) {
	vt := New()
	vt.cursor.Hyperlink = "https://example.com"
	vt.cursor.HyperlinkParams = "id=foo"

	vt.osc("8;;")

	if got, want := vt.cursor.Hyperlink, ""; got != want {
		t.Fatalf("hyperlink = %q, want empty", got)
	}
	if got, want := vt.cursor.HyperlinkParams, ""; got != want {
		t.Fatalf("hyperlink params = %q, want empty", got)
	}
}

func TestOSC8EmptyIDWithURIStartsHyperlinkWithoutParams(t *testing.T) {
	vt := New()

	vt.osc("8;id=;https://example.com")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got := vt.cursor.HyperlinkParams; got != "" {
		t.Fatalf("hyperlink params = %q, want empty", got)
	}
}

func TestOSC8IncompleteKeyIgnored(t *testing.T) {
	vt := New()

	vt.osc("8;id;https://example.com")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got := vt.cursor.HyperlinkParams; got != "" {
		t.Fatalf("hyperlink params = %q, want empty", got)
	}
}

func TestOSC8EmptyKeyIgnored(t *testing.T) {
	vt := New()

	vt.osc("8;=value;https://example.com")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got := vt.cursor.HyperlinkParams; got != "" {
		t.Fatalf("hyperlink params = %q, want empty", got)
	}
}

func TestOSC8EmptyKeyWithIDIgnoresEmptyKey(t *testing.T) {
	vt := New()

	vt.osc("8;=value:id=foo;https://example.com")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got, want := vt.cursor.HyperlinkParams, "id=foo"; got != want {
		t.Fatalf("hyperlink params = %q, want %q", got, want)
	}
}

func TestOSC8EmptyURIWithNonEmptyIDIgnored(t *testing.T) {
	vt := New()
	vt.cursor.Hyperlink = "https://example.com"
	vt.cursor.HyperlinkParams = "id=old"

	vt.osc("8;id=foo;")

	if got, want := vt.cursor.Hyperlink, "https://example.com"; got != want {
		t.Fatalf("hyperlink = %q, want %q", got, want)
	}
	if got, want := vt.cursor.HyperlinkParams, "id=old"; got != want {
		t.Fatalf("hyperlink params = %q, want %q", got, want)
	}
}

func TestOSC8EmptyURIWithEmptyIDIEndsHyperlink(t *testing.T) {
	vt := New()
	vt.cursor.Hyperlink = "https://example.com"
	vt.cursor.HyperlinkParams = "id=old"

	vt.osc("8;id=;")

	if got, want := vt.cursor.Hyperlink, ""; got != want {
		t.Fatalf("hyperlink = %q, want empty", got)
	}
	if got, want := vt.cursor.HyperlinkParams, ""; got != want {
		t.Fatalf("hyperlink params = %q, want empty", got)
	}
}

func TestSaveRestoreCursorDoesNotRestoreHyperlink(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.osc("8;id=saved;https://saved.example")
	vt.update(testESC('7'))
	vt.osc("8;id=active;https://active.example")
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Hyperlink, "https://active.example"; got != want {
		t.Fatalf("cell hyperlink = %q, want %q", got, want)
	}
	if got, want := cell.HyperlinkParams, "id=active"; got != want {
		t.Fatalf("cell hyperlink params = %q, want %q", got, want)
	}
}

func TestSaveRestoreCursorDoesNotModifyActiveHyperlink(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.osc("8;id=active;https://active.example")
	vt.update(testESC('7'))
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Hyperlink, "https://active.example"; got != want {
		t.Fatalf("cell hyperlink = %q, want %q", got, want)
	}
	if got, want := cell.HyperlinkParams, "id=active"; got != want {
		t.Fatalf("cell hyperlink params = %q, want %q", got, want)
	}
}

func TestRestoreUnsavedCursorDoesNotResetHyperlink(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.osc("8;id=active;https://active.example")
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Hyperlink, "https://active.example"; got != want {
		t.Fatalf("cell hyperlink after unsaved restore = %q, want %q", got, want)
	}
	if got, want := cell.HyperlinkParams, "id=active"; got != want {
		t.Fatalf("cell hyperlink params after unsaved restore = %q, want %q", got, want)
	}
}

func TestRISClearsHyperlink(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.osc("8;id=active;https://active.example")
	vt.update(testESC('c'))
	vt.update(testPrint("x"))

	cell := vt.activeScreen.cell(0, 0)
	if got := cell.Hyperlink; got != "" {
		t.Fatalf("cell hyperlink after RIS = %q, want empty", got)
	}
	if got := cell.HyperlinkParams; got != "" {
		t.Fatalf("cell hyperlink params after RIS = %q, want empty", got)
	}
}

func TestPrintWithHyperlink(t *testing.T) {
	vt := New()
	vt.resize(8, 1)

	vt.osc("8;id=one;http://example.com")
	printText(vt, "123456")

	for col := column(0); col < 6; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "http://example.com"; got != want {
			t.Fatalf("cell %d hyperlink = %q, want %q", col, got, want)
		}
		if got, want := cell.HyperlinkParams, "id=one"; got != want {
			t.Fatalf("cell %d hyperlink params = %q, want %q", col, got, want)
		}
	}
}

func TestPrintAndEndHyperlink(t *testing.T) {
	vt := New()
	vt.resize(8, 1)

	vt.osc("8;id=one;http://example.com")
	printText(vt, "123")
	vt.osc("8;;")
	printText(vt, "456")

	for col := column(0); col < 3; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "http://example.com"; got != want {
			t.Fatalf("linked cell %d hyperlink = %q, want %q", col, got, want)
		}
	}
	for col := column(3); col < 6; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got := cell.Hyperlink; got != "" {
			t.Fatalf("unlinked cell %d hyperlink = %q, want empty", col, got)
		}
		if got := cell.HyperlinkParams; got != "" {
			t.Fatalf("unlinked cell %d hyperlink params = %q, want empty", col, got)
		}
	}
}

func TestPrintAndChangeHyperlink(t *testing.T) {
	vt := New()
	vt.resize(8, 1)

	vt.osc("8;id=one;http://one.example.com")
	printText(vt, "123")
	vt.osc("8;id=two;http://two.example.com")
	printText(vt, "456")

	for col := column(0); col < 3; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "http://one.example.com"; got != want {
			t.Fatalf("first link cell %d hyperlink = %q, want %q", col, got, want)
		}
		if got, want := cell.HyperlinkParams, "id=one"; got != want {
			t.Fatalf("first link cell %d hyperlink params = %q, want %q", col, got, want)
		}
	}
	for col := column(3); col < 6; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "http://two.example.com"; got != want {
			t.Fatalf("second link cell %d hyperlink = %q, want %q", col, got, want)
		}
		if got, want := cell.HyperlinkParams, "id=two"; got != want {
			t.Fatalf("second link cell %d hyperlink params = %q, want %q", col, got, want)
		}
	}
}

func TestOverwriteHyperlinkClearsOldHyperlink(t *testing.T) {
	vt := New()
	vt.resize(8, 1)

	vt.osc("8;id=one;http://one.example.com")
	printText(vt, "123")
	vt.cursor.col = 0
	vt.osc("8;;")
	printText(vt, "456")

	for col := column(0); col < 3; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got := cell.Hyperlink; got != "" {
			t.Fatalf("overwritten cell %d hyperlink = %q, want empty", col, got)
		}
		if got := cell.HyperlinkParams; got != "" {
			t.Fatalf("overwritten cell %d hyperlink params = %q, want empty", col, got)
		}
	}
}

func TestOSC133PromptAndInputMarkSemanticCells(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P;k=i")
	vt.update(testPrint("$"))
	vt.osc("133;B")
	vt.update(testPrint("x"))

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("row semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticPromptContent; got != want {
		t.Fatalf("prompt cell semantic content = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 1).semanticContent, semanticInput; got != want {
		t.Fatalf("input cell semantic content = %d, want %d", got, want)
	}
}

func TestOSC133FreshLineNewPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	vt.update(testPrint("cmd"))
	vt.osc("133;A;k=s")
	vt.update(testPrint(">"))

	if vt.cursor.row != 1 || vt.cursor.col != 1 {
		t.Fatalf("cursor after fresh-line prompt = %d,%d, want 1,1", vt.cursor.row, vt.cursor.col)
	}
	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptContinuation; got != want {
		t.Fatalf("row semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(1, 0).semanticContent, semanticPromptContent; got != want {
		t.Fatalf("prompt cell semantic content = %d, want %d", got, want)
	}
}

func TestOSC133FreshLineNewPromptRedrawOption(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawTrue; got != want {
		t.Fatalf("default redraw = %d, want %d", got, want)
	}
	vt.osc("133;A;redraw=0")
	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawFalse; got != want {
		t.Fatalf("redraw=0 = %d, want %d", got, want)
	}
	vt.osc("133;A;redraw=1")
	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawTrue; got != want {
		t.Fatalf("redraw=1 = %d, want %d", got, want)
	}
	vt.osc("133;A;redraw=last")
	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawLast; got != want {
		t.Fatalf("redraw=last = %d, want %d", got, want)
	}
	vt.osc("133;A;redraw=x")
	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawLast; got != want {
		t.Fatalf("invalid redraw changed state to %d, want %d", got, want)
	}
}

func TestOSC133RedrawOptionOnlyAppliesToFreshLineNewPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	vt.osc("133;P;redraw=0")
	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawTrue; got != want {
		t.Fatalf("prompt_start redraw = %d, want %d", got, want)
	}
}

func TestOSC133ClickEventsOption(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	vt.osc("133;A;click_events=1")
	if got, want := vt.semanticPromptClick, semanticPromptClickEvents; got != want {
		t.Fatalf("click_events=1 click = %d, want %d", got, want)
	}
	vt.osc("133;A;click_events=0;cl=v")
	if got, want := vt.semanticPromptClick, semanticPromptClickConservativeVertical; got != want {
		t.Fatalf("click_events=0 fallback click = %d, want %d", got, want)
	}
	vt.osc("133;A;click_events=1;cl=m")
	if got, want := vt.semanticPromptClick, semanticPromptClickEvents; got != want {
		t.Fatalf("click_events priority click = %d, want %d", got, want)
	}
}

func TestOSC133ClickLineOption(t *testing.T) {
	tests := []struct {
		option string
		want   semanticPromptClick
	}{
		{option: "cl=line", want: semanticPromptClickLine},
		{option: "cl=m", want: semanticPromptClickMultiple},
		{option: "cl=v", want: semanticPromptClickConservativeVertical},
		{option: "cl=w", want: semanticPromptClickSmartVertical},
	}
	for _, tt := range tests {
		t.Run(tt.option, func(t *testing.T) {
			vt := New()
			vt.resize(8, 3)

			vt.osc("133;A;" + tt.option)

			if got := vt.semanticPromptClick; got != tt.want {
				t.Fatalf("click option = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOSC133InvalidClickOptionsDoNotChangeState(t *testing.T) {
	vt := New()
	vt.resize(8, 3)
	vt.semanticPromptClick = semanticPromptClickLine

	vt.osc("133;A;click_events=yes;cl=invalid")

	if got, want := vt.semanticPromptClick, semanticPromptClickLine; got != want {
		t.Fatalf("invalid click options changed state to %d, want %d", got, want)
	}
}

func TestOSC133ClickOptionsOnlyApplyToFreshLineNewPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	vt.osc("133;P;click_events=1;cl=m")

	if got, want := vt.semanticPromptClick, semanticPromptClickNone; got != want {
		t.Fatalf("prompt_start click = %d, want %d", got, want)
	}
}

func TestRISResetsOSC133RedrawOption(t *testing.T) {
	vt := New()
	vt.resize(8, 3)

	vt.osc("133;A;redraw=0")
	vt.osc("133;A;click_events=1")
	vt.update(testESC('c'))

	if got, want := vt.shellRedrawsPrompt, semanticPromptRedrawTrue; got != want {
		t.Fatalf("redraw after RIS = %d, want %d", got, want)
	}
	if got, want := vt.semanticPromptClick, semanticPromptClickNone; got != want {
		t.Fatalf("click after RIS = %d, want %d", got, want)
	}
}

func TestOSC133EndInputStartOutputClearsPromptAtColumnZero(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.cursor.col = 0
	vt.osc("133;C")
	vt.update(testPrint("o"))

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("output cell semantic content = %d, want %d", got, want)
	}
}

func TestOSC133EndInputOnNewlineClearsEOLInput(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;I")
	vt.update(testPrint("x"))
	vt.lf()
	vt.update(testPrint("o"))

	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticInput; got != want {
		t.Fatalf("input cell semantic content = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(1, 1).semanticContent, semanticOutput; got != want {
		t.Fatalf("output cell semantic content = %d, want %d", got, want)
	}
}

func TestOSC133InvalidCommandIgnored(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;Pextra")
	vt.update(testPrint("x"))

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("cell semantic content = %d, want %d", got, want)
	}
}

func TestOSC133FreshLineWithOptionsIgnored(t *testing.T) {
	vt := New()
	vt.resize(8, 2)
	printText(vt, "cmd")

	vt.osc("133;L;ignored")

	if got, want := vt.cursor.row, row(0); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestOSC133EraseCharacterClearsSemanticContent(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.update(testPrint("$"))
	vt.cursor.col = 0
	vt.update(testCSI('X', []uint32{1}))

	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("erased cell semantic content = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("partial erase row semantic prompt = %d, want %d", got, want)
	}
}

func TestOSC133CompleteLineErasePreservesSemanticPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.update(testPrint("$"))
	vt.update(testCSI('K', []uint32{2}))

	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("erased cell semantic content = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("erased line semantic prompt = %d, want %d", got, want)
	}
}

func TestOSC133CompleteDisplayEraseClearsSemanticPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.update(testPrint("$"))
	vt.update(testCSI('J', []uint32{2}))

	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("erased cell semantic content = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("erased display semantic prompt = %d, want %d", got, want)
	}
}

func TestOSC133PromptNewlineMarksContinuation(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	printText(vt, "hello")
	vt.cr()
	vt.lf()

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("row 0 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptContinuation; got != want {
		t.Fatalf("row 1 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.cursor.semanticContent, semanticPromptContent; got != want {
		t.Fatalf("cursor semantic content = %d, want %d", got, want)
	}
}

func TestOSC133InputNewlineMarksContinuation(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	printText(vt, "$ ")
	vt.osc("133;B")
	printText(vt, "echo")
	vt.cr()
	vt.lf()

	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptContinuation; got != want {
		t.Fatalf("row 1 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.cursor.semanticContent, semanticInput; got != want {
		t.Fatalf("cursor semantic content = %d, want %d", got, want)
	}
}

func TestOSC133OutputNewlineDoesNotMarkContinuation(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	printText(vt, "$ ")
	vt.osc("133;B")
	printText(vt, "ls")
	vt.osc("133;C")
	vt.cr()
	vt.lf()

	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row 1 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.cursor.semanticContent, semanticOutput; got != want {
		t.Fatalf("cursor semantic content = %d, want %d", got, want)
	}
}

func TestOSC133EndInputStartOutputDoesNotClearPromptAwayFromColumnZero(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	printText(vt, "$ ")
	vt.cr()
	vt.lf()
	vt.osc("133;P;k=c")
	printText(vt, "> ")
	vt.osc("133;C")

	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptContinuation; got != want {
		t.Fatalf("row 1 semantic prompt = %d, want %d", got, want)
	}
}

func TestOSC133MultiplePromptNewlinesMarkContinuations(t *testing.T) {
	vt := New()
	vt.resize(10, 5)

	vt.osc("133;P")
	printText(vt, "line1")
	vt.cr()
	vt.lf()
	printText(vt, "line2")
	vt.cr()
	vt.lf()
	printText(vt, "line3")

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("row 0 semantic prompt = %d, want %d", got, want)
	}
	for r := row(1); r <= 2; r += 1 {
		if got, want := vt.primaryScreen.row(r).semanticPrompt, semanticPromptContinuation; got != want {
			t.Fatalf("row %d semantic prompt = %d, want %d", r, got, want)
		}
	}
}

func TestOSC133CursorIsAtPrompt(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	if vt.cursorIsAtPrompt() {
		t.Fatal("fresh terminal cursor reported at prompt")
	}
	vt.osc("133;P")
	if !vt.cursorIsAtPrompt() {
		t.Fatal("prompt cursor did not report at prompt")
	}
	printText(vt, "$ ")
	vt.osc("133;B")
	if !vt.cursorIsAtPrompt() {
		t.Fatal("input cursor did not report at prompt")
	}
	printText(vt, "ls")
	vt.osc("133;C")
	if !vt.cursorIsAtPrompt() {
		t.Fatal("prompt row did not report at prompt")
	}
	vt.lf()
	if vt.cursorIsAtPrompt() {
		t.Fatal("output row reported at prompt")
	}
}

func TestOSC133CursorIsAtPromptIgnoresAlternateScreen(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	if !vt.cursorIsAtPrompt() {
		t.Fatal("primary prompt cursor did not report at prompt")
	}
	vt.decset(testCSI('h', []uint32{1049}, '?'))
	if vt.cursorIsAtPrompt() {
		t.Fatal("alternate screen reported primary prompt")
	}
	vt.osc("133;P")
	if vt.cursorIsAtPrompt() {
		t.Fatal("alternate screen semantic prompt reported at prompt")
	}
}

func TestOSC133AlternateScreenKeepsSemanticCellTagging(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	vt.osc("133;P")
	vt.decset(testCSI('h', []uint32{1047}, '?'))
	vt.update(testPrint("$"))

	if vt.cursorIsAtPrompt() {
		t.Fatal("alternate screen reported prompt despite smcup")
	}
	if got, want := vt.activeScreen.cell(0, 0).semanticContent, semanticPromptContent; got != want {
		t.Fatalf("alternate screen semantic content = %d, want %d", got, want)
	}
}

func TestRISClearsSemanticPromptState(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.update(testPrint("$"))
	vt.osc("133;B")
	vt.update(testESC('c'))
	vt.update(testPrint("x"))

	if got, want := vt.cursor.semanticContent, semanticOutput; got != want {
		t.Fatalf("cursor semantic content after RIS = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row semantic prompt after RIS = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("cell semantic content after RIS = %d, want %d", got, want)
	}
	if vt.cursorIsAtPrompt() {
		t.Fatal("cursor reported at prompt after RIS")
	}
}

func TestSaveRestoreCursorDoesNotRestoreSemanticPromptState(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;P")
	vt.update(testESC('7'))
	vt.osc("133;C")
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	if got, want := vt.cursor.semanticContent, semanticOutput; got != want {
		t.Fatalf("cursor semantic content after restore = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticOutput; got != want {
		t.Fatalf("cell semantic content after restore = %d, want %d", got, want)
	}
}

func TestRestoreUnsavedCursorDoesNotResetSemanticPromptState(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("133;B")
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	if got, want := vt.cursor.semanticContent, semanticInput; got != want {
		t.Fatalf("cursor semantic content after unsaved restore = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(0, 0).semanticContent, semanticInput; got != want {
		t.Fatalf("cell semantic content after unsaved restore = %d, want %d", got, want)
	}
}

func TestOSC7WorkingDirectoryEvent(t *testing.T) {
	vt := New()

	vt.osc("7;file:///tmp/example")

	if got, want := vt.workingDirectoryURL, "file:///tmp/example"; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	ev := <-vt.events
	pwd, ok := ev.(EventWorkingDirectory)
	if !ok {
		t.Fatalf("event = %T, want EventWorkingDirectory", ev)
	}
	if got, want := pwd.URL, "file:///tmp/example"; got != want {
		t.Fatalf("event URL = %q, want %q", got, want)
	}
}

func TestOSC7WorkingDirectoryEmpty(t *testing.T) {
	vt := New()
	vt.workingDirectoryURL = "file:///tmp/old"

	vt.osc("7;")

	if got, want := vt.workingDirectoryURL, ""; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	ev := <-vt.events
	pwd, ok := ev.(EventWorkingDirectory)
	if !ok {
		t.Fatalf("event = %T, want EventWorkingDirectory", ev)
	}
	if pwd.URL != "" {
		t.Fatalf("event URL = %q, want empty", pwd.URL)
	}
}

func TestOSC1337CurrentDirReportsWorkingDirectory(t *testing.T) {
	vt := New()

	vt.osc("1337;CurrentDir=file:///tmp/example")

	if got, want := vt.workingDirectoryURL, "file:///tmp/example"; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	ev := <-vt.events
	pwd, ok := ev.(EventWorkingDirectory)
	if !ok {
		t.Fatalf("event = %T, want EventWorkingDirectory", ev)
	}
	if got, want := pwd.URL, "file:///tmp/example"; got != want {
		t.Fatalf("event URL = %q, want %q", got, want)
	}
}

func TestOSC1337CurrentDirIsCaseInsensitive(t *testing.T) {
	vt := New()

	vt.osc("1337;currentdir=file:///tmp/example")

	if got, want := vt.workingDirectoryURL, "file:///tmp/example"; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
}

func TestOSC1337CurrentDirRequiresNonEmptyValue(t *testing.T) {
	vt := New()
	vt.workingDirectoryURL = "file:///tmp/old"

	vt.osc("1337;CurrentDir")
	vt.osc("1337;CurrentDir=")

	if got, want := vt.workingDirectoryURL, "file:///tmp/old"; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for invalid CurrentDir: %T", ev)
	default:
	}
}

func TestOSC1337CopyInvalidOrWithoutVaxisDoesNotPanic(t *testing.T) {
	vt := New()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("OSC 1337 Copy panicked: %v", r)
		}
	}()

	vt.osc("1337;Copy")
	vt.osc("1337;Copy=")
	vt.osc("1337;Copy=:")
	vt.osc("1337;Copy=:?")
	vt.osc("1337;Copy=YWJj")
	vt.osc("1337;Copy=:not-base64")
	vt.osc("1337;Copy=:YWJj")
}

func TestOSC9ConEmuWorkingDirectory(t *testing.T) {
	vt := New()

	vt.osc("9;9;file:///tmp/example")

	if got, want := vt.workingDirectoryURL, "file:///tmp/example"; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	ev := <-vt.events
	pwd, ok := ev.(EventWorkingDirectory)
	if !ok {
		t.Fatalf("event = %T, want EventWorkingDirectory", ev)
	}
	if got, want := pwd.URL, "file:///tmp/example"; got != want {
		t.Fatalf("event URL = %q, want %q", got, want)
	}
}

func TestOSC9IncompleteConEmuWorkingDirectoryIsNotification(t *testing.T) {
	vt := New()

	vt.osc("9;9")

	if got, want := vt.workingDirectoryURL, ""; got != want {
		t.Fatalf("working directory URL = %q, want %q", got, want)
	}
	ev := <-vt.events
	notify, ok := ev.(EventNotify)
	if !ok {
		t.Fatalf("event = %T, want EventNotify", ev)
	}
	if got, want := notify.Body, "9"; got != want {
		t.Fatalf("notification body = %q, want %q", got, want)
	}
}

func TestOSC9ConEmuProgressReport(t *testing.T) {
	tests := []struct {
		name        string
		osc         string
		state       ProgressState
		progress    int
		hasProgress bool
	}{
		{name: "set", osc: "9;4;1;50", state: ProgressSet, progress: 50, hasProgress: true},
		{name: "set default zero", osc: "9;4;1", state: ProgressSet, progress: 0, hasProgress: true},
		{name: "set clamps", osc: "9;4;1;500", state: ProgressSet, progress: 100, hasProgress: true},
		{name: "remove", osc: "9;4;0;50", state: ProgressRemove},
		{name: "error", osc: "9;4;2", state: ProgressError},
		{name: "error with progress", osc: "9;4;2;75", state: ProgressError, progress: 75, hasProgress: true},
		{name: "indeterminate", osc: "9;4;3;75", state: ProgressIndeterminate},
		{name: "pause", osc: "9;4;4;25", state: ProgressPause, progress: 25, hasProgress: true},
		{name: "invalid progress value", osc: "9;4;1;nope", state: ProgressSet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()

			vt.osc(tt.osc)

			ev := <-vt.events
			progress, ok := ev.(EventProgress)
			if !ok {
				t.Fatalf("event = %T, want EventProgress", ev)
			}
			if progress.State != tt.state {
				t.Fatalf("progress state = %d, want %d", progress.State, tt.state)
			}
			if progress.Progress != tt.progress {
				t.Fatalf("progress value = %d, want %d", progress.Progress, tt.progress)
			}
			if progress.HasProgress != tt.hasProgress {
				t.Fatalf("has progress = %v, want %v", progress.HasProgress, tt.hasProgress)
			}
		})
	}
}

func TestOSC9ConEmuPromptStartMarksSemanticPrompt(t *testing.T) {
	vt := New()
	vt.resize(8, 2)
	printText(vt, "cmd")

	vt.osc("9;12")
	printText(vt, ">")

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.cell(1, 0).semanticContent, semanticPromptContent; got != want {
		t.Fatalf("semantic content = %d, want %d", got, want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for OSC 9;12: %T", ev)
	default:
	}
}

func TestOSC9ConEmuPromptStartIgnoresTrailingData(t *testing.T) {
	vt := New()
	vt.resize(8, 2)

	vt.osc("9;12;ignored")
	printText(vt, ">")

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("semantic prompt = %d, want %d", got, want)
	}
}

func TestOSC9IncompleteConEmuProgressIsNotification(t *testing.T) {
	vt := New()

	vt.osc("9;4")

	ev := <-vt.events
	notify, ok := ev.(EventNotify)
	if !ok {
		t.Fatalf("event = %T, want EventNotify", ev)
	}
	if got, want := notify.Body, "4"; got != want {
		t.Fatalf("notification body = %q, want %q", got, want)
	}
}

func TestOSC9Notification(t *testing.T) {
	vt := New()

	vt.osc("9;hello")

	ev := <-vt.events
	notify, ok := ev.(EventNotify)
	if !ok {
		t.Fatalf("event = %T, want EventNotify", ev)
	}
	if got, want := notify.Body, "hello"; got != want {
		t.Fatalf("notification body = %q, want %q", got, want)
	}
}

func TestOSC777RxvtNotification(t *testing.T) {
	vt := New()

	vt.osc("777;notify;Title;Body")

	ev := <-vt.events
	notify, ok := ev.(EventNotify)
	if !ok {
		t.Fatalf("event = %T, want EventNotify", ev)
	}
	if got, want := notify.Title, "Title"; got != want {
		t.Fatalf("notification title = %q, want %q", got, want)
	}
	if got, want := notify.Body, "Body"; got != want {
		t.Fatalf("notification body = %q, want %q", got, want)
	}
}

func TestRISClearsWorkingDirectory(t *testing.T) {
	vt := New()
	vt.resize(80, 24)
	vt.osc("7;file:///tmp/example")
	<-vt.events

	vt.update(testESC('c'))

	if got, want := vt.workingDirectoryURL, ""; got != want {
		t.Fatalf("working directory URL after RIS = %q, want %q", got, want)
	}
}

func TestRISResetsTerminalColors(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.osc("4;1;rgb:ff/00/00")
	vt.osc("10;rgb:00/ff/00")
	vt.osc("11;rgb:00/00/ff")
	vt.osc("12;rgb:01/02/03")
	vt.update(testESC('c'))

	if vt.colors.paletteSet(1) {
		t.Fatal("palette color stayed set after RIS")
	}
	if vt.colors.foreground.set || vt.colors.background.set || vt.colors.cursor.set {
		t.Fatal("dynamic colors stayed set after RIS")
	}
}

func TestOSC22MouseShapeEvent(t *testing.T) {
	vt := New()

	vt.osc("22;pointer")

	if got, want := vt.mouseShape, vaxis.MouseShapeClickable; got != want {
		t.Fatalf("mouse shape = %q, want %q", got, want)
	}
	ev := <-vt.events
	shape, ok := ev.(EventMouseShape)
	if !ok {
		t.Fatalf("event = %T, want EventMouseShape", ev)
	}
	if got, want := shape.Shape, vaxis.MouseShapeClickable; got != want {
		t.Fatalf("event shape = %q, want %q", got, want)
	}
}

func TestOSC22MouseShapeGhosttyNames(t *testing.T) {
	tests := []struct {
		name string
		want vaxis.MouseShape
	}{
		{"crosshair", vaxis.MouseShapeCrosshair},
		{"vertical-text", vaxis.MouseShapeVerticalText},
		{"not-allowed", vaxis.MouseShapeNotAllowed},
		{"nesw-resize", vaxis.MouseShapeResizeNESW},
		{"zoom-out", vaxis.MouseShapeZoomOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shape, ok := parseMouseShape(tt.name)
			if !ok {
				t.Fatal("shape was not accepted")
			}
			if shape != tt.want {
				t.Fatalf("shape = %q, want %q", shape, tt.want)
			}
		})
	}
}

func TestOSC22MouseShapeAliases(t *testing.T) {
	tests := []struct {
		name string
		want vaxis.MouseShape
	}{
		{"left_ptr", vaxis.MouseShapeDefault},
		{"question_arrow", vaxis.MouseShapeHelp},
		{"hand", vaxis.MouseShapeClickable},
		{"cross", vaxis.MouseShapeCrosshair},
		{"xterm", vaxis.MouseShapeTextInput},
		{"dnd-copy", vaxis.MouseShapeCopy},
		{"crossed_circle", vaxis.MouseShapeNotAllowed},
		{"fleur", vaxis.MouseShapeAllScroll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shape, ok := parseMouseShape(tt.name)
			if !ok {
				t.Fatal("shape alias was not accepted")
			}
			if shape != tt.want {
				t.Fatalf("shape = %q, want %q", shape, tt.want)
			}
		})
	}
}

func TestNewTerminalUsesTextMouseShape(t *testing.T) {
	vt := New()

	if got, want := vt.mouseShape, vaxis.MouseShapeTextInput; got != want {
		t.Fatalf("mouse shape = %q, want %q", got, want)
	}
}

func TestRISResetsMouseShape(t *testing.T) {
	vt := New()
	vt.resize(80, 24)
	vt.osc("22;pointer")
	<-vt.events

	vt.update(testESC('c'))

	if got, want := vt.mouseShape, vaxis.MouseShapeTextInput; got != want {
		t.Fatalf("mouse shape after RIS = %q, want %q", got, want)
	}
}

func TestOSC22UnknownMouseShapeIgnored(t *testing.T) {
	vt := New()
	vt.mouseShape = vaxis.MouseShapeTextInput

	vt.osc("22;definitely-not-a-cursor")

	if got, want := vt.mouseShape, vaxis.MouseShapeTextInput; got != want {
		t.Fatalf("mouse shape = %q, want %q", got, want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for unknown mouse shape: %T", ev)
	default:
	}
}

func TestOSC4ColorQueryWithoutVaxisDoesNotReply(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.osc("4;8;?")

	assertNoReply(t, r)
}

func TestOSC4MalformedColorQueryIgnored(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.osc("4;8;?;bad")
	vt.osc("4;300;?")
	vt.osc("4;8;rgb:not-a-color")

	assertNoReply(t, r)
	if vt.colors.paletteSet(8) {
		t.Fatal("malformed OSC 4 set palette color")
	}
}

func TestOSC4SetAndResetPaletteColor(t *testing.T) {
	vt := New()

	vt.osc("4;8; red ")
	if got, want := vt.colors.palette[8], vaxis.RGBColor(255, 0, 0); got != want {
		t.Fatalf("palette color = %v, want %v", got, want)
	}
	if !vt.colors.paletteSet(8) {
		t.Fatal("palette color mask was not set")
	}
	if !vt.colors.paletteDirty {
		t.Fatal("palette dirty flag was not set")
	}

	vt.colors.paletteDirty = false
	vt.osc("104;8")
	if got := vt.colors.palette[8]; got != 0 {
		t.Fatalf("palette color after reset = %v, want default", got)
	}
	if vt.colors.paletteSet(8) {
		t.Fatal("palette color mask stayed set after reset")
	}
	if !vt.colors.paletteDirty {
		t.Fatal("palette dirty flag was not set after reset")
	}
}

func TestOSC4MultipleNamedColorRequests(t *testing.T) {
	vt := New()

	vt.osc("4;0;red;1;blue;2;ForestGreen;3;medium spring green")

	tests := []struct {
		index uint8
		color vaxis.Color
	}{
		{index: 0, color: vaxis.RGBColor(255, 0, 0)},
		{index: 1, color: vaxis.RGBColor(0, 0, 255)},
		{index: 2, color: vaxis.RGBColor(34, 139, 34)},
		{index: 3, color: vaxis.RGBColor(0, 250, 154)},
	}
	for _, tt := range tests {
		if got := vt.colors.palette[tt.index]; got != tt.color {
			t.Fatalf("palette %d = %v, want %v", tt.index, got, tt.color)
		}
	}
}

func TestOSC4KeepsValidPairsBeforeInvalidPair(t *testing.T) {
	vt := New()

	vt.osc("4;1;rgb:01/02/03;2;rgb:not-a-color;3;rgb:04/05/06")

	if got, want := vt.colors.palette[1], vaxis.RGBColor(1, 2, 3); got != want {
		t.Fatalf("palette 1 = %v, want %v", got, want)
	}
	if vt.colors.paletteSet(2) {
		t.Fatal("invalid palette pair was applied")
	}
	if vt.colors.paletteSet(3) {
		t.Fatal("palette pair after invalid pair was applied")
	}
}

func TestOSC4KeepsValidPairBeforeMissingTrailingSpec(t *testing.T) {
	vt := New()

	vt.osc("4;1;rgb:01/02/03;2")

	if got, want := vt.colors.palette[1], vaxis.RGBColor(1, 2, 3); got != want {
		t.Fatalf("palette 1 = %v, want %v", got, want)
	}
	if vt.colors.paletteSet(2) {
		t.Fatal("palette with missing spec was applied")
	}
}

func TestOSC4SpecialColorsAreExplicitNoops(t *testing.T) {
	vt := New()

	vt.osc("4;1;red;256;blue;260;?;2;green")

	if got, want := vt.colors.palette[1], vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.paletteSet(1) {
		t.Fatalf("palette 1 = %v set %v, want %v set", got, vt.colors.paletteSet(1), want)
	}
	if got, want := vt.colors.palette[2], vaxis.RGBColor(0, 255, 0); got != want || !vt.colors.paletteSet(2) {
		t.Fatalf("palette 2 = %v set %v, want %v set", got, vt.colors.paletteSet(2), want)
	}
	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for OSC 4 special colors: %T", ev)
	default:
	}
}

func TestOSC4InvalidSpecialColorStopsParsing(t *testing.T) {
	vt := New()

	vt.osc("4;1;red;261;blue;2;green")

	if got, want := vt.colors.palette[1], vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.paletteSet(1) {
		t.Fatalf("palette 1 = %v set %v, want %v set", got, vt.colors.paletteSet(1), want)
	}
	if vt.colors.paletteSet(2) {
		t.Fatal("palette pair after invalid special color was applied")
	}
}

func TestOSC5And105SpecialColorsAreExplicitNoops(t *testing.T) {
	vt := New()

	vt.osc("5;0;red;4;?")
	vt.osc("105")
	vt.osc("105;0;4")

	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for OSC 5/105 special colors: %T", ev)
	default:
	}
}

func TestOSC104WithoutParametersResetsAllPaletteColors(t *testing.T) {
	vt := New()

	vt.osc("4;1;rgb:01/02/03")
	vt.osc("4;2;rgb:04/05/06")
	vt.colors.paletteDirty = false
	vt.osc("104")

	if vt.colors.paletteSet(1) || vt.colors.paletteSet(2) {
		t.Fatal("palette masks stayed set after OSC 104")
	}
	if got := vt.colors.palette[1]; got != 0 {
		t.Fatalf("palette 1 after OSC 104 = %v, want default", got)
	}
	if got := vt.colors.palette[2]; got != 0 {
		t.Fatalf("palette 2 after OSC 104 = %v, want default", got)
	}
	if !vt.colors.paletteDirty {
		t.Fatal("palette dirty flag was not set after OSC 104")
	}
}

func TestOSC104OnlyEmptyIndexesResetsAllPaletteColors(t *testing.T) {
	vt := New()

	vt.osc("4;1;rgb:01/02/03")
	vt.osc("4;2;rgb:04/05/06")
	vt.colors.paletteDirty = false
	vt.osc("104;;")

	if vt.colors.paletteSet(1) || vt.colors.paletteSet(2) {
		t.Fatal("palette masks stayed set after OSC 104 empty indexes")
	}
	if !vt.colors.paletteDirty {
		t.Fatal("palette dirty flag was not set after OSC 104 empty indexes")
	}
}

func TestOSC104WithoutPaletteOverridesDoesNotMarkDirty(t *testing.T) {
	vt := New()

	vt.osc("104")

	if vt.colors.paletteDirty {
		t.Fatal("OSC 104 marked clean palette dirty")
	}
}

func TestOSCDynamicColorQueriesWithoutVaxisDoNotReply(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.osc("10;?")
	vt.osc("11;?")

	assertNoReply(t, r)
}

func TestOSCDynamicColorSetAndReset(t *testing.T) {
	vt := New()

	vt.osc("10;red;blue;#040506")

	if got, want := vt.colors.foreground.color, vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.foreground.set {
		t.Fatalf("foreground = %v set %v, want %v set", got, vt.colors.foreground.set, want)
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(0, 0, 255); got != want || !vt.colors.background.set {
		t.Fatalf("background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if got, want := vt.colors.cursor.color, vaxis.RGBColor(4, 5, 6); got != want || !vt.colors.cursor.set {
		t.Fatalf("cursor = %v set %v, want %v set", got, vt.colors.cursor.set, want)
	}

	vt.osc("110")
	vt.osc("111")
	vt.osc("112")
	if vt.colors.foreground.set || vt.colors.background.set || vt.colors.cursor.set {
		t.Fatal("dynamic colors stayed set after reset")
	}
}

func TestOSCDynamicColorResetAllowsOnlyEmptyParameters(t *testing.T) {
	vt := New()

	vt.osc("10;red;blue;#040506")
	vt.osc("110; ")
	vt.osc("111;ignored")
	if !vt.colors.foreground.set {
		t.Fatal("OSC 110 with whitespace reset foreground")
	}
	if !vt.colors.background.set {
		t.Fatal("OSC 111 with parameter reset background")
	}

	vt.osc("110;")
	vt.osc("111;;")
	vt.osc("112;")
	if vt.colors.foreground.set || vt.colors.background.set || vt.colors.cursor.set {
		t.Fatal("dynamic colors stayed set after empty reset parameters")
	}
}

func TestOSCDynamicColorSetStartsAtSelector(t *testing.T) {
	vt := New()

	vt.osc("11;red;blue")

	if vt.colors.foreground.set {
		t.Fatal("OSC 11 set foreground")
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.background.set {
		t.Fatalf("background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if got, want := vt.colors.cursor.color, vaxis.RGBColor(0, 0, 255); got != want || !vt.colors.cursor.set {
		t.Fatalf("cursor = %v set %v, want %v set", got, vt.colors.cursor.set, want)
	}
}

func TestOSCDynamicColorSkipsEmptySegments(t *testing.T) {
	vt := New()

	vt.osc("10;;red;;;blue")

	if got, want := vt.colors.foreground.color, vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.foreground.set {
		t.Fatalf("foreground = %v set %v, want %v set", got, vt.colors.foreground.set, want)
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(0, 0, 255); got != want || !vt.colors.background.set {
		t.Fatalf("background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if vt.colors.cursor.set {
		t.Fatal("empty dynamic color segments advanced target color")
	}
}

func TestOSCDynamicColorSupportsGhosttyDynamicRange(t *testing.T) {
	vt := New()

	vt.osc("13;red;blue;#010203;#040506;#070809;#0a0b0c;#0d0e0f")

	tests := []struct {
		name string
		got  dynamicColor
		want vaxis.Color
	}{
		{name: "pointer foreground", got: vt.colors.pointerForeground, want: vaxis.RGBColor(255, 0, 0)},
		{name: "pointer background", got: vt.colors.pointerBackground, want: vaxis.RGBColor(0, 0, 255)},
		{name: "tektronix foreground", got: vt.colors.tektronixForeground, want: vaxis.RGBColor(1, 2, 3)},
		{name: "tektronix background", got: vt.colors.tektronixBackground, want: vaxis.RGBColor(4, 5, 6)},
		{name: "highlight background", got: vt.colors.highlightBackground, want: vaxis.RGBColor(7, 8, 9)},
		{name: "tektronix cursor", got: vt.colors.tektronixCursor, want: vaxis.RGBColor(10, 11, 12)},
		{name: "highlight foreground", got: vt.colors.highlightForeground, want: vaxis.RGBColor(13, 14, 15)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.color != tt.want || !tt.got.set {
				t.Fatalf("color = %v set %v, want %v set", tt.got.color, tt.got.set, tt.want)
			}
		})
	}

	vt.osc("113")
	vt.osc("114")
	vt.osc("115")
	vt.osc("116")
	vt.osc("117")
	vt.osc("118")
	vt.osc("119")
	if vt.colors.pointerForeground.set || vt.colors.pointerBackground.set ||
		vt.colors.tektronixForeground.set || vt.colors.tektronixBackground.set ||
		vt.colors.highlightBackground.set || vt.colors.tektronixCursor.set ||
		vt.colors.highlightForeground.set {
		t.Fatal("extended dynamic colors stayed set after reset")
	}
}

func TestOSCDynamicColorStopsAfterGhosttyDynamicRange(t *testing.T) {
	vt := New()

	vt.osc("19;red;blue")

	if got, want := vt.colors.highlightForeground.color, vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.highlightForeground.set {
		t.Fatalf("highlight foreground = %v set %v, want %v set", got, vt.colors.highlightForeground.set, want)
	}
	if vt.colors.foreground.set {
		t.Fatal("OSC 19 wrapped around to foreground")
	}
}

func TestOSCDynamicColorMultiValueQueryDoesNotStopLaterValues(t *testing.T) {
	vt := New()

	vt.osc("10;?;red;blue")

	if vt.colors.foreground.set {
		t.Fatal("dynamic foreground query changed foreground state")
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(255, 0, 0); got != want || !vt.colors.background.set {
		t.Fatalf("background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if got, want := vt.colors.cursor.color, vaxis.RGBColor(0, 0, 255); got != want || !vt.colors.cursor.set {
		t.Fatalf("cursor = %v set %v, want %v set", got, vt.colors.cursor.set, want)
	}
}

func TestOSC21KittyColorProtocolSetAndReset(t *testing.T) {
	vt := New()

	vt.osc("21;5=rgb:ff/00/ff;foreground=rgbi:1.0/0/0.5;background=#010203;cursor=rgb:04/05/06")

	if got, want := vt.colors.palette[5], vaxis.RGBColor(0xff, 0, 0xff); got != want || !vt.colors.paletteSet(5) {
		t.Fatalf("kitty palette = %v set %v, want %v set", got, vt.colors.paletteSet(5), want)
	}
	if got, want := vt.colors.foreground.color, vaxis.RGBColor(0xff, 0, 0x7f); got != want || !vt.colors.foreground.set {
		t.Fatalf("kitty foreground = %v set %v, want %v set", got, vt.colors.foreground.set, want)
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(1, 2, 3); got != want || !vt.colors.background.set {
		t.Fatalf("kitty background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if got, want := vt.colors.cursor.color, vaxis.RGBColor(4, 5, 6); got != want || !vt.colors.cursor.set {
		t.Fatalf("kitty cursor = %v set %v, want %v set", got, vt.colors.cursor.set, want)
	}

	vt.osc("21;5=;foreground=;background=;cursor=")
	if vt.colors.paletteSet(5) || vt.colors.foreground.set || vt.colors.background.set || vt.colors.cursor.set {
		t.Fatal("kitty color reset left colors set")
	}
}

func TestOSC21KittyColorProtocolSkipsQueriesAndInvalidEntries(t *testing.T) {
	vt := New()

	vt.osc("21;foreground=?;background=rgb:f0/f8/ff;cursor=aliceblue;cursor_text;visual_bell=;selection_foreground=#xxxyyzz;selection_background=?;selection_background=#aabbcc;2=?;3=rgbi:1.0/1.0/1.0")

	if vt.colors.foreground.set {
		t.Fatal("kitty foreground query changed foreground state")
	}
	if got, want := vt.colors.background.color, vaxis.RGBColor(0xf0, 0xf8, 0xff); got != want || !vt.colors.background.set {
		t.Fatalf("kitty background = %v set %v, want %v set", got, vt.colors.background.set, want)
	}
	if got, want := vt.colors.cursor.color, vaxis.RGBColor(0xf0, 0xf8, 0xff); got != want || !vt.colors.cursor.set {
		t.Fatalf("kitty cursor = %v set %v, want %v set", got, vt.colors.cursor.set, want)
	}
	if vt.colors.paletteSet(2) {
		t.Fatal("kitty palette query changed palette state")
	}
	if got, want := vt.colors.palette[3], vaxis.RGBColor(0xff, 0xff, 0xff); got != want || !vt.colors.paletteSet(3) {
		t.Fatalf("kitty palette 3 = %v set %v, want %v set", got, vt.colors.paletteSet(3), want)
	}
}

func TestOSC52IgnoredWithoutVaxis(t *testing.T) {
	vt := New()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("OSC 52 panicked without Vaxis: %v", r)
		}
	}()

	vt.osc("52;c;YWJj")
}

func TestOSC52InvalidBase64DoesNotPanic(t *testing.T) {
	vt := New()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("OSC 52 invalid base64 panicked: %v", r)
		}
	}()

	vt.osc("52;c;?")
}

func TestOSC52DataParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "clipboard", input: "c;YWJj", want: "YWJj", ok: true},
		{name: "selection", input: "s;YWJj", want: "YWJj", ok: true},
		{name: "optional kind", input: ";YWJj", want: "YWJj", ok: true},
		{name: "clear optional kind", input: ";", want: "", ok: true},
		{name: "query optional kind", input: ";?", want: "?", ok: true},
		{name: "empty", input: "", ok: false},
		{name: "missing separator", input: "c", ok: false},
		{name: "bad separator", input: "c:YWJj", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := osc52Data(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("data = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOSC52QueryIsUnsupportedNoop(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.osc("52;c;?")
	vt.osc("52;;?")

	assertNoReply(t, r)
}

func TestOSC52MalformedIgnored(t *testing.T) {
	vt := New()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("OSC 52 malformed input panicked: %v", r)
		}
	}()

	vt.osc("52;")
	vt.osc("52;c")
	vt.osc("52;c:YWJj")
}

func TestParsedButUnsupportedGhosttyOSCProtocolsAreExplicitNoops(t *testing.T) {
	vt := New()

	vt.osc("66;s=2;large text")
	vt.osc("3008;start=myctx;type=shell")
	vt.osc("5522;type=read:status=OK")

	select {
	case ev := <-vt.events:
		t.Fatalf("unexpected event for unsupported Ghostty OSC protocol: %T", ev)
	default:
	}
}
