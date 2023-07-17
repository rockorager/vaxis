package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleCell() {
	c := vaxis.Cell{
		Character: "a",
	}
	win := vaxis.Window{}

	// Fills the entire window with "a"s
	vaxis.Fill(win, c)
}
