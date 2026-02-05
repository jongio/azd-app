package service

import (
	"errors"
	"io"
	"testing"
)

type mockCloser struct {
	closeFunc func() error
	closed    bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestSafeClose(t *testing.T) {
	t.Run("closes successfully", func(t *testing.T) {
		closer := &mockCloser{}
		SafeClose(closer, "test resource")
		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
	})

	t.Run("handles nil closer", func(t *testing.T) {
		// Should not panic
		SafeClose(nil, "nil resource")
	})

	t.Run("handles close error", func(t *testing.T) {
		closer := &mockCloser{
			closeFunc: func() error {
				return errors.New("close error")
			},
		}
		// Should not panic, just log
		SafeClose(closer, "failing resource")
		if !closer.closed {
			t.Error("Expected closer to be called despite error")
		}
	})
}

func TestSafeCloseWithContext(t *testing.T) {
	t.Run("closes successfully with context", func(t *testing.T) {
		closer := &mockCloser{}
		SafeCloseWithContext(closer, "test resource", "key1", "value1", "key2", 42)
		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
	})

	t.Run("handles nil closer with context", func(t *testing.T) {
		// Should not panic
		SafeCloseWithContext(nil, "nil resource", "key", "value")
	})

	t.Run("handles close error with context", func(t *testing.T) {
		closer := &mockCloser{
			closeFunc: func() error {
				return errors.New("close error")
			},
		}
		// Should not panic, just log
		SafeCloseWithContext(closer, "failing resource", "service", "api", "port", 8080)
		if !closer.closed {
			t.Error("Expected closer to be called despite error")
		}
	})

	t.Run("empty context fields", func(t *testing.T) {
		closer := &mockCloser{}
		SafeCloseWithContext(closer, "test resource")
		if !closer.closed {
			t.Error("Expected closer to be closed")
		}
	})
}

// Also test with real io.Closer implementations
func TestSafeClose_RealCloser(t *testing.T) {
	t.Run("with pipe reader", func(t *testing.T) {
		r, w := io.Pipe()
		_ = w.Close() // Close writer first
		SafeClose(r, "pipe reader")
		// No assertion needed - just verify no panic
	})
}
