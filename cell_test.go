package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleCell() {
	c := vaxis.Cell{
		Character:  "a",
		Foreground: vaxis.IndexColor(1),
		Attribute:  vaxis.AttrBold | vaxis.AttrBlink,
	}
	win := vaxis.Window{}

	// Fills the entire window with blinking, bold, red "a"s
	vaxis.Fill(win, c)
}
