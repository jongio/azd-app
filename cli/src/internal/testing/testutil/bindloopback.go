package testutil

import (
	"context"
	"fmt"
	"net"
)

// ListenLoopback creates a TCP listener bound to the loopback interface on an
// ephemeral port (if port==0) or the specified port. Returns the listener and
// the chosen port. Caller should Close() the listener when done.
func ListenLoopback(port int) (net.Listener, int, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	// Extract actual port (in case 0 was passed)
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, 0, fmt.Errorf("listener addr has unexpected type %T", listener.Addr())
	}

	return listener, tcpAddr.Port, nil
}
