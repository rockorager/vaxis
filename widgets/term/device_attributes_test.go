package term

import "testing"

func TestPrimaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil))

	if got, want := readReply(t, r, len(primaryDeviceAttributesReply)), primaryDeviceAttributesReply; got != want {
		t.Fatalf("primary device attributes = %q, want %q", got, want)
	}
}

func TestPrimaryDeviceAttributesPreservesSixelCapability(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil))

	if got, want := readReply(t, r, len(primaryDeviceAttributesReply)), "\x1B[?62;4;22c"; got != want {
		t.Fatalf("primary device attributes = %q, want %q", got, want)
	}
}

func TestDECIDPrimaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testESC('Z'))

	if got, want := readReply(t, r, len(primaryDeviceAttributesReply)), primaryDeviceAttributesReply; got != want {
		t.Fatalf("DECID primary device attributes = %q, want %q", got, want)
	}
}

func TestSecondaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil, '>'))

	if got, want := readReply(t, r, len(secondaryDeviceAttributesReply)), "\x1B[>1;10;0c"; got != want {
		t.Fatalf("secondary device attributes = %q, want %q", got, want)
	}
}

func TestTertiaryDeviceAttributes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('c', nil, '='))

	if got, want := readReply(t, r, len(tertiaryDeviceAttributesReply)), tertiaryDeviceAttributesReply; got != want {
		t.Fatalf("tertiary device attributes = %q, want %q", got, want)
	}
}

func TestDeviceAttributesAllowExplicitZeroParam(t *testing.T) {
	tests := []struct {
		name         string
		intermediate []rune
		want         string
	}{
		{name: "primary", want: primaryDeviceAttributesReply},
		{name: "secondary", intermediate: []rune{'>'}, want: secondaryDeviceAttributesReply},
		{name: "tertiary", intermediate: []rune{'='}, want: tertiaryDeviceAttributesReply},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)

			vt.update(testCSI('c', []uint32{0}, tt.intermediate...))

			if got := readReply(t, r, len(tt.want)); got != tt.want {
				t.Fatalf("device attributes = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeviceAttributesIgnoreInvalidIntermediate(t *testing.T) {
	tests := []struct {
		name         string
		intermediate []rune
	}{
		{name: "unknown", intermediate: []rune{'?'}},
		{name: "multiple", intermediate: []rune{'>', '='}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)

			vt.update(testCSI('c', nil, tt.intermediate...))

			assertNoReply(t, r)
		})
	}
}

func TestXTVERSION(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('q', []uint32{0}, '>'))

	if got, want := readReply(t, r, len("\x1BP>|vaxis\x1B\\")), "\x1BP>|vaxis\x1B\\"; got != want {
		t.Fatalf("XTVERSION = %q, want %q", got, want)
	}
}
