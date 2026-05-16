package term

import (
	"strings"
	"testing"
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
