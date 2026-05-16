package term

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestXTWINOPSTextAreaSizeCharacters(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('t', []uint32{18}))

	if got, want := readReply(t, r, len("\x1B[8;24;80t")), "\x1B[8;24;80t"; got != want {
		t.Fatalf("text area size report = %q, want %q", got, want)
	}
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
