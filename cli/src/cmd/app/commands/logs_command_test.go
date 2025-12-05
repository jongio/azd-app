package commands

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestValidateLogsInputs(t *testing.T) {
	origTail := logsTail
	origFormat := logsFormat
	origLevel := logsLevel
	origSince := logsSince
	defer func() {
		logsTail = origTail
		logsFormat = origFormat
		logsLevel = origLevel
		logsSince = origSince
	}()

	tests := []struct {
		name      string
		tail      int
		format    string
		level     string
		since     string
		wantErr   bool
		errSubstr string
	}{
		{"valid defaults", 100, "text", "all", "", false, ""},
		{"valid json format", 100, "json", "all", "", false, ""},
		{"valid level info", 100, "text", "info", "", false, ""},
		{"valid level warn", 100, "text", "warn", "", false, ""},
		{"valid level error", 100, "text", "error", "", false, ""},
		{"valid level debug", 100, "text", "debug", "", false, ""},
		{"valid since 5m", 100, "text", "all", "5m", false, ""},
		{"valid since 1h", 100, "text", "all", "1h", false, ""},
		{"negative tail", -1, "text", "all", "", true, "--tail must be a positive"},
		{"invalid format", 100, "xml", "all", "", true, "--format must be"},
		{"invalid level", 100, "text", "trace", "", true, "--level must be one of"},
		{"invalid since", 100, "text", "all", "5x", true, "--since must be a valid duration"},
		{"tail capped at max", 20000, "text", "all", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logsTail = tt.tail
			logsFormat = tt.format
			logsLevel = tt.level
			logsSince = tt.since

			err := validateLogsInputs()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateLogsInputs() expected error containing %q, got nil", tt.errSubstr)
				} else if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateLogsInputs() error = %v, want error containing %q", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("validateLogsInputs() unexpected error: %v", err)
				}
			}
		})
	}

	t.Run("tail capped value", func(t *testing.T) {
		logsTail = 20000
		logsFormat = "text"
		logsLevel = "all"
		logsSince = ""

		if err := validateLogsInputs(); err != nil {
			t.Fatalf("validateLogsInputs() unexpected error: %v", err)
		}
		if logsTail != maxTailLines {
			t.Errorf("logsTail = %d, expected to be capped at %d", logsTail, maxTailLines)
		}
	})

	t.Run("zero tail is valid", func(t *testing.T) {
		logsTail = 0
		logsFormat = "text"
		logsLevel = "all"
		logsSince = ""

		if err := validateLogsInputs(); err != nil {
			t.Errorf("Unexpected error for tail=0: %v", err)
		}
	})

	t.Run("case insensitive level", func(t *testing.T) {
		levels := []string{"INFO", "Info", "WARNING", "Warning", "ERROR", "Error", "DEBUG", "Debug", "ALL", "All"}
		for _, level := range levels {
			logsTail = 100
			logsFormat = "text"
			logsLevel = level
			logsSince = ""

			if err := validateLogsInputs(); err != nil {
				t.Errorf("Unexpected error for level %q: %v", level, err)
			}
		}
	})

	t.Run("complex since durations", func(t *testing.T) {
		durations := []string{"1h30m", "2h45m30s", "500ms", "1h", "30s"}
		for _, d := range durations {
			logsTail = 100
			logsFormat = "text"
			logsLevel = "all"
			logsSince = d

			if err := validateLogsInputs(); err != nil {
				t.Errorf("Unexpected error for since %q: %v", d, err)
			}
		}
	})
}

func TestLogsConstants(t *testing.T) {
	t.Run("defaultTailLines", func(t *testing.T) {
		if defaultTailLines != 100 {
			t.Errorf("defaultTailLines = %d, expected 100", defaultTailLines)
		}
	})

	t.Run("maxTailLines", func(t *testing.T) {
		if maxTailLines != 10000 {
			t.Errorf("maxTailLines = %d, expected 10000", maxTailLines)
		}
	})

	t.Run("logChannelBufferSize", func(t *testing.T) {
		if logChannelBufferSize != 100 {
			t.Errorf("logChannelBufferSize = %d, expected 100", logChannelBufferSize)
		}
	})
}

func TestLogLevelAllConstant(t *testing.T) {
	if LogLevelAll != -1 {
		t.Errorf("LogLevelAll = %d, expected -1", LogLevelAll)
	}

	levels := []service.LogLevel{
		service.LogLevelInfo,
		service.LogLevelWarn,
		service.LogLevelError,
		service.LogLevelDebug,
	}

	for _, level := range levels {
		if LogLevelAll == level {
			t.Errorf("LogLevelAll should be different from %v", level)
		}
	}
}

func TestMaxLogLineSizeConstant(t *testing.T) {
	expectedSize := 1 * 1024 * 1024
	if maxLogLineSize != expectedSize {
		t.Errorf("maxLogLineSize = %d, expected %d (1MB)", maxLogLineSize, expectedSize)
	}
}

func TestLogsCommandStructure(t *testing.T) {
	cmd := NewLogsCommand()

	t.Run("command properties", func(t *testing.T) {
		if cmd.Use != "logs [service-name]" {
			t.Errorf("Use = %q, want %q", cmd.Use, "logs [service-name]")
		}
		if cmd.Short == "" {
			t.Error("Short should not be empty")
		}
		if cmd.Long == "" {
			t.Error("Long should not be empty")
		}
		if !cmd.SilenceUsage {
			t.Error("SilenceUsage should be true")
		}
	})

	t.Run("flags exist", func(t *testing.T) {
		flags := []string{
			"follow", "service", "tail", "since", "timestamps",
			"no-color", "level", "format", "file", "exclude", "no-builtins",
		}
		for _, flag := range flags {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("Flag %q should exist", flag)
			}
		}
	})

	t.Run("shorthand flags", func(t *testing.T) {
		if cmd.Flags().ShorthandLookup("f") == nil {
			t.Error("Shorthand -f should exist for --follow")
		}
		if cmd.Flags().ShorthandLookup("s") == nil {
			t.Error("Shorthand -s should exist for --service")
		}
		if cmd.Flags().ShorthandLookup("n") == nil {
			t.Error("Shorthand -n should exist for --tail")
		}
		if cmd.Flags().ShorthandLookup("e") == nil {
			t.Error("Shorthand -e should exist for --exclude")
		}
	})

	t.Run("default values", func(t *testing.T) {
		tailFlag := cmd.Flags().Lookup("tail")
		if tailFlag.DefValue != "100" {
			t.Errorf("tail default = %q, want %q", tailFlag.DefValue, "100")
		}

		timestampsFlag := cmd.Flags().Lookup("timestamps")
		if timestampsFlag.DefValue != "true" {
			t.Errorf("timestamps default = %q, want %q", timestampsFlag.DefValue, "true")
		}

		levelFlag := cmd.Flags().Lookup("level")
		if levelFlag.DefValue != "all" {
			t.Errorf("level default = %q, want %q", levelFlag.DefValue, "all")
		}

		formatFlag := cmd.Flags().Lookup("format")
		if formatFlag.DefValue != "text" {
			t.Errorf("format default = %q, want %q", formatFlag.DefValue, "text")
		}
	})
}

// TestLogsOptionsDocumentation verifies the logsOptions struct is properly defined.
func TestLogsOptionsDocumentation(t *testing.T) {
	opts := logsOptions{
		follow:     true,
		service:    "api",
		tail:       100,
		since:      "5m",
		timestamps: true,
		noColor:    false,
		level:      "info",
		format:     "text",
		file:       "output.log",
		exclude:    "pattern",
		noBuiltins: false,
	}

	if opts.follow != true {
		t.Error("follow field")
	}
	if opts.service != "api" {
		t.Error("service field")
	}
	if opts.tail != 100 {
		t.Error("tail field")
	}
	if opts.since != "5m" {
		t.Error("since field")
	}
	if opts.timestamps != true {
		t.Error("timestamps field")
	}
	if opts.noColor != false {
		t.Error("noColor field")
	}
	if opts.level != "info" {
		t.Error("level field")
	}
	if opts.format != "text" {
		t.Error("format field")
	}
	if opts.file != "output.log" {
		t.Error("file field")
	}
	if opts.exclude != "pattern" {
		t.Error("exclude field")
	}
	if opts.noBuiltins != false {
		t.Error("noBuiltins field")
	}
}
