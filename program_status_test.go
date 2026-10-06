package vaxis

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"go.rockorager.dev/vaxis/ansi"
)

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestSetProgramStatus(t *testing.T) {
	for _, test := range []struct {
		name   string
		status ProgramStatus
		want   string
	}{
		{
			name:   "minimal",
			status: ProgramStatus{State: ProgramIdle},
			want:   "state=idle",
		},
		{
			name: "blocked with everything",
			status: ProgramStatus{
				State:       ProgramBlocked,
				ID:          "build/test",
				Kind:        ProgramBlockedPermission,
				Progress:    40,
				HasProgress: true,
				App:         "terraform",
				Title:       "Plan",
				Msg:         "Apply 3 to add, 1 to change, 0 to destroy?",
			},
			want: "state=blocked:id=build/test:kind=permission:progress=40:app=terraform:title=" +
				b64("Plan") + ":msg=" + b64("Apply 3 to add, 1 to change, 0 to destroy?"),
		},
		{
			name:   "kind and progress dropped outside their states",
			status: ProgramStatus{State: ProgramDone, Kind: ProgramBlockedAuth, Progress: 50, HasProgress: true},
			want:   "state=done",
		},
		{
			name:   "progress clamped",
			status: ProgramStatus{State: ProgramWorking, Progress: 150, HasProgress: true},
			want:   "state=working:progress=100",
		},
		{
			name:   "zero progress",
			status: ProgramStatus{State: ProgramWorking, HasProgress: true},
			want:   "state=working:progress=0",
		},
		{
			name:   "controls stripped from text",
			status: ProgramStatus{State: ProgramError, Msg: "bad\x1b]\x07\u009cthing\n"},
			want:   "state=error:msg=" + b64("bad]thing"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			vx := newWriterTestVaxis(&out)
			if err := vx.SetProgramStatus(test.status); err != nil {
				t.Fatal(err)
			}
			if got, want := out.String(), "\x1b]7501;"+test.want+"\x1b\\"; got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}
}

func TestSetProgramStatusRejectsInvalid(t *testing.T) {
	for _, test := range []struct {
		name   string
		status ProgramStatus
	}{
		{"missing state", ProgramStatus{}},
		{"unknown state", ProgramStatus{State: "sleeping"}},
		{"clear state", ProgramStatus{State: "clear"}},
		{"bad id char", ProgramStatus{State: ProgramIdle, ID: "a:b"}},
		{"empty id segment", ProgramStatus{State: ProgramIdle, ID: "a//b"}},
		{"long id segment", ProgramStatus{State: ProgramIdle, ID: strings.Repeat("a", 33)}},
		{"deep id", ProgramStatus{State: ProgramIdle, ID: "a/b/c/d/e/f/g/h/i"}},
		{"long id", ProgramStatus{State: ProgramIdle, ID: strings.Repeat(strings.Repeat("a", 32)+"/", 4) + "a"}},
		{"bad app", ProgramStatus{State: ProgramIdle, App: "my app"}},
		{"bad kind", ProgramStatus{State: ProgramBlocked, Kind: "coffee"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			vx := newWriterTestVaxis(&out)
			if err := vx.SetProgramStatus(test.status); err == nil {
				t.Fatal("expected error")
			}
			if out.Len() != 0 {
				t.Fatalf("wrote %q for invalid status", out.String())
			}
		})
	}
}

func TestProgramStatusTextTruncatesOnRuneBoundary(t *testing.T) {
	// 2047 ASCII bytes followed by a 3 byte rune crosses the 2048 byte limit.
	s := strings.Repeat("a", 2047) + "€"
	got, err := base64.StdEncoding.DecodeString(programStatusText(s, programStatusMaxMsg))
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Repeat("a", 2047); string(got) != want {
		t.Fatalf("decoded length = %d, want %d", len(got), len(want))
	}

	// The largest legal report fits in the 4096 byte sequence limit.
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	err = vx.SetProgramStatus(ProgramStatus{
		State:       ProgramBlocked,
		ID:          strings.Repeat(strings.Repeat("a", 31)+"/", 3) + strings.Repeat("a", 32),
		Kind:        ProgramBlockedQuestion,
		Progress:    100,
		HasProgress: true,
		App:         strings.Repeat("a", 32),
		Title:       strings.Repeat("€", 100),
		Msg:         strings.Repeat("€", 1000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() > 4096 {
		t.Fatalf("report is %d bytes, want <= 4096", out.Len())
	}
}

func TestClearProgramStatus(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	if err := vx.ClearProgramStatus(""); err != nil {
		t.Fatal(err)
	}
	if err := vx.ClearProgramStatus("us-east"); err != nil {
		t.Fatal(err)
	}
	if err := vx.ClearProgramStatus("bad id"); err == nil {
		t.Fatal("expected error for invalid id")
	}
	want := "\x1b]7501;state=clear\x1b\\\x1b]7501;state=clear:id=us-east\x1b\\"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestProgramStatusQueryReplyPostsCapability(t *testing.T) {
	for _, reply := range []string{"7501;?", "7501;?:future=1"} {
		vx := &Vaxis{queue: make(chan Event, 1)}
		vx.handleSequence(ansi.OSC{Payload: []rune(reply)})
		select {
		case ev := <-vx.queue:
			if _, ok := ev.(capabilityProgramStatus); !ok {
				t.Fatalf("event = %T, want capabilityProgramStatus", ev)
			}
		default:
			t.Fatalf("reply %q did not post a capability", reply)
		}
	}
}
