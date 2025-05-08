// Package octreequant implements an image quantizer, for transforming bitmap
// images to palette images, before encoding them to SIXEL.
//
// This code was originally developed by delthas and taken from: https://github.com/delthas/octreequant
package octreequant

import (
	"image"
	imagecolor "image/color"
)

const maxDepth = 8

type color struct {
	r  int
	g  int
	b  int
	a0 bool // true if transparent
}

func (c color) color() imagecolor.RGBA {
	return imagecolor.RGBA{
		R: uint8(c.r),
		G: uint8(c.g),
		B: uint8(c.b),
		A: 255,
	}
}

func newColor(c imagecolor.Color) color {
	cr, cg, cb, ca := c.RGBA()
	var r, g, b int
	if ca == 0 {
		return color{
			a0: true,
		}
	}
	if ca == 0xFFFF {
		r = int(cr) >> 8
		g = int(cg) >> 8
		b = int(cb) >> 8
	} else {
		r = int(cr) * 255 / int(ca)
		g = int(cg) * 255 / int(ca)
		b = int(cb) * 255 / int(ca)
	}
	return color{
		r: r,
		g: g,
		b: b,
	}
}

type node struct {
	c        color
	n        int
	i        int
	children []*node
}

func (n *node) leaf() bool {
	return n.n > 0
}

func (n *node) leafs() []*node {
	nodes := make([]*node, 0, 8)
	for _, n := range n.children {
		if n == nil {
			continue
		}
		if n.leaf() {
			nodes = append(nodes, n)
		} else {
			nodes = append(nodes, n.leafs()...)
		}
	}
	return nodes
}

func (n *node) addColor(color color, level int, parent *tree) {
	if level >= maxDepth {
		n.c.r += color.r
		n.c.g += color.g
		n.c.b += color.b
		n.n++
		return
	}
	i := n.colorIndex(color, level)
	c := n.children[i]
	if c == nil {
		c = newNode(level, parent)
		n.children[i] = c
	}
	c.addColor(color, level+1, parent)
}

func (n *node) paletteIndex(color color, level int) int {
	if n.leaf() {
		return n.i
	}
	i := n.colorIndex(color, level)
	if c := n.children[i]; c != nil {
		return c.paletteIndex(color, level+1)
	}
	for _, n := range n.children {
		if n == nil {
			continue
		}
		return n.paletteIndex(color, level+1)
	}
	panic("unreachable")
}

func (n *node) removeLeaves() int {
	r := 0
	for _, c := range n.children {
		if c == nil {
			continue
		}
		n.c.r += c.c.r
		n.c.g += c.c.g
		n.c.b += c.c.b
		n.n += c.n
		r += 1
	}
	return r - 1
}

func (n *node) colorIndex(color color, level int) int {
	i := 0
	mask := 0x80 >> level
	if color.r&mask != 0 {
		i |= 4
	}
	if color.g&mask != 0 {
		i |= 2
	}
	if color.b&mask != 0 {
		i |= 1
	}
	return i
}

func (n *node) color() color {
	return color{
		r: n.c.r / n.n,
		g: n.c.g / n.n,
		b: n.c.b / n.n,
	}
}

func newNode(level int, parent *tree) *node {
	n := node{
		children: make([]*node, 8),
	}
	if level < maxDepth-1 {
		parent.addNode(level, &n)
	}
	return &n
}

type tree struct {
	levels [][]*node
	root   *node

	count int  // size of palette
	a0    bool // true if any color is transparent
}

func (t *tree) leaves() []*node {
	return t.root.leafs()
}

func (t *tree) addNode(level int, n *node) {
	t.levels[level] = append(t.levels[level], n)
}

func (t *tree) addColor(color color) {
	t.a0 = t.a0 || color.a0
	t.root.addColor(color, 0, t)
}

func (t *tree) makePalette(count int) imagecolor.Palette {
	palette := make(imagecolor.Palette, 0, count)
	i := 0
	c := len(t.leaves())

	if t.a0 {
		count--
	}

	for level := maxDepth - 1; level >= 0; level-- {
		if len(t.levels[level]) == 0 {
			continue
		}
		for _, n := range t.levels[level] {
			c -= n.removeLeaves()
			if c <= count {
				break
			}
		}
		if c <= count {
			break
		}
		t.levels[level] = t.levels[level][:0]
	}

	for _, n := range t.leaves() {
		if i >= count {
			break
		}
		if n.leaf() {
			palette = append(palette, n.color().color())
		}
		n.i = i
		i++
	}

	if t.a0 {
		palette = append(palette, imagecolor.RGBA{})
	}
	t.count = len(palette)
	return palette
}

func (t *tree) paletteIndex(color color) int {
	if color.a0 {
		return t.count - 1
	}
	return t.root.paletteIndex(color, 0)
}

func newTree() *tree {
	t := tree{
		levels: make([][]*node, maxDepth),
	}
	t.root = newNode(0, &t)
	return &t
}

// Paletted quantizes an image and returns a paletted image, with
// a palette up to the specified color count.
func Paletted(img image.Image, colors int) *image.Paletted {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	t := newTree()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t.addColor(newColor(img.At(x, y)))
		}
	}
	out := image.NewPaletted(img.Bounds(), t.makePalette(colors))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetColorIndex(x, y, uint8(t.paletteIndex(newColor(img.At(x, y)))))
		}
	}
	return out
}
