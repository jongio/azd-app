package service

import (
	"testing"
)

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		// Valid names
		{"valid uppercase", "MY_VAR", true},
		{"valid lowercase", "my_var", true},
		{"valid mixed", "My_Var_123", true},
		{"valid underscore start", "_private", true},
		{"valid single char", "X", true},
		{"valid with numbers", "VAR123", true},

		// Invalid names - security critical
		{"empty", "", false},
		{"starts with number", "1VAR", false},
		{"contains newline", "VAR\nINJECTION", false},
		{"contains carriage return", "VAR\rINJECTION", false},
		{"contains tab", "VAR\tINJECTION", false},
		{"contains null byte", "VAR\000INJECTION", false},
		{"contains equals", "VAR=value", false},
		{"contains dollar", "VAR$INJECTION", false},
		{"contains semicolon", "VAR;cmd", false},
		{"contains pipe", "VAR|cmd", false},
		{"contains ampersand", "VAR&cmd", false},
		{"contains redirect", "VAR>file", false},
		{"contains backtick", "VAR`cmd`", false},
		{"contains quote", "VAR\"quote", false},
		{"contains single quote", "VAR'quote", false},
		{"contains backslash", "VAR\\path", false},
		{"contains paren", "VAR()", false},
		{"contains bracket", "VAR[]", false},
		{"contains brace", "VAR{}", false},
		{"contains less than", "VAR<file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEnvVarName(tt.varName)
			if result != tt.expected {
				t.Errorf("isValidEnvVarName(%q) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestEnvSliceToMap_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "filters invalid var names",
			input:    []string{"VALID=value", "IN;VALID=value", "ALSO_VALID=value"},
			expected: map[string]string{"VALID": "value", "ALSO_VALID": "value"},
		},
		{
			name:     "filters null bytes in keys",
			input:    []string{"VALID=value", "BAD\000KEY=value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters null bytes in values",
			input:    []string{"VALID=value", "KEY=bad\000value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters newlines in keys",
			input:    []string{"VALID=value", "BAD\nKEY=value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "handles empty strings",
			input:    []string{"", "VALID=value", ""},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters keys starting with numbers",
			input:    []string{"VALID=value", "1BAD=value"},
			expected: map[string]string{"VALID": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := envSliceToMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("envSliceToMap() returned %d items, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("envSliceToMap()[%q] = %q, want %q", k, result[k], v)
				}
			}
			// Ensure no unexpected keys
			for k := range result {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("envSliceToMap() has unexpected key %q", k)
				}
			}
		})
	}
}
