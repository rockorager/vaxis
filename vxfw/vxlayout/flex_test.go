package vxlayout

import (
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/vxfw"
	"go.rockorager.dev/vaxis/vxfw/text"
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

func TestFlexItemJustify(t *testing.T) {
	testCases := []struct {
		name             string
		children         []*FlexItem
		direction        FlexDirection
		max              vxfw.Size
		wantSurfaceSize  uint16
		wantChildSizes   []uint16
		wantChildOrigins []int
	}{
		{
			// "hello" (width 5) centered within 20-wide slot: (20-5)/2 = 7
			name: "center",
			children: []*FlexItem{
				{Widget: text.New("hello"), Flex: 1, Tight: true, Justify: AlignCenter},
			},
			direction:        FlexHorizontal,
			max:              vxfw.Size{Width: 20, Height: 16},
			wantSurfaceSize:  20,
			wantChildSizes:   []uint16{5},
			wantChildOrigins: []int{7},
		},
		{
			// "hello" (width 5) right-aligned within 20-wide slot: 20-5 = 15
			name: "end",
			children: []*FlexItem{
				{Widget: text.New("hello"), Flex: 1, Tight: true, Justify: AlignEnd},
			},
			direction:        FlexHorizontal,
			max:              vxfw.Size{Width: 20, Height: 16},
			wantSurfaceSize:  20,
			wantChildSizes:   []uint16{5},
			wantChildOrigins: []int{15},
		},
		{
			// AlignStart (default) forces the widget to fill its allocated slot
			name: "default-fills",
			children: []*FlexItem{
				{Widget: text.New("hello"), Flex: 1, Tight: true},
			},
			direction:        FlexHorizontal,
			max:              vxfw.Size{Width: 20, Height: 16},
			wantSurfaceSize:  20,
			wantChildSizes:   []uint16{20},
			wantChildOrigins: []int{0},
		},
		{
			// Non-Tight flex=1 item with AlignEnd: inherent width=5, flex=1,
			// remaining=15, child_size=5+15=20, but the widget renders at its
			// natural width (5) right-aligned within the 20-wide slot: 20-5=15
			name: "flex-item",
			children: []*FlexItem{
				{Widget: text.New("hello"), Flex: 1, Justify: AlignEnd},
			},
			direction:        FlexHorizontal,
			max:              vxfw.Size{Width: 20, Height: 16},
			wantSurfaceSize:  20,
			wantChildSizes:   []uint16{5},
			wantChildOrigins: []int{15},
		},
		{
			// each child gets 30/3 = 10 wide slot
			// AlignStart: fills slot, width=10, origin=0
			// AlignCenter: "b" is 1 wide in 10 slot: offset=(10-1)/2=4, origin=10+4=14
			// AlignEnd: "c" is 1 wide in 10 slot: offset=10-1=9, origin=20+9=29
			name: "multiple-items",
			children: []*FlexItem{
				{Widget: text.New("a"), Flex: 1, Tight: true, Justify: AlignStart},
				{Widget: text.New("b"), Flex: 1, Tight: true, Justify: AlignCenter},
				{Widget: text.New("c"), Flex: 1, Tight: true, Justify: AlignEnd},
			},
			direction:        FlexHorizontal,
			max:              vxfw.Size{Width: 30, Height: 16},
			wantSurfaceSize:  30,
			wantChildSizes:   []uint16{10, 1, 1},
			wantChildOrigins: []int{0, 14, 29},
		},
		{
			// "a" is 1 row high, slot is 20 rows high; centered: (20-1)/2 = 9
			name: "column",
			children: []*FlexItem{
				{Widget: text.New("a"), Flex: 1, Tight: true, Justify: AlignCenter},
			},
			direction:        FlexVertical,
			max:              vxfw.Size{Width: 16, Height: 20},
			wantSurfaceSize:  20,
			wantChildSizes:   []uint16{1},
			wantChildOrigins: []int{9},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			layout := FlexLayout{Children: tc.children, Direction: tc.direction}
			ctx := vxfw.DrawContext{Max: tc.max, Characters: vaxis.Characters}
			surface, err := layout.Draw(ctx)
			if err != nil {
				t.Fatal(err)
			}

			gotSurfaceSize := surface.Size.Width
			if tc.direction == FlexVertical {
				gotSurfaceSize = surface.Size.Height
			}
			if gotSurfaceSize != tc.wantSurfaceSize {
				t.Logf("wrong surface size, got=%d, want=%d", gotSurfaceSize, tc.wantSurfaceSize)
				t.Fail()
			}

			if len(surface.Children) != len(tc.wantChildSizes) || len(surface.Children) != len(tc.wantChildOrigins) {
				t.Fatalf("mismatched test case: %d children, %d wantChildSizes, %d wantChildOrigins",
					len(surface.Children), len(tc.wantChildSizes), len(tc.wantChildOrigins))
			}

			for i, child := range surface.Children {
				gotSize, gotOrigin := child.Surface.Size.Width, child.Origin.Col
				if tc.direction == FlexVertical {
					gotSize, gotOrigin = child.Surface.Size.Height, child.Origin.Row
				}
				if gotSize != tc.wantChildSizes[i] {
					t.Logf("child %d: wrong size, got=%d, want=%d", i, gotSize, tc.wantChildSizes[i])
					t.Fail()
				}
				if gotOrigin != tc.wantChildOrigins[i] {
					t.Logf("child %d: wrong origin, got=%d, want=%d", i, gotOrigin, tc.wantChildOrigins[i])
					t.Fail()
				}
			}
		})
	}
}

func TestFlexAnchor(t *testing.T) {
	tight := func(s string) *FlexItem {
		return &FlexItem{Widget: text.New(s), Flex: 2, Tight: true}
	}
	anchored := func(s string) *FlexItem {
		return &FlexItem{Widget: text.New(s), Flex: 0, Justify: AlignEnd}
	}

	testCases := []struct {
		name             string
		children         []*FlexItem
		width            uint16
		wantChildSizes   []uint16
		wantChildOrigins []int
	}{
		// anchor ("x") is flex=0, so it takes its inherent width of 1 and the second
		// tight item absorbs whatever's left; even width splits the two tight items
		// unevenly (30, 29), odd splits them evenly (30, 30)
		{
			name:             "last-width-even",
			children:         []*FlexItem{tight("ab"), tight("cd"), anchored("x")},
			width:            60,
			wantChildSizes:   []uint16{30, 29, 1},
			wantChildOrigins: []int{0, 30, 59},
		},
		{
			name:             "last-width-odd",
			children:         []*FlexItem{tight("ab"), tight("cd"), anchored("x")},
			width:            61,
			wantChildSizes:   []uint16{30, 30, 1},
			wantChildOrigins: []int{0, 30, 60},
		},
		// first tight item's width (40) stays the same regardless of the anchor's
		// width; the second tight item absorbs the difference instead
		{
			name: "first-stable-short",
			children: []*FlexItem{
				{Widget: text.New("first"), Flex: 2, Tight: true},
				{Widget: text.New("second"), Flex: 2, Tight: true},
				anchored("x"),
			},
			width:            80,
			wantChildSizes:   []uint16{40, 39, 1},
			wantChildOrigins: []int{0, 40, 79},
		},
		{
			name: "first-stable-long",
			children: []*FlexItem{
				{Widget: text.New("first"), Flex: 2, Tight: true},
				{Widget: text.New("second"), Flex: 2, Tight: true},
				anchored("a rather long anchor"), // 20 characters wide
			},
			width:            80,
			wantChildSizes:   []uint16{40, 20, 20},
			wantChildOrigins: []int{0, 40, 60},
		},
		// non-anchored flex=0 gap item (" ") sits between the tight item and the
		// anchor and must not overlap either
		{
			name: "with-gap",
			children: []*FlexItem{
				{Widget: text.New("ab"), Flex: 2, Tight: true},
				{Widget: text.New(" "), Flex: 0},
				anchored("x"),
			},
			width:            60,
			wantChildSizes:   []uint16{58, 1, 1},
			wantChildOrigins: []int{0, 58, 59},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			row := FlexLayout{Children: tc.children, Direction: FlexHorizontal}
			ctx := vxfw.DrawContext{Max: vxfw.Size{Width: tc.width, Height: 16}, Characters: vaxis.Characters}
			surface, err := row.Draw(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if surface.Size.Width != tc.width {
				t.Logf("wrong surface width, got=%d, want=%d", surface.Size.Width, tc.width)
				t.Fail()
			}

			if len(surface.Children) != len(tc.wantChildSizes) || len(surface.Children) != len(tc.wantChildOrigins) {
				t.Fatalf("mismatched test case: %d children, %d wantChildSizes, %d wantChildOrigins",
					len(surface.Children), len(tc.wantChildSizes), len(tc.wantChildOrigins))
			}

			for i, child := range surface.Children {
				if child.Surface.Size.Width != tc.wantChildSizes[i] {
					t.Logf("child %d: wrong width, got=%d, want=%d", i, child.Surface.Size.Width, tc.wantChildSizes[i])
					t.Fail()
				}
				if child.Origin.Col != tc.wantChildOrigins[i] {
					t.Logf("child %d: wrong origin, got=%d, want=%d", i, child.Origin.Col, tc.wantChildOrigins[i])
					t.Fail()
				}
			}
		})
	}
}

func TestFlexAlignInheritedMinimum(t *testing.T) {
	row := FlexLayout{
		Children: []*FlexItem{
			{Widget: text.New("hi"), Flex: 1, Tight: true, Justify: AlignCenter},
			{Widget: text.New("yo"), Flex: 1, Tight: true, Justify: AlignCenter},
		},
		Direction: FlexHorizontal,
	}

	ctx := vxfw.DrawContext{
		Min:        vxfw.Size{Width: 20, Height: 0},
		Max:        vxfw.Size{Width: 20, Height: 16},
		Characters: vaxis.Characters,
	}
	surface, err := row.Draw(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// aach child gets a 10-wide slot; each child's rendered width must stay within its slot
	// rather than being forced up to the inherited Min of 20.
	for i, ch := range surface.Children {
		if ch.Surface.Size.Width > 10 {
			t.Logf("child %d: width=%d leaked container Min, want<=10", i, ch.Surface.Size.Width)
			t.Fail()
		}
	}

	// No overlap between children
	prevEnd := 0
	for i, ch := range surface.Children {
		if ch.Origin.Col < prevEnd {
			t.Logf("child %d origin=%d overlaps previous child ending at %d", i, ch.Origin.Col, prevEnd)
			t.Fail()
		}
		prevEnd = ch.Origin.Col + int(ch.Surface.Size.Width)
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
