package vaxis

import (
	"testing"
)

func TestWrapSegments(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		width    int
		expected [][]Segment
	}{
		{
			name: "simple text fits on one line",
			segments: []Segment{
				{Text: "Hello world", Style: Style{}},
			},
			width: 20,
			expected: [][]Segment{
				{{Text: "Hello world", Style: Style{}}},
			},
		},
		{
			name: "text wraps at word boundary",
			segments: []Segment{
				{Text: "Hello world this is a test", Style: Style{}},
			},
			width: 15,
			// Should break at word boundaries, not mid-word.
			expected: [][]Segment{
				{{Text: "Hello world ", Style: Style{}}},
				{{Text: "this is a test", Style: Style{}}},
			},
		},
		{
			name: "CJK characters double width",
			segments: []Segment{
				// "Hello " = 6 chars, "世界" = 4 cells (2 chars × 2 width each).
				{Text: "Hello 世界", Style: Style{}},
			},
			width: 10,
			expected: [][]Segment{
				{{Text: "Hello 世界", Style: Style{}}},
			},
		},
		{
			name: "CJK characters wrap correctly",
			segments: []Segment{
				// Each character is 2 cells wide.
				{Text: "我是雨果", Style: Style{}},
			},
			width: 4,
			expected: [][]Segment{
				{{Text: "我是", Style: Style{}}},
				{{Text: "雨果", Style: Style{}}},
			},
		},
		{
			name: "emoji sequences handled correctly",
			segments: []Segment{
				// "Hello " = 6, "🏳️‍🌈" = 2 cells, " world" = 6
				{Text: "Hello 🏳️‍🌈 world", Style: Style{}},
			},
			width: 15,
			expected: [][]Segment{
				{{Text: "Hello 🏳️‍🌈 world", Style: Style{}}},
			},
			// TODO: test with this emoji at index == width - 1.
		},
		{
			name: "URL too long to fit - splits with unique ID",
			segments: []Segment{
				{
					Text: "https://example.com/very/long/path/that/does/not/fit",
					Style: Style{
						Hyperlink: "https://example.com/very/long/path/that/does/not/fit",
					},
				},
			},
			width: 20,
			expected: [][]Segment{
				{{
					Text: "https://example.com/",
					Style: Style{
						Hyperlink:       "https://example.com/very/long/path/that/does/not/fit",
						HyperlinkParams: "id=1",
					},
				}},
				{{
					Text: "very/long/path/that/",
					Style: Style{
						Hyperlink:       "https://example.com/very/long/path/that/does/not/fit",
						HyperlinkParams: "id=1",
					},
				}},
				{{
					Text: "does/not/fit",
					Style: Style{
						Hyperlink:       "https://example.com/very/long/path/that/does/not/fit",
						HyperlinkParams: "id=1",
					},
				}},
			},
		},
		{
			name: "multiple segments with different styles",
			segments: []Segment{
				{Text: "Normal ", Style: Style{}},
				{Text: "bold", Style: Style{Attribute: AttrBold}},
				{Text: " text", Style: Style{}},
			},
			width: 20,
			expected: [][]Segment{
				{
					{Text: "Normal ", Style: Style{}},
					{Text: "bold", Style: Style{Attribute: AttrBold}},
					{Text: " text", Style: Style{}},
				},
			},
		},
		{
			name: "URL with text wraps preserving segments",
			segments: []Segment{
				{Text: "Check out ", Style: Style{}},
				{
					Text:  "https://example.com/path",
					Style: Style{Hyperlink: "https://example.com/path"},
				},
				{Text: " for more", Style: Style{}},
			},
			width: 20,
			expected: [][]Segment{
				// FIXME: we shouldn't break URLs like this?
				{{Text: "Check out ", Style: Style{}}},
				{{
					Text: "https://example.com/",
					Style: Style{
						Hyperlink:       "https://example.com/path",
						HyperlinkParams: "id=1",
					},
				}},
				{
					{
						Text: "path",
						Style: Style{
							Hyperlink:       "https://example.com/path",
							HyperlinkParams: "id=1",
						},
					},
					{Text: " for more", Style: Style{}},
				},
			},
		},
		{
			name: "empty segments",
			segments: []Segment{
				{Text: "", Style: Style{}},
			},
			width: 10,
			expected: [][]Segment{
				{{Text: "", Style: Style{}}},
			},
		},
		{
			name: "zero width",
			segments: []Segment{
				{Text: "text", Style: Style{}},
			},
			width:    0,
			expected: [][]Segment{},
		},
		{
			name: "combining characters don't add width",
			segments: []Segment{
				// Hint: these are five bytes.
				{Text: "café", Style: Style{}},
			}, // e + ´ = é
			width: 4,
			expected: [][]Segment{
				{{Text: "café", Style: Style{}}},
			},
		},
		{
			// Breaking after a hyphen is technically correct 🤷
			name: "break at hyphen",
			segments: []Segment{
				{Text: "foo-bar-baz", Style: Style{}},
			},
			width: 8,
			expected: [][]Segment{
				{{Text: "foo-bar-", Style: Style{}}},
				{{Text: "baz", Style: Style{}}},
			},
		},
		{
			name: "multiple URLs each get unique ID when split",
			segments: []Segment{
				{
					Text:  "https://first.example.com/very/long/path",
					Style: Style{Hyperlink: "https://first.example.com/very/long/path"},
				},
				{Text: " and ", Style: Style{}},
				{
					Text:  "https://second.example.com/also/very/long",
					Style: Style{Hyperlink: "https://second.example.com/also/very/long"},
				},
			},
			width: 20,
			expected: [][]Segment{
				{{
					Text: "https://",
					Style: Style{
						Hyperlink:       "https://first.example.com/very/long/path",
						HyperlinkParams: "id=1",
					},
				}},
				{{
					Text: "first.example.com/",
					Style: Style{
						Hyperlink:       "https://first.example.com/very/long/path",
						HyperlinkParams: "id=1",
					},
				}},
				{
					{
						Text: "very/long/path",
						Style: Style{
							Hyperlink:       "https://first.example.com/very/long/path",
							HyperlinkParams: "id=1",
						},
					},
					{Text: " and ", Style: Style{}},
				},
				{{
					Text: "https://",
					Style: Style{
						Hyperlink:       "https://second.example.com/also/very/long",
						HyperlinkParams: "id=2",
					},
				}},
				{{
					Text: "second.example.com/",
					Style: Style{
						Hyperlink:       "https://second.example.com/also/very/long",
						HyperlinkParams: "id=2",
					},
				}},
				{{
					Text: "also/very/long",
					Style: Style{
						Hyperlink:       "https://second.example.com/also/very/long",
						HyperlinkParams: "id=2",
					},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := NewSegmentWrapper(tt.width)
			for _, seg := range tt.segments {
				if err := wrapper.AddSegment(seg); err != nil {
					t.Fatalf("AddSegment failed: %v", err)
				}
			}
			result := wrapper.Lines()

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d lines, got %d", len(tt.expected), len(result))
				for i, line := range result {
					t.Logf("  line %d: %d segments", i, len(line))
					for j, seg := range line {
						t.Logf("    seg %d: %q (link=%q, params=%q, attr=%v)",
							j, seg.Text, seg.Style.Hyperlink, seg.Style.HyperlinkParams, seg.Style.Attribute)
					}
				}
				return
			}

			for lineIdx, expected := range tt.expected {
				got := result[lineIdx]

				if len(got) != len(expected) {
					t.Errorf("line %d: expected %d segments, got %d",
						lineIdx, len(expected), len(got))
					for j, seg := range got {
						t.Logf("  actual seg %d: %q (link=%q, params=%q)",
							j, seg.Text, seg.Style.Hyperlink, seg.Style.HyperlinkParams)
					}
					for j, seg := range expected {
						t.Logf("  expected seg %d: %q (link=%q, params=%q)",
							j, seg.Text, seg.Style.Hyperlink, seg.Style.HyperlinkParams)
					}
					continue
				}

				for segIdx, expectedSeg := range expected {
					actualSeg := got[segIdx]

					if actualSeg.Text != expectedSeg.Text {
						t.Errorf("line %d, segment %d: expected text %q, got %q",
							lineIdx, segIdx, expectedSeg.Text, actualSeg.Text)
					}

					if actualSeg.Style.Hyperlink != expectedSeg.Style.Hyperlink {
						t.Errorf("line %d, segment %d: expected hyperlink %q, got %q",
							lineIdx, segIdx, expectedSeg.Style.Hyperlink, actualSeg.Style.Hyperlink)
					}

					if actualSeg.Style.HyperlinkParams != expectedSeg.Style.HyperlinkParams {
						t.Errorf("line %d, segment %d: expected params %q, got %q",
							lineIdx, segIdx, expectedSeg.Style.HyperlinkParams, actualSeg.Style.HyperlinkParams)
					}

					if actualSeg.Style.Attribute != expectedSeg.Style.Attribute {
						t.Errorf("line %d, segment %d: expected attribute %v, got %v",
							lineIdx, segIdx, expectedSeg.Style.Attribute, actualSeg.Style.Attribute)
					}
				}
			}
		})
	}
}

func TestSegmentWidth(t *testing.T) {
	tests := []struct {
		text          string
		expectedWidth int
	}{
		{
			text:          "Hello",
			expectedWidth: 5,
		},
		{
			text:          "世界", // Each chinese character is width=2
			expectedWidth: 4,
		},
		{
			text:          "Hello世界", // Each chinese character is width=2
			expectedWidth: 9,
		},
		{
			text:          "🏳️‍🌈",
			expectedWidth: 2,
		},
		{
			text:          "👨‍👩‍👧‍👦",
			expectedWidth: 2,
		},
		{
			text:          "é", // e + ´
			expectedWidth: 1,
		},
		{
			text:          "a\u200Bb", // a + zero-width space + b
			expectedWidth: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			seg := Segment{Text: tt.text}
			width := segmentWidth(seg)

			if width != tt.expectedWidth {
				t.Errorf("expected width %d, got %d for text %q",
					tt.expectedWidth, width, tt.text)
			}
		})
	}
}
