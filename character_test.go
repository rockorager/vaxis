package vaxis

import "testing"

func TestCharacterIterator(t *testing.T) {
	it := NewCharacterIterator("a\t🇺🇸")
	var got []Character
	for char, ok := it.Next(); ok; char, ok = it.Next() {
		got = append(got, char)
	}

	want := []Character{
		{Grapheme: "a", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: " ", Width: 1},
		{Grapheme: "🇺🇸", Width: 2},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d characters, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("character %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}
