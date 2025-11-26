package service

import (
	"testing"
)

func TestLogFilter_ShouldFilter(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		message  string
		want     bool
	}{
		{
			name:     "matches exact pattern",
			patterns: []string{"npm warn"},
			message:  "npm warn Unknown env config",
			want:     true,
		},
		{
			name:     "matches regex pattern",
			patterns: []string{`Request Autofill\.enable failed`},
			message:  `[8324:1126/105630.828:ERROR:CONSOLE(1)] "Request Autofill.enable failed. {"code":-32601}`,
			want:     true,
		},
		{
			name:     "case insensitive match",
			patterns: []string{"NPM WARN"},
			message:  "npm warn something",
			want:     true,
		},
		{
			name:     "no match",
			patterns: []string{"npm warn"},
			message:  "Server started on port 3000",
			want:     false,
		},
		{
			name:     "empty filter",
			patterns: []string{},
			message:  "any message",
			want:     false,
		},
		{
			name:     "multiple patterns - first matches",
			patterns: []string{"npm warn", "error occurred"},
			message:  "npm warn something",
			want:     true,
		},
		{
			name:     "multiple patterns - second matches",
			patterns: []string{"npm warn", "error occurred"},
			message:  "error occurred at line 10",
			want:     true,
		},
		{
			name:     "multiple patterns - none match",
			patterns: []string{"npm warn", "error occurred"},
			message:  "Server started successfully",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lf, err := NewLogFilter(tt.patterns)
			if err != nil {
				t.Fatalf("NewLogFilter() error = %v", err)
			}

			got := lf.ShouldFilter(tt.message)
			if got != tt.want {
				t.Errorf("ShouldFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogFilter_NilFilter(t *testing.T) {
	var lf *LogFilter
	if lf.ShouldFilter("any message") {
		t.Error("nil filter should not filter any message")
	}
}

func TestNewLogFilterWithBuiltins(t *testing.T) {
	lf, err := NewLogFilterWithBuiltins(nil)
	if err != nil {
		t.Fatalf("NewLogFilterWithBuiltins() error = %v", err)
	}

	// Test built-in patterns
	tests := []struct {
		message string
		want    bool
	}{
		{"Request Autofill.enable failed", true},
		{"npm warn Unknown env config", true},
		{"Debugger listening on ws://127.0.0.1:5858/abc", true},
		{"Server started on port 3000", false},
	}

	for _, tt := range tests {
		t.Run(tt.message[:min(30, len(tt.message))], func(t *testing.T) {
			got := lf.ShouldFilter(tt.message)
			if got != tt.want {
				t.Errorf("ShouldFilter(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestNewLogFilterWithBuiltins_CustomPatterns(t *testing.T) {
	customPatterns := []string{"my custom pattern", "another pattern"}
	lf, err := NewLogFilterWithBuiltins(customPatterns)
	if err != nil {
		t.Fatalf("NewLogFilterWithBuiltins() error = %v", err)
	}

	// Should filter both built-in and custom patterns
	tests := []struct {
		message string
		want    bool
	}{
		{"Request Autofill.enable failed", true},                 // built-in
		{"This has my custom pattern in it", true},               // custom
		{"This has another pattern in it", true},                 // custom
		{"Server started on port 3000", false},                   // neither
	}

	for _, tt := range tests {
		got := lf.ShouldFilter(tt.message)
		if got != tt.want {
			t.Errorf("ShouldFilter(%q) = %v, want %v", tt.message, got, tt.want)
		}
	}
}

func TestLogFilterConfig_ShouldIncludeBuiltins(t *testing.T) {
	tests := []struct {
		name   string
		config *LogFilterConfig
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			want:   true,
		},
		{
			name:   "nil includeBuiltins",
			config: &LogFilterConfig{},
			want:   true,
		},
		{
			name: "includeBuiltins true",
			config: &LogFilterConfig{
				IncludeBuiltins: boolPtr(true),
			},
			want: true,
		},
		{
			name: "includeBuiltins false",
			config: &LogFilterConfig{
				IncludeBuiltins: boolPtr(false),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldIncludeBuiltins()
			if got != tt.want {
				t.Errorf("ShouldIncludeBuiltins() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogFilterConfig_BuildLogFilter(t *testing.T) {
	tests := []struct {
		name          string
		config        *LogFilterConfig
		testMessage   string
		shouldFilter  bool
	}{
		{
			name:          "nil config uses builtins",
			config:        nil,
			testMessage:   "npm warn Unknown env config",
			shouldFilter:  true,
		},
		{
			name:          "config with builtins",
			config:        &LogFilterConfig{IncludeBuiltins: boolPtr(true)},
			testMessage:   "npm warn Unknown env config",
			shouldFilter:  true,
		},
		{
			name:          "config without builtins",
			config:        &LogFilterConfig{IncludeBuiltins: boolPtr(false)},
			testMessage:   "npm warn Unknown env config",
			shouldFilter:  false,
		},
		{
			name: "config with custom patterns",
			config: &LogFilterConfig{
				Exclude:         []string{"custom pattern"},
				IncludeBuiltins: boolPtr(false),
			},
			testMessage:   "this has custom pattern",
			shouldFilter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lf, err := tt.config.BuildLogFilter()
			if err != nil {
				t.Fatalf("BuildLogFilter() error = %v", err)
			}

			got := lf.ShouldFilter(tt.testMessage)
			if got != tt.shouldFilter {
				t.Errorf("ShouldFilter(%q) = %v, want %v", tt.testMessage, got, tt.shouldFilter)
			}
		})
	}
}

func TestFilterLogEntries(t *testing.T) {
	entries := []LogEntry{
		{Service: "api", Message: "Server started on port 3000"},
		{Service: "api", Message: "npm warn Unknown env config"},
		{Service: "web", Message: "Listening on http://localhost:5173"},
		{Service: "web", Message: "Request Autofill.enable failed"},
	}

	lf, err := NewLogFilterWithBuiltins(nil)
	if err != nil {
		t.Fatalf("NewLogFilterWithBuiltins() error = %v", err)
	}

	filtered := FilterLogEntries(entries, lf)

	// Should only have 2 entries (the non-noise ones)
	if len(filtered) != 2 {
		t.Errorf("FilterLogEntries() returned %d entries, want 2", len(filtered))
	}

	// Verify the correct entries remain
	for _, entry := range filtered {
		if lf.ShouldFilter(entry.Message) {
			t.Errorf("FilterLogEntries() left in a filtered message: %q", entry.Message)
		}
	}
}

func TestFilterLogEntries_NilFilter(t *testing.T) {
	entries := []LogEntry{
		{Service: "api", Message: "Server started"},
		{Service: "api", Message: "npm warn something"},
	}

	filtered := FilterLogEntries(entries, nil)

	if len(filtered) != len(entries) {
		t.Errorf("FilterLogEntries(nil) = %d entries, want %d", len(filtered), len(entries))
	}
}

func TestParseExcludePatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single pattern",
			input: "npm warn",
			want:  []string{"npm warn"},
		},
		{
			name:  "multiple patterns",
			input: "npm warn,error,warning",
			want:  []string{"npm warn", "error", "warning"},
		},
		{
			name:  "patterns with spaces",
			input: " npm warn , error , warning ",
			want:  []string{"npm warn", "error", "warning"},
		},
		{
			name:  "empty segments ignored",
			input: "npm warn,,error",
			want:  []string{"npm warn", "error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseExcludePatterns(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseExcludePatterns() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseExcludePatterns()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLogFilter_AddPattern(t *testing.T) {
	lf, err := NewLogFilter([]string{"initial"})
	if err != nil {
		t.Fatalf("NewLogFilter() error = %v", err)
	}

	if lf.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", lf.PatternCount())
	}

	err = lf.AddPattern("added")
	if err != nil {
		t.Fatalf("AddPattern() error = %v", err)
	}

	if lf.PatternCount() != 2 {
		t.Errorf("PatternCount() after add = %d, want 2", lf.PatternCount())
	}

	if !lf.ShouldFilter("this is added pattern") {
		t.Error("ShouldFilter() should match added pattern")
	}
}

func TestLogFilter_GetPatterns(t *testing.T) {
	patterns := []string{"pattern1", "pattern2"}
	lf, err := NewLogFilter(patterns)
	if err != nil {
		t.Fatalf("NewLogFilter() error = %v", err)
	}

	got := lf.GetPatterns()
	if len(got) != len(patterns) {
		t.Errorf("GetPatterns() = %v, want %v", got, patterns)
	}
}

func TestLogsConfig_GetFilters(t *testing.T) {
	tests := []struct {
		name   string
		config *LogsConfig
		want   bool // whether GetFilters returns non-nil
	}{
		{
			name:   "nil config",
			config: nil,
			want:   false,
		},
		{
			name:   "empty config",
			config: &LogsConfig{},
			want:   false,
		},
		{
			name: "config with filters",
			config: &LogsConfig{
				Filters: &LogFilterConfig{
					Exclude: []string{"test"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetFilters()
			hasFilters := got != nil
			if hasFilters != tt.want {
				t.Errorf("GetFilters() returned non-nil = %v, want %v", hasFilters, tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
