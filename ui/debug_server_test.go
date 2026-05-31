package ui

import (
	"net"
	"testing"
)

func TestStartDebugServerIgnoresAddrInUse(t *testing.T) {
	t.Setenv("VAXIS_UI_DEBUG", "test-token")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	t.Setenv("VAXIS_UI_DEBUG_ADDR", ln.Addr().String())

	stop, err := startDebugServer(NewApp(Text{Value: "demo"}), func(fn func()) { fn() }, nil, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("startDebugServer returned error for occupied addr: %v", err)
	}
	stop()
}
