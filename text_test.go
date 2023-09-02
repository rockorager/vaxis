package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleText() {
	vx, _ := vaxis.New(vaxis.Options{})
	c := vaxis.Text{
		Content:    "a",
		Foreground: vaxis.IndexColor(1),
		Attribute:  vaxis.AttrBold | vaxis.AttrBlink,
	}

	// Fills the entire window with blinking, bold, red "a"s
	vx.Window().Fill(c)
}
