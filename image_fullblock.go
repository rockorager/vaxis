package vaxis

//
// import (
// 	"image"
// )
//
// var blitterCells = map[int]rune{
// 	0b1111: '█', // FULL BLOCK
// 	0b0000: ' ', // SPACE
// 	0b1100: '▀', // UPPER HALF BLOCK
// 	0b0011: '▄', // LOWER HALF BLOCK
// 	0b0101: '▐', // RIGHT HALF BLOCK
// 	0b1010: '▌', // LEFT HALF BLOCK
// 	0b0010: '▖', // QUADRANT LOWER LEFT
// 	0b0001: '▗', // QUADRANT LOWER RIGHT
// 	0b1000: '▘', // QUADRANT UPPER LEFT
// 	0b1011: '▙', // QUADRANT UPPER LEFT AND LOWER LEFT AND LOWER RIGHT
// 	0b1001: '▚', // QUADRANT UPPER LEFT AND LOWER RIGHT
// 	0b1110: '▛', // QUADRANT UPPER LEFT AND UPPER RIGHT AND LOWER LEFT
// 	0b1101: '▜', // QUADRANT UPPER LEFT AND UPPER RIGHT AND LOWER RIGHT
// 	0b0100: '▝', // QUADRANT UPPER RIGHT
// 	0b0110: '▞', // QUADRANT UPPER RIGHT AND LOWER LEFT
// 	0b0111: '▟', // QUADRANT UPPER RIGHT AND LOWER LEFT AND LOWER RIGHT
// }
//
// func blit(img image.Image) []Cell {
// 	w := img.Bounds().Max.X
// 	h := img.Bounds().Max.Y
// 	if h%2 != 0 {
// 		h += 1
// 	}
// 	// The image will be made into an array of cells, each cell will capture
// 	// 1x2 pixels
// 	cells := make([]Cell, (w * h / 2))
// 	for i, cell := range cells {
// 		y := i / w
// 		x := i - (y * w)
// 		y *= 2
//
// 		top := img.At(x, y)
// 		bottom := img.At(x, y+1)
// 		var (
// 			r uint8
// 			g uint8
// 			b uint8
// 		)
// 		switch top {
// 		case bottom:
// 			cell.Character = Character{
// 				Grapheme: " ",
// 				Width:    1,
// 			}
// 			pr, pg, pb, a := top.RGBA()
// 			if a > 0 {
// 				r = uint8((pr * 255) / a)
// 				g = uint8((pg * 255) / a)
// 				b = uint8((pb * 255) / a)
// 				cell.Background = RGBColor(r, g, b)
// 			}
// 		default:
// 			cell.Character = Character{
// 				Grapheme: "▀",
// 			}
// 			pr, pg, pb, a := top.RGBA()
// 			switch a {
// 			case 0:
// 				r = uint8(pr)
// 				g = uint8(pg)
// 				b = uint8(pb)
// 			default:
// 				r = uint8((pr * 255) / a)
// 				g = uint8((pg * 255) / a)
// 				b = uint8((pb * 255) / a)
// 			}
// 			cell.Foreground = RGBColor(r, g, b)
// 			pr, pg, pb, a = bottom.RGBA()
// 			if a > 0 {
// 				r = uint8((pr * 255) / a)
// 				g = uint8((pg * 255) / a)
// 				b = uint8((pb * 255) / a)
// 				cell.Background = RGBColor(r, g, b)
// 			}
// 		}
// 		cells[i] = cell
// 	}
// 	return cells
// }
