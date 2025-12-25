package portmanager

import (
	"testing"
)

func TestPortConflictAction_Constants(t *testing.T) {
	// Verify the constants are defined
	actions := []PortConflictAction{
		ActionKill,
		ActionReassign,
		ActionCancel,
		ActionAlwaysKill,
	}

	// Just verify they have different values
	seen := make(map[PortConflictAction]bool)
	for _, action := range actions {
		if seen[action] {
			t.Errorf("Duplicate PortConflictAction value: %d", action)
		}
		seen[action] = true
	}

	if len(seen) != 4 {
		t.Errorf("Expected 4 unique PortConflictAction values, got %d", len(seen))
	}
}

func TestGetProcessInfoString(t *testing.T) {
	// This function calls getProcessInfoOnPort which is a method, not mockable easily
	// We'll test with a real PortManager instance
	pm := &PortManager{}

	// Test with a port that likely has no process
	result := getProcessInfoString(pm, 65500)

	// Result should be either empty or contain PID info
	// We can't predict the exact output, just verify it doesn't panic
	_ = result
}

func TestPrintFunctions(t *testing.T) {
	// These tests verify the print functions don't panic
	// They write to stderr so we can't easily capture output
	// but we can verify they execute without errors

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "printAutoKillMessage explicit",
			fn: func() {
				printAutoKillMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printAutoKillMessage non-explicit",
			fn: func() {
				printAutoKillMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
		{
			name: "printConflictMessage explicit",
			fn: func() {
				printConflictMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printConflictMessage non-explicit",
			fn: func() {
				printConflictMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
		{
			name: "printPortFreedMessage",
			fn: func() {
				printPortFreedMessage("test-service", 8080)
			},
		},
		{
			name: "printPortAssignedMessage",
			fn: func() {
				printPortAssignedMessage("test-service", 8080)
			},
		},
		{
			name: "printPortAvailableMessage",
			fn: func() {
				printPortAvailableMessage("test-service", 8080)
			},
		},
		{
			name: "printFindingPortMessage",
			fn: func() {
				printFindingPortMessage("test-service")
			},
		},
		{
			name: "printPreferenceSavedMessage",
			fn: func() {
				printPreferenceSavedMessage()
			},
		},
		{
			name: "printKillFailedTip",
			fn: func() {
				printKillFailedTip()
			},
		},
		{
			name: "printPortStillInUseMessage",
			fn: func() {
				printPortStillInUseMessage(8080)
			},
		},
		{
			name: "printAutoAssignedMessage",
			fn: func() {
				printAutoAssignedMessage("test-service", 8080)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Defer recover to catch any panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Function panicked: %v", r)
				}
			}()

			tt.fn()
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are defined with expected types and reasonable values
	if PortRangeStart < 1 || PortRangeStart > 65535 {
		t.Errorf("PortRangeStart = %d, should be in valid port range", PortRangeStart)
	}

	if PortRangeEnd < 1 || PortRangeEnd > 65535 {
		t.Errorf("PortRangeEnd = %d, should be in valid port range", PortRangeEnd)
	}

	if PortRangeStart >= PortRangeEnd {
		t.Errorf("PortRangeStart (%d) should be less than PortRangeEnd (%d)", PortRangeStart, PortRangeEnd)
	}

	if ProcessKillGracePeriod <= 0 {
		t.Errorf("ProcessKillGracePeriod = %v, should be positive", ProcessKillGracePeriod)
	}

	if ProcessKillMaxRetries < 1 {
		t.Errorf("ProcessKillMaxRetries = %d, should be at least 1", ProcessKillMaxRetries)
	}

	if ProcessKillTimeout <= 0 {
		t.Errorf("ProcessKillTimeout = %v, should be positive", ProcessKillTimeout)
	}

	if StalePortCleanupAge <= 0 {
		t.Errorf("StalePortCleanupAge = %v, should be positive", StalePortCleanupAge)
	}
}
