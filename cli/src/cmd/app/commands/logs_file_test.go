package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadLogsFromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logs_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, ".azure", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}

	logContent := `[2024-01-15 10:30:45.100] [INFO] [OUT] Line 1
[2024-01-15 10:30:45.200] [INFO] [OUT] Line 2
[2024-01-15 10:30:45.300] [WARN] [OUT] Line 3
[2024-01-15 10:30:45.400] [ERROR] [ERR] Line 4
[2024-01-15 10:30:45.500] [INFO] [OUT] Line 5
`
	logFile := filepath.Join(logsDir, "api.log")
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("read all logs with tail", func(t *testing.T) {
		logs, err := readLogsFromFile(tmpDir, "api", 100, time.Time{})
		if err != nil {
			t.Fatalf("readLogsFromFile() error: %v", err)
		}
		if len(logs) != 5 {
			t.Errorf("Expected 5 logs, got %d", len(logs))
		}
	})

	t.Run("read with tail limit", func(t *testing.T) {
		logs, err := readLogsFromFile(tmpDir, "api", 3, time.Time{})
		if err != nil {
			t.Fatalf("readLogsFromFile() error: %v", err)
		}
		if len(logs) != 3 {
			t.Errorf("Expected 3 logs (tail limit), got %d", len(logs))
		}
		if logs[0].Message != "Line 3" {
			t.Errorf("First entry should be 'Line 3', got %q", logs[0].Message)
		}
	})

	t.Run("read with since filter", func(t *testing.T) {
		since := time.Date(2024, 1, 15, 10, 30, 45, 250000000, time.UTC)
		logs, err := readLogsFromFile(tmpDir, "api", 100, since)
		if err != nil {
			t.Fatalf("readLogsFromFile() error: %v", err)
		}
		if len(logs) != 3 {
			t.Errorf("Expected 3 logs after since filter, got %d", len(logs))
		}
	})

	t.Run("nonexistent service", func(t *testing.T) {
		_, err := readLogsFromFile(tmpDir, "nonexistent", 100, time.Time{})
		if err == nil {
			t.Error("Expected error for nonexistent service")
		}
	})

	t.Run("zero tail", func(t *testing.T) {
		logs, err := readLogsFromFile(tmpDir, "api", 0, time.Time{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(logs) != 5 {
			t.Errorf("Expected 5 entries, got %d", len(logs))
		}
	})

	t.Run("since filters all", func(t *testing.T) {
		since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := readLogsFromFile(tmpDir, "api", 100, since)
		if err == nil {
			logs, _ := readLogsFromFile(tmpDir, "api", 100, since)
			if len(logs) != 0 {
				t.Errorf("Expected 0 entries (all filtered by since), got %d", len(logs))
			}
		}
	})
}

func TestReadLogsFromRotatedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logs_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, ".azure", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}

	log2Content := `[2024-01-15 10:30:40.000] [INFO] [OUT] Oldest entry 1
[2024-01-15 10:30:41.000] [INFO] [OUT] Oldest entry 2
`
	log1Content := `[2024-01-15 10:30:42.000] [INFO] [OUT] Middle entry 1
[2024-01-15 10:30:43.000] [INFO] [OUT] Middle entry 2
`
	logContent := `[2024-01-15 10:30:44.000] [INFO] [OUT] Current entry 1
[2024-01-15 10:30:45.000] [INFO] [OUT] Current entry 2
`

	if err := os.WriteFile(filepath.Join(logsDir, "api.log.2"), []byte(log2Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "api.log.1"), []byte(log1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "api.log"), []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("read from all rotated files", func(t *testing.T) {
		logs, err := readLogsFromFile(tmpDir, "api", 100, time.Time{})
		if err != nil {
			t.Fatalf("readLogsFromFile() error: %v", err)
		}
		if len(logs) != 6 {
			t.Errorf("Expected 6 logs from all rotated files, got %d", len(logs))
		}
		if !strings.Contains(logs[0].Message, "Oldest entry 1") {
			t.Errorf("First entry should be oldest, got %q", logs[0].Message)
		}
		if !strings.Contains(logs[5].Message, "Current entry 2") {
			t.Errorf("Last entry should be newest, got %q", logs[5].Message)
		}
	})

	t.Run("tail limit across rotated files", func(t *testing.T) {
		logs, err := readLogsFromFile(tmpDir, "api", 3, time.Time{})
		if err != nil {
			t.Fatalf("readLogsFromFile() error: %v", err)
		}
		if len(logs) != 3 {
			t.Errorf("Expected 3 logs with tail limit, got %d", len(logs))
		}
		if !strings.Contains(logs[0].Message, "Middle entry 2") {
			t.Errorf("First of tail should be 'Middle entry 2', got %q", logs[0].Message)
		}
	})

	t.Run("partial rotated files", func(t *testing.T) {
		content1 := `[2024-01-15 10:30:44.000] [INFO] [OUT] Backup entry
`
		content := `[2024-01-15 10:30:45.000] [INFO] [OUT] Current entry
`
		if err := os.WriteFile(filepath.Join(logsDir, "partial.log.1"), []byte(content1), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(logsDir, "partial.log"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		logs, err := readLogsFromFile(tmpDir, "partial", 100, time.Time{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 entries from 2 files, got %d", len(logs))
		}
	})
}

func TestReadSingleLogFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logs_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("file not found", func(t *testing.T) {
		_, err := readSingleLogFile(filepath.Join(tmpDir, "nonexistent.log"), "api", time.Time{})
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.log")
		if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		entries, err := readSingleLogFile(emptyFile, "api", time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("Expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("malformed lines", func(t *testing.T) {
		badFile := filepath.Join(tmpDir, "malformed.log")
		content := `not a valid log line
another invalid line
[2024-01-15 10:30:45.123] [INFO] [OUT] Valid line
malformed again
[2024-01-15 10:30:46.000] [ERROR] [ERR] Another valid line
`
		if err := os.WriteFile(badFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		entries, err := readSingleLogFile(badFile, "api", time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("Expected 2 valid entries, got %d", len(entries))
		}
	})

	t.Run("long log lines", func(t *testing.T) {
		longFile := filepath.Join(tmpDir, "long.log")
		longMsg := strings.Repeat("x", 100000)
		content := fmt.Sprintf("[2024-01-15 10:30:45.123] [INFO] [OUT] %s\n", longMsg)
		if err := os.WriteFile(longFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		entries, err := readSingleLogFile(longFile, "api", time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(entries))
		}
		if len(entries) > 0 && len(entries[0].Message) != 100000 {
			t.Errorf("Message length = %d, want 100000", len(entries[0].Message))
		}
	})

	t.Run("line exceeds max buffer size causes scanner error", func(t *testing.T) {
		hugeFile := filepath.Join(tmpDir, "huge.log")
		hugeMsg := strings.Repeat("x", maxLogLineSize+1000)
		content := fmt.Sprintf("[2024-01-15 10:30:45.123] [INFO] [OUT] %s", hugeMsg)
		if err := os.WriteFile(hugeFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readSingleLogFile(hugeFile, "api", time.Time{})
		if err == nil {
			t.Error("Expected error for line exceeding max buffer size")
		}
	})
}
