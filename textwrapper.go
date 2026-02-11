package vaxis

import (
	"fmt"
	"strings"

	"github.com/rivo/uniseg"
)

// segmentWidth calculates the actual display width of a segment.
func segmentWidth(seg Segment) int {
	return uniseg.StringWidth(seg.Text)
}

// SegmentWrapper wraps text segments to fit within a specified width,
// handling proper Unicode line breaking and hyperlink preservation.
type SegmentWrapper struct {
	width         int
	lines         [][]Segment
	currentLine   []Segment
	currentWidth  int
	urlSplits     map[string]int // URLs that have been split and their IDs.
	linkIDCounter int            // Counter for generating unique hyperlink IDs.
}

// NewSegmentWrapper creates a new segment wrapper with the specified width.
func NewSegmentWrapper(width int) *SegmentWrapper {
	return &SegmentWrapper{
		width:     width,
		urlSplits: make(map[string]int),
	}
}

// AddSegment adds a segment to the wrapper, handling wrapping as needed.
// Returns an error if the content cannot be rendered (e.g., terminal too narrow).
func (sw *SegmentWrapper) AddSegment(seg Segment) error {
	if sw.width <= 0 {
		return nil
	}

	segWidth := segmentWidth(seg)

	if sw.canFit(segWidth) {
		sw.addToCurrentLine(seg, segWidth)
		return nil
	}

	if len(sw.currentLine) > 0 {
		sw.finishCurrentLine()

		if sw.canFit(segWidth) {
			sw.addToCurrentLine(seg, segWidth)
			return nil
		}
	}

	return sw.splitAndAddSegment(seg)
}

// Lines returns the wrapped lines and finishes any pending line.
func (sw *SegmentWrapper) Lines() [][]Segment {
	if len(sw.currentLine) > 0 {
		sw.finishCurrentLine()
	}

	return sw.lines
}

// canFit checks if a segment of the given width can fit on the current line.
func (sw *SegmentWrapper) canFit(segWidth int) bool {
	return sw.currentWidth+segWidth <= sw.width
}

// addToCurrentLine adds a segment to the current line.
func (sw *SegmentWrapper) addToCurrentLine(seg Segment, segWidth int) {
	sw.currentLine = append(sw.currentLine, seg)
	sw.currentWidth += segWidth
}

// finishCurrentLine finishes the current line and starts a new one.
func (sw *SegmentWrapper) finishCurrentLine() {
	sw.lines = append(sw.lines, sw.currentLine)
	sw.currentLine = []Segment{}
	sw.currentWidth = 0
}

// splitAndAddSegment splits a segment that is too long to fit on one line,
// using uniseg for proper Unicode boundary breaking.
// Returns an error if content cannot be rendered.
func (sw *SegmentWrapper) splitAndAddSegment(seg Segment) error {
	remaining := seg.Text
	style := seg.Style
	state := -1

	for remaining != "" {
		availableWidth := sw.width - sw.currentWidth
		if availableWidth <= 0 {
			if len(sw.currentLine) > 0 {
				sw.finishCurrentLine()
			}
			availableWidth = sw.width
		}

		chunk, restAfterChunk := findBreakPoint(remaining, availableWidth, &state)

		if chunk != "" {
			// Found a good break point.
			chunkStyle := sw.assignURLID(style, chunk, len(seg.Text))
			sw.addToCurrentLine(Segment{
				Text:  chunk,
				Style: chunkStyle,
			}, uniseg.StringWidth(chunk))
			remaining = restAfterChunk
		} else {
			// Can't fit anything into current line.

			if len(sw.currentLine) > 0 {
				sw.finishCurrentLine()
				continue
			}
			// Else: Current line empty AND can't fit anything.

			forced, restAfterForced := forceBreakByGrapheme(remaining, availableWidth)
			if forced != "" {
				forcedStyle := sw.assignURLID(style, forced, len(seg.Text))
				sw.addToCurrentLine(Segment{
					Text:  forced,
					Style: forcedStyle,
				}, uniseg.StringWidth(forced))
				remaining = restAfterForced
				state = -1 // Reset state after forced break
			} else {
				// Should never really happen. Is the terminal literally 1 char wide!?
				// Better handle it than loop forever.
				return fmt.Errorf("cannot render text: grapheme width exceeds line width (%d cells)", sw.width)
			}
		}
	}

	return nil
}

// findBreakPoint finds a good break point within the available width using uniseg.
// Returns the chunk to use and the remaining text, or empty string if nothing fits.
func findBreakPoint(text string, availableWidth int, state *int) (chunk string, rest string) {
	var accumulated strings.Builder
	accumulatedWidth := 0
	var lastGoodBreak string

	tempRemaining := text
	tempState := *state

	for tempRemaining != "" {
		segment, nextRest, _, nextState := uniseg.FirstLineSegmentInString(tempRemaining, tempState)
		segmentWidth := uniseg.StringWidth(segment)

		if accumulatedWidth+segmentWidth <= availableWidth {
			// Segment fits.
			accumulated.WriteString(segment)
			accumulatedWidth += segmentWidth
			lastGoodBreak = accumulated.String()
			tempRemaining = nextRest
			tempState = nextState
		} else {
			// Segment doesn't fit.
			break
		}
	}

	if lastGoodBreak != "" {
		*state = tempState
		return lastGoodBreak, text[len(lastGoodBreak):]
	}

	return "", text
}

// forceBreakByGrapheme forces a break at grapheme boundaries when uniseg can't help.
// Returns the forced chunk and remaining text, or empty if first grapheme is too wide.
func forceBreakByGrapheme(text string, availableWidth int) (chunk string, rest string) {
	chars := Characters(text)
	if len(chars) == 0 {
		return "", text
	}

	forcedWidth := 0
	forcedBytes := 0
	for _, char := range chars {
		if forcedWidth+char.Width > availableWidth {
			break
		}
		forcedWidth += char.Width
		forcedBytes += len(char.Grapheme)
	}

	if forcedBytes > 0 {
		return text[:forcedBytes], text[forcedBytes:]
	}

	return "", text
}

// assignURLID assigns a unique ID to a URL segment if it's being split.
func (sw *SegmentWrapper) assignURLID(style Style, chunk string, originalLength int) Style {
	if style.Hyperlink == "" {
		return style
	}

	resultStyle := style

	if existingID, exists := sw.urlSplits[style.Hyperlink]; exists {
		// URL has already an ID.
		resultStyle.HyperlinkParams = fmt.Sprintf("id=%d", existingID)
	} else if len(chunk) < originalLength {
		// New URL: assign new ID.
		sw.linkIDCounter++
		sw.urlSplits[style.Hyperlink] = sw.linkIDCounter
		resultStyle.HyperlinkParams = fmt.Sprintf("id=%d", sw.linkIDCounter)
	}

	return resultStyle
}
