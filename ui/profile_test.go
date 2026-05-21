package ui

import (
	"strings"
	"testing"
	"time"
)

func TestProfileStoreKeepsRecentWindow(t *testing.T) {
	var profile profileStore
	for i := 1; i <= profileWindow+10; i++ {
		profile.record(profileKey, time.Duration(i)*time.Millisecond)
	}

	snapshot := profile.snapshot()
	if snapshot.Key.Count != profileWindow {
		t.Fatalf("count = %d, want %d", snapshot.Key.Count, profileWindow)
	}
	if snapshot.Key.LastMS != float64(profileWindow+10) {
		t.Fatalf("last = %v, want %v", snapshot.Key.LastMS, profileWindow+10)
	}
	if snapshot.Key.P95MS != 960 {
		t.Fatalf("p95 = %v, want 960", snapshot.Key.P95MS)
	}
	if snapshot.Key.P99MS != 1000 {
		t.Fatalf("p99 = %v, want 1000", snapshot.Key.P99MS)
	}
}

func TestProfileOverlayDrawsTopRightTable(t *testing.T) {
	p := NewPainter(Size{Width: 40, Height: 10})
	drawProfileOverlay(p, DebugProfileSnapshot{
		Window: profileWindow,
		Key:    DebugProfileSample{Count: 1, LastMS: 1.2, P95MS: 1.2, P99MS: 1.2},
		Frame:  DebugProfileSample{Count: 1, LastMS: 16.7, P95MS: 16.7, P99MS: 16.7},
	})

	var firstLine strings.Builder
	for x := 11; x < 38; x++ {
		firstLine.WriteString(p.Cell(x, 1).Grapheme)
	}
	if got := firstLine.String(); got != "metric   last    p95    p99" {
		t.Fatalf("header = %q, want overlay header", got)
	}
	if got := p.Cell(9, 0).Grapheme; got != "╭" {
		t.Fatalf("top-left corner = %q, want rounded border", got)
	}
	if got := p.Cell(11, 3).Grapheme + p.Cell(12, 3).Grapheme + p.Cell(13, 3).Grapheme; got != "key" {
		t.Fatalf("key row prefix = %q, want key", got)
	}
}
