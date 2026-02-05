package portmanager

import (
	"net"
	"testing"
	"time"

	testutil "github.com/jongio/azd-app/cli/src/internal/testing/testutil"
)

func TestPortReservation_Release(t *testing.T) {
	// Create a listener to simulate a reservation. Bind to localhost to avoid
	// triggering Windows Firewall prompts that occur when binding to all
	// interfaces (":0"). Using "127.0.0.1:0" keeps semantics of ephemeral
	// ports while restricting the bind to the loopback interface.
	listener, _, err := testutil.ListenLoopback(0)
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	reservation := &PortReservation{
		Port:     port,
		listener: listener,
		released: false,
	}

	// First release should succeed
	err = reservation.Release()
	if err != nil {
		t.Errorf("First Release() error = %v, want nil", err)
	}

	if !reservation.released {
		t.Error("reservation.released should be true after Release()")
	}

	// Second release should be safe (idempotent)
	err = reservation.Release()
	if err != nil {
		t.Errorf("Second Release() error = %v, want nil (should be idempotent)", err)
	}
}

func TestPortReservation_Release_NilListener(t *testing.T) {
	reservation := &PortReservation{
		Port:     8080,
		listener: nil,
		released: false,
	}

	err := reservation.Release()
	if err != nil {
		t.Errorf("Release() with nil listener error = %v, want nil", err)
	}
}

func TestPortReservation_Release_AlreadyReleased(t *testing.T) {
	listener, _, err := testutil.ListenLoopback(0)
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	_ = listener.Close() // Close it first

	reservation := &PortReservation{
		Port:     8080,
		listener: listener,
		released: true, // Already marked as released
	}

	err = reservation.Release()
	if err != nil {
		t.Errorf("Release() when already released error = %v, want nil", err)
	}
}

func TestPortAssignment_Fields(t *testing.T) {
	// Test that PortAssignment structure has expected fields
	assignment := PortAssignment{
		ServiceName: "test-service",
		Port:        8080,
		LastUsed:    time.Now(),
	}

	if assignment.ServiceName != "test-service" {
		t.Errorf("ServiceName = %s, want test-service", assignment.ServiceName)
	}
	if assignment.Port != 8080 {
		t.Errorf("Port = %d, want 8080", assignment.Port)
	}
	if assignment.LastUsed.IsZero() {
		t.Error("LastUsed should not be zero")
	}
}

func TestProcessInfo_Fields(t *testing.T) {
	// Test that ProcessInfo structure has expected fields
	info := ProcessInfo{
		PID:  1234,
		Name: "node",
	}

	if info.PID != 1234 {
		t.Errorf("PID = %d, want 1234", info.PID)
	}
	if info.Name != "node" {
		t.Errorf("Name = %s, want node", info.Name)
	}
}
