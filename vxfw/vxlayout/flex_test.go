package vxlayout

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
)

func TestFlexRow(t *testing.T) {
	row := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("abc"), Flex: 0},
			{Widget: text.New("def"), Flex: 1},
			{Widget: text.New("ghi"), Flex: 1},
			{Widget: text.New("jkl\nmno"), Flex: 1},
		},
		Direction: FlexHorizontal,
	}

	ctx := vxfw.DrawContext{Max: vxfw.Size{Width: 16, Height: 16}, Characters: vaxis.Characters}
	surface, err := row.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if surface.Size.Width != 16 {
		t.Logf("wrong flex width, got=%d, want=16", surface.Size.Width)
		t.Fail()
	}

	if surface.Size.Height != 2 {
		t.Logf("wrong flex height, got=%d, want=2", surface.Size.Height)
		t.Fail()
	}

	if len(surface.Children) != 4 {
		t.Logf("wrong number of flex children, got=%d, want=4", len(surface.Children))
		t.Fail()
	}

	// col moves forward by the width of each child, used to assert origins
	col := 0

	// expected widths of each child
	// first should be 3, as that's the width it asked for and it has flex=0
	// the remaining all have flex=1 so the remaining space (16-12) is distributed with leftovers
	// going to the last child, giving: 4, 4, and 5
	widths := []uint16{
		3,
		3 + 1,
		3 + 1,
		3 + 1 + 1,
	}

	for i, want := range widths {
		child := surface.Children[i]

		got := child.Surface.Size.Width
		if got != want {
			t.Logf("wrong width for child %d, got=%d, want=%d", i, got, want)
			t.Fail()
		}
		if child.Origin.Col != col {
			t.Logf("wrong origin for child %d, got=%d, want=%d", i, child.Origin.Col, col)
			t.Fail()
		}
		col += int(want)
	}
}

func TestFlexRowTight(t *testing.T) {
	row := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("abc"), Flex: 0},
			{Widget: text.New("def"), Flex: 1, Tight: true},
			{Widget: text.New("ghi"), Flex: 1},
			{Widget: text.New("jkl\nmno"), Flex: 1, Tight: true},
		},
		Direction: FlexHorizontal,
	}

	ctx := vxfw.DrawContext{Max: vxfw.Size{Width: 16, Height: 16}, Characters: vaxis.Characters}
	surface, err := row.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if surface.Size.Width != 16 {
		t.Logf("wrong flex width, got=%d, want=16", surface.Size.Width)
		t.Fail()
	}

	if len(surface.Children) != 4 {
		t.Logf("wrong number of flex children, got=%d, want=4", len(surface.Children))
		t.Fail()
	}

	col := 0
	// expected widths of each child
	// "abc" has flex=0, so it keeps its inherent width of 3
	// inherent sizes are only measured for non-Tight items, so first_pass_size is
	// 3 ("abc") + 3 ("ghi") = 6, leaving remaining = 16-6 = 10 to distribute across
	// the three flex items (total_flex=3)
	// "def" is Tight, so its width is purely its share: 10*1/3 = 3
	// "ghi" is not Tight, so its width is its inherent size (3) plus its share: 3 + 10*1/3 = 6
	// "jkl\nmno" is both Tight and the last child; the last-child rule takes priority and it
	// gets whatever remains: 16 - (3+3+6) = 4
	widths := []uint16{
		3,
		3,
		6,
		4,
	}

	for i, want := range widths {
		child := surface.Children[i]
		if child.Surface.Size.Width != want {
			t.Logf("wrong width for child %d, got=%d, want=%d", i, child.Surface.Size.Width, want)
			t.Fail()
		}
		if child.Origin.Col != col {
			t.Logf("wrong origin for child %d, got=%d, want=%d", i, child.Origin.Col, col)
			t.Fail()
		}
		col += int(want)
	}
}

func TestFlexColumnTight(t *testing.T) {
	layout := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("abc"), Flex: 0},
			{Widget: text.New("def"), Flex: 1, Tight: true},
			{Widget: text.New("ghi"), Flex: 1},
			{Widget: text.New("jkl\nmno"), Flex: 1, Tight: true},
		},
		Direction: FlexVertical,
	}

	ctx := vxfw.DrawContext{Max: vxfw.Size{Width: 16, Height: 16}, Characters: vaxis.Characters}
	surface, err := layout.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if surface.Size.Height != 16 {
		t.Logf("wrong flex height, got=%d, want=16", surface.Size.Height)
		t.Fail()
	}

	if len(surface.Children) != 4 {
		t.Logf("wrong number of flex children, got=%d, want=4", len(surface.Children))
		t.Fail()
	}

	// row moves forward by the height of each child, used to assert origins
	row := 0

	// expected heights of each child
	// "abc" has flex=0, so it keeps its inherent height of 1
	// inherent sizes are only measured for non-Tight items, so first_pass_size is
	// 1 ("abc") + 1 ("ghi") = 2, leaving remaining = 16-2 = 14 to distribute across
	// the three flex items (total_flex=3)
	// "def" is Tight, so its height is purely its share: 14*1/3 = 4
	// "ghi" is not Tight, so its height is its inherent size (1) plus its share: 1 + 14*1/3 = 5
	// "jkl\nmno" is both Tight and the last child; the last-child rule takes priority and it
	// gets whatever remains: 16 - (1+4+5) = 6
	heights := []uint16{
		1,
		4,
		5,
		6,
	}

	for i, want := range heights {
		child := surface.Children[i]

		got := child.Surface.Size.Height
		if got != want {
			t.Logf("wrong height for child %d, got=%d, want=%d", i, got, want)
			t.Fail()
		}
		if child.Origin.Row != row {
			t.Logf("wrong origin for child %d, got=%d, want=%d", i, child.Origin.Row, row)
			t.Fail()
		}
		row += int(want)
	}
}

func TestFlexEqualTight(t *testing.T) {
	row := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("a"), Flex: 1, Tight: true},
			{Widget: text.New("bb"), Flex: 1, Tight: true},
			{Widget: text.New("ccc"), Flex: 1, Tight: true},
			{Widget: text.New("dddd"), Flex: 1, Tight: true},
		},
		Direction: FlexHorizontal,
	}

	ctx := vxfw.DrawContext{Max: vxfw.Size{Width: 20, Height: 16}, Characters: vaxis.Characters}
	surface, err := row.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if surface.Size.Width != 20 {
		t.Logf("wrong flex width, got=%d, want=20", surface.Size.Width)
		t.Fail()
	}

	// all four items have equal flex and are Tight, so despite their differing content
	// lengths ("a", "bb", "ccc", "dddd") each should get an equal 20/4=5 share
	for i, child := range surface.Children {
		if child.Surface.Size.Width != 5 {
			t.Logf("wrong width for child %d, got=%d, want=5", i, child.Surface.Size.Width)
			t.Fail()
		}
	}
}

func TestFlexColumn(t *testing.T) {
	layout := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("abc"), Flex: 0},
			{Widget: text.New("def"), Flex: 1},
			{Widget: text.New("ghi"), Flex: 1},
			{Widget: text.New("jkl\nmno"), Flex: 1},
		},
		Direction: FlexVertical,
	}

	ctx := vxfw.DrawContext{Max: vxfw.Size{Width: 16, Height: 16}, Characters: vaxis.Characters}
	surface, err := layout.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if surface.Size.Width != 3 {
		t.Logf("wrong flex width, got=%d, want=3", surface.Size.Width)
		t.Fail()
	}

	if surface.Size.Height != 16 {
		t.Logf("wrong flex height, got=%d, want=16", surface.Size.Height)
		t.Fail()
	}

	if len(surface.Children) != 4 {
		t.Logf("wrong number of flex children, got=%d, want=4", len(surface.Children))
		t.Fail()
	}

	// row moves forward by the height of each child, used to assert origins
	row := 0

	// expected heights of each child
	// inherent sizes are 1+1+1+2, leaving 16-(1+1+1+2)=11 rows for flex
	// first child gets 1 row due to flex=0
	// next three have flex=1 so they split the remaining 11 with 3 each, and the leftover 2 is
	// given to the last child
	heights := []uint16{
		1,
		1 + 3,
		1 + 3,
		2 + 3 + 2,
	}

	for i, want := range heights {
		child := surface.Children[i]

		got := child.Surface.Size.Height
		if got != want {
			t.Logf("wrong height for child %d, got=%d, want=%d", i, got, want)
			t.Fail()
		}
		if child.Origin.Row != row {
			t.Logf("wrong origin for child %d, got=%d, want=%d", i, child.Origin.Row, row)
			t.Fail()
		}
		row += int(want)
	}
}
