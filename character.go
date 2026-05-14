package vaxis

import "github.com/rockorager/go-uucode"

// Character is a single extended-grapheme-cluster. It also contains the width
// of the EGC
type Character struct {
	Grapheme string
	Width    int
}

// Converts a string into a slice of Characters suitable to assign to terminal cells
func Characters(s string) []Character {
	egcs := make([]Character, 0, len(s))
	it := uucode.NewGraphemeIterator(s)
	for g, ok := it.Next(); ok; g, ok = it.Next() {
		cluster := s[g.Start:g.End]
		if cluster == "\t" {
			for i := 0; i < 8; i += 1 {
				egcs = append(egcs, Character{" ", 1})
			}
			continue
		}
		egcs = append(egcs, Character{cluster, uucode.StringWidth(cluster)})
	}
	return egcs
}
