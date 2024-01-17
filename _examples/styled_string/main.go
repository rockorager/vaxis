package main

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
)

func main() {
	s := "\x1b[31;4:3;58:5:5mfoo"
	cells := vaxis.ParseStyledString(s)
	fmt.Println(vaxis.EncodeCells(cells))
}
