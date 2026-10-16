package vaxis

import (
	"bytes"
	"testing"
)

func TestSetTitle(t *testing.T) {
	for _, test := range []struct {
		name string
		set  func(*Vaxis, string)
		want string
	}{
		{
			name: "window title",
			set:  (*Vaxis).SetTitle,
			want: "\x1b]2;app - ]2;Fix pager\x1b\\",
		},
		{
			name: "window title and icon name",
			set:  (*Vaxis).SetTitleAndIconName,
			want: "\x1b]0;app - ]2;Fix pager\x1b\\",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			vx := newWriterTestVaxis(&out)
			test.set(vx, "app - \x1b]2;Fix\a pager\u009c")
			if got := out.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}
