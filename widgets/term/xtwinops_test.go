package term

import (
	"io"
	"os"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

func TestXTWINOPSTextAreaSizeCharacters(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('t', []uint32{18}))

	if got, want := readReply(t, r, len("\x1B[8;24;80t")), "\x1B[8;24;80t"; got != want {
		t.Fatalf("text area size report = %q, want %q", got, want)
	}
}

func TestXTWINOPSTextAreaSizePixels(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})

	vt.update(testCSI('t', []uint32{14}))

	if got, want := readReply(t, r, len("\x1B[4;432;720t")), "\x1B[4;432;720t"; got != want {
		t.Fatalf("text area pixel size report = %q, want %q", got, want)
	}
}

func TestXTWINOPSCellSizePixels(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})

	vt.update(testCSI('t', []uint32{16}))

	if got, want := readReply(t, r, len("\x1B[6;18;9t")), "\x1B[6;18;9t"; got != want {
		t.Fatalf("cell pixel size report = %q, want %q", got, want)
	}
}

func TestXTWINOPSPixelReportsRequirePixelSize(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('t', []uint32{14}))
	vt.update(testCSI('t', []uint32{16}))

	assertNoReply(t, r)
}

func TestXTWINOPSReportTitle(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.osc("2;My Title")
	vt.update(testCSI('t', []uint32{21}))

	if got, want := readReply(t, r, len("\x1B]lMy Title\x1B\\")), "\x1B]lMy Title\x1B\\"; got != want {
		t.Fatalf("title report = %q, want %q", got, want)
	}
}

func TestRISClearsTitle(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.osc("2;My Title")
	vt.update(testESC('c'))
	vt.update(testCSI('t', []uint32{21}))

	if got, want := readReply(t, r, len("\x1B]l\x1B\\")), "\x1B]l\x1B\\"; got != want {
		t.Fatalf("title report after RIS = %q, want %q", got, want)
	}
}

func TestXTWINOPSIgnoresUnknownReports(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('t', []uint32{19}))

	assertNoReply(t, r)
}

func assertNoReply(t *testing.T, r *os.File) {
	t.Helper()
	ch := make(chan struct{}, 1)
	go func() {
		var buf [1]byte
		_, _ = io.ReadFull(r, buf[:])
		ch <- struct{}{}
	}()
	select {
	case <-ch:
		t.Fatal("unexpected terminal reply")
	case <-time.After(30 * time.Millisecond):
	}
}
