package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleRGBColor() {
	color := vaxis.RGBColor(1, 2, 3)
	vaxis.Fill(vaxis.Window{}, vaxis.Cell{
		Character:  " ",
		Background: color,
	})
}

func ExampleIndexColor() {
	// Index 1 is usually a red
	color := vaxis.IndexColor(1)
	vaxis.Fill(vaxis.Window{}, vaxis.Cell{
		Character:  " ",
		Background: color,
	})
}

func ExampleHexColor() {
	// Creates an RGB color from a hex value
	color := vaxis.HexColor(0x00AABB)
	vaxis.Fill(vaxis.Window{}, vaxis.Cell{
		Character:  " ",
		Background: color,
	})
}
