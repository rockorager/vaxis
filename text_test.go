package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleSegment() {
	vx, _ := vaxis.New(vaxis.Options{})
	c := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "a",
			Width:    1,
		},
		Style: vaxis.Style{
			Foreground: vaxis.IndexColor(1),
			Attribute:  vaxis.AttrBold | vaxis.AttrBlink,
		},
	}

	// Fills the entire window with blinking, bold, red "a"s
	vx.Window().Fill(c)
}
