package term

import "testing"

func TestPrimaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil))

	if got, want := readReply(t, r, len("\x1B[?62;4;22c")), "\x1B[?62;4;22c"; got != want {
		t.Fatalf("primary device attributes = %q, want %q", got, want)
	}
}

func TestSecondaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil, '>'))

	if got, want := readReply(t, r, len("\x1B[>1;0;0c")), "\x1B[>1;0;0c"; got != want {
		t.Fatalf("secondary device attributes = %q, want %q", got, want)
	}
}

func TestTertiaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil, '='))

	if got, want := readReply(t, r, len("\x1BP!|00000000\x1B\\")), "\x1BP!|00000000\x1B\\"; got != want {
		t.Fatalf("tertiary device attributes = %q, want %q", got, want)
	}
}
