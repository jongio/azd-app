package dashboard

import (
	"testing"
)

func TestSubstituteQueryPlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		serviceName string
		timespan    string
		expected    string
	}{
		{
			name:        "replace service name",
			query:       "SELECT * FROM logs WHERE service = '{serviceName}'",
			serviceName: "api",
			timespan:    "1h",
			expected:    "SELECT * FROM logs WHERE service = 'api'",
		},
		{
			name:        "replace timespan",
			query:       "SELECT * FROM logs WHERE timestamp > ago({timespan})",
			serviceName: "web",
			timespan:    "24h",
			expected:    "SELECT * FROM logs WHERE timestamp > ago(24h)",
		},
		{
			name:        "replace both placeholders",
			query:       "SELECT * FROM logs WHERE service = '{serviceName}' AND timestamp > ago({timespan})",
			serviceName: "worker",
			timespan:    "12h",
			expected:    "SELECT * FROM logs WHERE service = 'worker' AND timestamp > ago(12h)",
		},
		{
			name:        "no placeholders",
			query:       "SELECT * FROM logs",
			serviceName: "api",
			timespan:    "1h",
			expected:    "SELECT * FROM logs",
		},
		{
			name:        "multiple occurrences",
			query:       "{serviceName} logs for {serviceName}",
			serviceName: "api",
			timespan:    "1h",
			expected:    "api logs for api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteQueryPlaceholders(tt.query, tt.serviceName, tt.timespan)
			if result != tt.expected {
				t.Errorf("substituteQueryPlaceholders() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		old      string
		new      string
		expected string
	}{
		{
			name:     "simple replacement",
			s:        "hello world",
			old:      "world",
			new:      "universe",
			expected: "hello universe",
		},
		{
			name:     "multiple occurrences",
			s:        "foo bar foo",
			old:      "foo",
			new:      "baz",
			expected: "baz bar baz",
		},
		{
			name:     "no match",
			s:        "hello world",
			old:      "xyz",
			new:      "abc",
			expected: "hello world",
		},
		{
			name:     "replace with empty",
			s:        "hello world",
			old:      " world",
			new:      "",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceAll(tt.s, tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("replaceAll() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "found at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "hello world",
			substr:   "xyz",
			expected: -1,
		},
		{
			name:     "empty substring",
			s:        "test",
			substr:   "",
			expected: 0,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: -1,
		},
		{
			name:     "exact match",
			s:        "test",
			substr:   "test",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("indexOf() = %v, want %v", result, tt.expected)
			}
		})
	}
}
