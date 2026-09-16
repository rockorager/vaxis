package vaxis

import (
	"testing"
)

func TestStripControls(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "plain ascii",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "multi-byte utf8",
			input: "héllo 👩‍🚀",
			want:  "héllo 👩‍🚀",
		},
		{
			name:  "escape",
			input: "a\x1bb",
			want:  "ab",
		},
		{
			name:  "bell",
			input: "a\x07b",
			want:  "ab",
		},
		{
			name:  "string terminator",
			input: "a\u009cb",
			want:  "ab",
		},
		{
			name:  "delete",
			input: "a\x7fb",
			want:  "ab",
		},
		{
			name:  "tab and newline",
			input: "a\tb\nc",
			want:  "abc",
		},
		{
			name:  "nested osc",
			input: "hi\x1b]0;pwned\x07bye",
			want:  "hi]0;pwnedbye",
		},
		{
			name:  "invalid utf8 becomes replacement char",
			input: "a\x9cb",
			want:  "a\ufffdb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := stripControls(test.input); got != test.want {
				t.Fatalf("stripControls(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
