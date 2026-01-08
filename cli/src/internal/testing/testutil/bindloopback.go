package testutil

import (
	"fmt"
	"net"
)

// ListenLoopback creates a TCP listener bound to the loopback interface on an
// ephemeral port (if port==0) or the specified port. Returns the listener and
// the chosen port. Caller should Close() the listener when done.
func ListenLoopback(port int) (net.Listener, int, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	// Extract actual port (in case 0 was passed)
	actual := listener.Addr().(*net.TCPAddr).Port
	return listener, actual, nil
}
