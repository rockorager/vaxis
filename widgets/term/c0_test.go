package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestLinefeedModePerformsCarriageReturn(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	vt.mode.lnm = true
	printText(vt, "123456")

	vt.lf()
	printText(vt, "X")

	if got, want := trimScreenString(vt.String()), "123456\nX"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestParsedLinefeedModePerformsCarriageReturn(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	parseAndApply(t, vt, "\x1b[20h123456\nX")

	if got, want := trimScreenString(vt.String()), "123456\nX"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestParsedLinefeedModeReset(t *testing.T) {
	vt := New()
	vt.resize(10, 3)

	parseAndApply(t, vt, "\x1b[20h123456\x1b[20l\nX")

	if got, want := trimScreenString(vt.String()), "123456\n      X"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestLinefeedModeUsesCarriageReturnSemantics(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.mode.lnm = true
	vt.margin.left = 2
	vt.cursor.col = 1

	vt.lf()

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnOriginModeMovesToLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.mode.decom = true
	vt.margin.left = 2
	vt.cursor.col = 0

	vt.cr()

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnLeftOfMarginMovesToZero(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.margin.left = 2
	vt.cursor.col = 1

	vt.cr()

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnRightOfMarginMovesToLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.margin.left = 2
	vt.cursor.col = 3

	vt.cr()

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnClearsPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "hello")

	vt.cr()

	if vt.lastCol {
		t.Fatal("carriage return did not clear pending wrap")
	}
}

func TestCarriageReturnPreservesSoftWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abcd")

	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("source row was not marked wrapped before carriage return")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("destination row was not marked as wrap continuation before carriage return")
	}

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.cr()

	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("carriage return cleared source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("carriage return cleared destination row wrap continuation")
	}
}

func TestEnquiryWithoutResponseDoesNotReply(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.update(ansi.C0(0x05))

	assertNoReply(t, r)
}

func TestEnquiryWritesConfiguredResponse(t *testing.T) {
	vt, r := newReplyTestModel(t, WithEnquiryResponse("vaxis-answerback"))

	vt.update(ansi.C0(0x05))

	if got, want := readReply(t, r, len("vaxis-answerback")), "vaxis-answerback"; got != want {
		t.Fatalf("enquiry response = %q, want %q", got, want)
	}
}
