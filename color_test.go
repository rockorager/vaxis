package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

func ExampleRGBColor() {
	vx, _ := vaxis.New(vaxis.Options{})
	color := vaxis.RGBColor(1, 2, 3)
	vx.Window().Fill(vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "a",
		},
		Style: vaxis.Style{
			Background: color,
		},
	})
}

func ExampleIndexColor() {
	vx, _ := vaxis.New(vaxis.Options{})
	// Index 1 is usually a red
	color := vaxis.IndexColor(1)
	vx.Window().Fill(vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: " ",
			Width:    1,
		},
		Style: vaxis.Style{
			Background: color,
		},
	})
}

func ExampleHexColor() {
	vx, _ := vaxis.New(vaxis.Options{})
	// Creates an RGB color from a hex value
	color := vaxis.HexColor(0x00AABB)
	vx.Window().Fill(vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: " ",
			Width:    1,
		},
		Style: vaxis.Style{
			Background: color,
		},
	})
}
