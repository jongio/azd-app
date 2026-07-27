package service

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectContainerLogs_ParsesLines(t *testing.T) {
	logOutput := "2026-01-01 INFO starting service\nERROR failed to connect\nDEBUG retry attempt 3\n"
	reader := io.NopCloser(strings.NewReader(logOutput))

	buf, err := NewLogBuffer("test-svc", 100, false, t.TempDir())
	require.NoError(t, err)

	collectContainerLogs(reader, "test-svc", buf)

	entries := buf.GetRecent(100)
	require.Len(t, entries, 3)

	assert.Equal(t, "test-svc", entries[0].Service)
	assert.Contains(t, entries[0].Message, "INFO starting service")
	assert.Contains(t, entries[1].Message, "ERROR failed to connect")
	assert.Contains(t, entries[2].Message, "DEBUG retry attempt")
}

func TestCollectContainerLogs_InfersLogLevel(t *testing.T) {
	logOutput := "INFO all good\nERROR something broke\nWARN careful\njust a plain line\n"
	reader := io.NopCloser(strings.NewReader(logOutput))

	buf, err := NewLogBuffer("level-svc", 100, false, t.TempDir())
	require.NoError(t, err)

	collectContainerLogs(reader, "level-svc", buf)

	entries := buf.GetRecent(100)
	require.Len(t, entries, 4)

	assert.Equal(t, LogLevelInfo, entries[0].Level)
	assert.Equal(t, LogLevelError, entries[1].Level)
	assert.Equal(t, LogLevelWarn, entries[2].Level)
	assert.Equal(t, LogLevelInfo, entries[3].Level)
}

func TestCollectContainerLogs_EmptyInput(t *testing.T) {
	reader := io.NopCloser(strings.NewReader(""))

	buf, err := NewLogBuffer("empty-svc", 100, false, t.TempDir())
	require.NoError(t, err)

	collectContainerLogs(reader, "empty-svc", buf)

	assert.Empty(t, buf.GetRecent(100))
}

func TestCollectContainerLogs_SetsTimestamp(t *testing.T) {
	reader := io.NopCloser(strings.NewReader("hello world\n"))

	buf, err := NewLogBuffer("ts-svc", 100, false, t.TempDir())
	require.NoError(t, err)

	before := time.Now()
	collectContainerLogs(reader, "ts-svc", buf)
	after := time.Now()

	entries := buf.GetRecent(100)
	require.Len(t, entries, 1)

	// Timestamp should be within the test's time window.
	assert.False(t, entries[0].Timestamp.Before(before))
	assert.False(t, entries[0].Timestamp.After(after))
}

func TestIsContainerRunning_EmptyContainerID(t *testing.T) {
	proc := &ServiceProcess{Name: "svc", ContainerID: ""}
	assert.False(t, IsContainerRunning(proc))
}

func TestStopContainerService_NilProcess(t *testing.T) {
	err := StopContainerService(nil, 10*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestStopContainerService_EmptyContainerID(t *testing.T) {
	proc := &ServiceProcess{Name: "svc", ContainerID: ""}
	err := StopContainerService(proc, 10*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no container ID")
}

func TestStartContainerLogCollection_EmptyContainerID(t *testing.T) {
	proc := &ServiceProcess{Name: "svc", ContainerID: ""}
	err := StartContainerLogCollection(proc, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no container ID")
}
