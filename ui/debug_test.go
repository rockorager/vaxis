package ui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDebugSnapshotDumpsTreeAndFocusTargets(t *testing.T) {
	app := NewApp(Column(
		Button{Label: "Run"},
		RichText{Spans: []TextSpan{
			{Text: "see "},
			{Text: "docs", OnPressed: func(EventContext) {}},
		}},
	))
	app.Pump(Size{Width: 40, Height: 3})
	snapshot := app.DebugSnapshot()
	if snapshot.Size != (DebugSize{Width: 40, Height: 3}) {
		t.Fatalf("size = %#v, want 40x3", snapshot.Size)
	}
	if snapshot.Tree == nil {
		t.Fatal("expected tree")
	}
	if snapshot.Focused == "" {
		t.Fatal("expected focused target")
	}
	if !debugSnapshotHasWidget(snapshot.Tree, "ui.RichText") {
		t.Fatalf("snapshot missing RichText node: %#v", snapshot.Tree)
	}
	if !debugSnapshotHasFocusLabel(snapshot.Focusables, "Run") {
		t.Fatalf("focusables = %#v, want Run label", snapshot.Focusables)
	}
	if !debugSnapshotHasFocusLabel(snapshot.Focusables, "docs") {
		t.Fatalf("focusables = %#v, want docs label", snapshot.Focusables)
	}
}

func TestDebugSnapshotUpdatesFocusedTarget(t *testing.T) {
	app := NewApp(RichText{Spans: []TextSpan{
		{Text: "one", OnPressed: func(EventContext) {}},
		{Text: " "},
		{Text: "two", OnPressed: func(EventContext) {}},
	}})
	app.Pump(Size{Width: 20, Height: 1})
	first := app.DebugSnapshot().Focused
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(Size{Width: 20, Height: 1})
	second := app.DebugSnapshot().Focused
	if first == "" || second == "" || first == second {
		t.Fatalf("focused targets = %q then %q, want distinct non-empty ids", first, second)
	}
}

func TestDebugSnapshotViaDispatch(t *testing.T) {
	app := NewApp(Text{Value: "debug"})
	app.Pump(Size{Width: 10, Height: 1})
	snapshot, ok := debugSnapshotViaDispatch(context.Background(), app, func(fn func()) { fn() })
	if !ok {
		t.Fatal("expected snapshot")
	}
	if snapshot.Tree == nil || !debugSnapshotHasWidget(snapshot.Tree, "ui.Text") {
		t.Fatalf("snapshot tree = %#v, want Text node", snapshot.Tree)
	}
}

func TestDebugServerRequiresToken(t *testing.T) {
	app := NewApp(Text{Value: "debug"})
	app.Pump(Size{Width: 10, Height: 1})
	handler := newDebugServerHandler(app, "secret", func(fn func()) { fn() }, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/debug/ui", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want unauthorized", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/ui", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
}

func TestDebugServerSubmitsEvents(t *testing.T) {
	pressed := false
	app := NewApp(Button{Label: "go", OnPressed: func(EventContext) { pressed = true }})
	app.Pump(Size{Width: 10, Height: 1})
	handler := newDebugServerHandler(app, "secret", func(fn func()) { fn() }, func(ev Event) {
		app.Send(ev)
	}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/debug/ui/events", bytes.NewBufferString(`{"type":"key","key":"Enter"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
	if !pressed {
		t.Fatal("expected submitted key event to press button")
	}
}

func TestDebugServerPerformsActions(t *testing.T) {
	var first, second FocusNode
	clicked := false
	app := NewApp(Row(
		Focus(&first, Text{Value: "first"}),
		Button{Label: "go", OnPressed: func(EventContext) { clicked = true }},
		Focus(&second, Text{Value: "second"}),
	))
	app.Pump(Size{Width: 30, Height: 1})
	handler := newDebugServerHandler(app, "secret", func(fn func()) { fn() }, func(ev Event) {
		app.Send(ev)
	}, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/debug/ui/actions", bytes.NewBufferString(`{"action":"focus","label":"second"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
	if !second.HasFocus() {
		t.Fatal("expected action to focus second target")
	}

	req = httptest.NewRequest(http.MethodPost, "/debug/ui/actions", bytes.NewBufferString(`{"action":"click","label":"go"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
	if !clicked {
		t.Fatal("expected action to click button")
	}
}

func TestDebugServerRenderedEndpoints(t *testing.T) {
	app := NewApp(Text{Value: "unused"})
	painter := NewPainter(Size{Width: 5, Height: 2})
	painter.DrawText(Offset{}, "hi", Style{Foreground: RGB(1, 2, 3)})
	handler := newDebugServerHandler(app, "secret", func(fn func()) { fn() }, nil, func() (DebugRenderedSnapshot, bool) {
		return debugRenderedSnapshot(painter), true
	}, func() (string, bool) {
		return debugRenderedText(painter), true
	})

	req := httptest.NewRequest(http.MethodGet, "/debug/ui/rendered", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"grapheme": "h"`) {
		t.Fatalf("rendered response = %s, want painted cells", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/ui/rendered.txt", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want ok: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.HasPrefix(got, "hi\n") {
		t.Fatalf("rendered text = %q, want hi prefix", got)
	}
}

func TestDebugRenderedTextPreservesBlankCells(t *testing.T) {
	painter := NewPainter(Size{Width: 8, Height: 2})
	painter.DrawText(Offset{X: 4, Y: 0}, "hi", Style{})
	painter.DrawText(Offset{X: 2, Y: 1}, "ok", Style{})

	if got, want := debugRenderedText(painter), "    hi\n  ok"; got != want {
		t.Fatalf("rendered text = %q, want %q", got, want)
	}
}

func debugSnapshotHasWidget(node *DebugNode, widget string) bool {
	if node == nil {
		return false
	}
	if node.Widget == widget {
		return true
	}
	for i := range node.Children {
		if debugSnapshotHasWidget(&node.Children[i], widget) {
			return true
		}
	}
	return false
}

func debugSnapshotHasFocusLabel(targets []DebugFocusTarget, label string) bool {
	for _, target := range targets {
		if target.Label == label || strings.Contains(target.Label, label) {
			return true
		}
	}
	return false
}
