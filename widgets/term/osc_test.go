package term

import "testing"

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
