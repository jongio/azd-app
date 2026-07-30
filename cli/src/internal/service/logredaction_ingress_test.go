package service

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dashboard, the Connect-RPC log stream, and the MCP log tools all read
// from the ring buffer. Masking used to happen only on the file-write path,
// so a service that printed a credential kept it out of the log file but
// still handed it to the browser and to any connected LLM.
func TestLogBufferAddMasksSecretsBeforeTheyReachConsumers(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		mustNotHave string
	}{
		{
			name:        "key value token",
			message:     "starting with token=ghp_abcdefghijklmnopqrstuvwxyz123456",
			mustNotHave: "ghp_abcdefghijklmnopqrstuvwxyz123456",
		},
		{
			name:        "jwt bearer",
			message:     "auth eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r",
			mustNotHave: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r",
		},
		{
			name:        "password assignment",
			message:     "connecting password=SuperSecret123!",
			mustNotHave: "SuperSecret123!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewLogBuffer("svc", 16, false, t.TempDir())
			require.NoError(t, err)

			buf.Add(LogEntry{
				Service:   "svc",
				Message:   tt.message,
				Timestamp: time.Now(),
			})

			got := buf.GetRecent(10)
			require.Len(t, got, 1)
			assert.NotContains(t, got[0].Message, tt.mustNotHave,
				"raw secret must not survive into the ring buffer")
			assert.Contains(t, got[0].Message, "***", "value should be masked")
		})
	}
}

func TestLogBufferAddLeavesOrdinaryLinesUntouched(t *testing.T) {
	buf, err := NewLogBuffer("svc", 16, false, t.TempDir())
	require.NoError(t, err)

	const msg = "listening on http://localhost:3000"
	buf.Add(LogEntry{Service: "svc", Message: msg, Timestamp: time.Now()})

	got := buf.GetRecent(10)
	require.Len(t, got, 1)
	assert.Equal(t, msg, got[0].Message)
}

// discardStream is the safety valve used when no log buffer could be created.
// Without it the child process blocks once the OS pipe buffer fills.
func TestDiscardStreamDrainsAndClosesReader(t *testing.T) {
	payload := strings.Repeat("noisy output line\n", 5000)
	rc := &countingReadCloser{Reader: bytes.NewReader([]byte(payload))}

	discardStream(rc, "svc", false)

	assert.True(t, rc.closed, "reader must be closed so the pipe is released")
	n, err := rc.Read(make([]byte, 1))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err, "stream should be fully drained")
}

func TestDiscardStreamToleratesNilReader(t *testing.T) {
	assert.NotPanics(t, func() { discardStream(nil, "svc", true) })
}

type countingReadCloser struct {
	io.Reader
	closed bool
}

func (c *countingReadCloser) Close() error {
	c.closed = true
	return nil
}
