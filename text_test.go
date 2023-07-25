package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleText() {
	c := vaxis.Text{
		Content:    "a",
		Foreground: vaxis.IndexColor(1),
		Attribute:  vaxis.AttrBold | vaxis.AttrBlink,
	}
	win := vaxis.Window{}

	// Fills the entire window with blinking, bold, red "a"s
	vaxis.Fill(win, c)
}
