package ui

import "go.rockorager.dev/vaxis"

func vaxisCharacters(s string) []Character {
	return vaxis.Characters(s)
}

func vaxisEncodeCells(cells []Cell) string {
	return vaxis.EncodeCells(cells)
}
