package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestLogsCommandHasRedactFlag(t *testing.T) {
	cmd := NewLogsCommand()
	flag := cmd.Flags().Lookup("redact")
	if flag == nil {
		t.Fatal("redact flag not found")
	}
	if flag.Value.Type() != "bool" {
		t.Fatalf("redact flag type = %s, want bool", flag.Value.Type())
	}
}

func TestRedactCollectedLogs(t *testing.T) {
	collected := &CollectedLogs{
		Entries: []service.LogEntry{
			{
				Service:   "api",
				Message:   "token=abc123456789 password: hunter222",
				Level:     service.LogLevelInfo,
				Timestamp: time.Now(),
			},
		},
		EntriesWithContext: []LogEntryWithContext{
			{
				Service: "worker",
				Message: "Authorization token=eyJaaaaaaaaaa.bbbbbbbbbb.cccccccccc",
				Context: &LogContext{
					Before: []string{"apiKey=1234567890"},
					After:  []string{"normal line"},
				},
			},
		},
	}

	redactCollectedLogs(collected)

	if strings.Contains(collected.Entries[0].Message, "abc123456789") || strings.Contains(collected.Entries[0].Message, "hunter222") {
		t.Fatalf("entry message was not redacted: %s", collected.Entries[0].Message)
	}
	if strings.Contains(collected.EntriesWithContext[0].Message, "eyJaaaaaaaaaa") {
		t.Fatalf("context entry message was not redacted: %s", collected.EntriesWithContext[0].Message)
	}
	if strings.Contains(collected.EntriesWithContext[0].Context.Before[0], "1234567890") {
		t.Fatalf("before context was not redacted: %s", collected.EntriesWithContext[0].Context.Before[0])
	}
	if collected.EntriesWithContext[0].Context.After[0] != "normal line" {
		t.Fatalf("non-sensitive context changed: %s", collected.EntriesWithContext[0].Context.After[0])
	}
}

func TestMaskSecretsInLogLineIsAvailableForShareableLogs(t *testing.T) {
	got := service.MaskSecretsInLogLine("connection token=abcdefghi")
	if strings.Contains(got, "abcdefghi") {
		t.Fatalf("secret value was not masked: %s", got)
	}
}
