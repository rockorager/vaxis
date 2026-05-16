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

func TestXTWINOPSTitleStackOpsAreAcceptedNoops(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "push default target", params: []uint32{22, 0}},
		{name: "push window target", params: []uint32{22, 2}},
		{name: "push with index", params: []uint32{22, 0, 5}},
		{name: "pop default target", params: []uint32{23, 0}},
		{name: "pop window target", params: []uint32{23, 2}},
		{name: "pop with index", params: []uint32{23, 0, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)
			vt.title = "current"

			vt.update(testCSI('t', tt.params))

			if got, want := vt.title, "current"; got != want {
				t.Fatalf("title = %q, want %q", got, want)
			}
			assertNoReply(t, r)
		})
	}
}

func TestXTWINOPSTitleStackOpsIgnoreIconTarget(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.title = "current"

	vt.update(testCSI('t', []uint32{22, 1}))
	vt.update(testCSI('t', []uint32{23, 1}))

	if got, want := vt.title, "current"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	assertNoReply(t, r)
}

func TestXTWINOPSIgnoresUnknownReports(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('t', []uint32{19}))

	assertNoReply(t, r)
}

func TestXTWINOPSReportsRejectExtraParameters(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "text area pixels", params: []uint32{14, 1}},
		{name: "cell pixels", params: []uint32{16, 1}},
		{name: "text area characters", params: []uint32{18, 1}},
		{name: "title", params: []uint32{21, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)
			vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})
			vt.title = "current"

			vt.update(testCSI('t', tt.params))

			assertNoReply(t, r)
		})
	}
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
