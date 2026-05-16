package vaxis

import "github.com/rockorager/go-uucode"

// Character is a single extended-grapheme-cluster. It also contains the width
// of the EGC
type Character struct {
	Grapheme string
	Width    int
}

// CharacterIterator streams extended-grapheme-cluster Characters from a string.
type CharacterIterator struct {
	input    string
	grapheme uucode.GraphemeIterator
	tabs     int
}

// NewCharacterIterator creates a streaming iterator over Characters in s.
func NewCharacterIterator(s string) CharacterIterator {
	return CharacterIterator{
		input:    s,
		grapheme: uucode.NewGraphemeIterator(s),
	}
}

// Next returns the next Character and true, or a zero Character and false when
// iteration is complete.
func (it *CharacterIterator) Next() (Character, bool) {
	if it.tabs > 0 {
		it.tabs -= 1
		return Character{" ", 1}, true
	}
	g, ok := it.grapheme.Next()
	if !ok {
		return Character{}, false
	}
	cluster := it.input[g.Start:g.End]
	if cluster == "\t" {
		it.tabs = 7
		return Character{" ", 1}, true
	}
	return Character{cluster, uucode.StringWidth(cluster)}, true
}

// Converts a string into a slice of Characters suitable to assign to terminal cells
func Characters(s string) []Character {
	egcs := make([]Character, 0, len(s))
	it := NewCharacterIterator(s)
	for char, ok := it.Next(); ok; char, ok = it.Next() {
		egcs = append(egcs, char)
	}
	return egcs
}
