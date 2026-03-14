package testutil

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestListenLoopback verifies ListenLoopback returns a listener and a reachable port.
func TestListenLoopback(t *testing.T) {
	listener, port, err := ListenLoopback(0)
	if err != nil {
		t.Fatalf("ListenLoopback failed: %v", err)
	}
	defer func() { _ = listener.Close() }()

	if port == 0 {
		t.Fatalf("ListenLoopback returned port 0")
	}

	// Ensure the listener is bound to loopback interface
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCPAddr, got %T", listener.Addr())
	}
	if !tcpAddr.IP.IsLoopback() {
		t.Fatalf("listener is not bound to loopback: %s", tcpAddr.IP.String())
	}

	// Try to connect to the listener to ensure it's actually listening.
	deadline := time.Now().Add(500 * time.Millisecond)
	var conn net.Conn
	for time.Now().Before(deadline) {
		dialer := net.Dialer{}
		conn, err = dialer.DialContext(context.Background(), "tcp", listener.Addr().String())
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Could not connect to listener on %s: %v", listener.Addr().String(), err)
}
