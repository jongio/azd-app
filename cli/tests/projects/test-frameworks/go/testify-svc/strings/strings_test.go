package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverse(t *testing.T) {
	assert.Equal(t, "olleh", Reverse("hello"))
	assert.Equal(t, "", Reverse(""))
	assert.Equal(t, "a", Reverse("a"))
}

func TestIsPalindrome(t *testing.T) {
	assert.True(t, IsPalindrome("radar"))
	assert.True(t, IsPalindrome("Racecar"))
	assert.False(t, IsPalindrome("hello"))
	assert.True(t, IsPalindrome(""))
}

func TestCapitalize(t *testing.T) {
	assert.Equal(t, "Hello", Capitalize("hello"))
	assert.Equal(t, "H", Capitalize("h"))
	assert.Equal(t, "", Capitalize(""))
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "helloWorld"},
		{"snake_case_string", "snakeCaseString"},
		{"already", "already"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, ToCamelCase(tc.input))
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	assert.Equal(t, "hello_world", ToSnakeCase("helloWorld"))
	assert.Equal(t, "snake_case_string", ToSnakeCase("snakeCaseString"))
	assert.Equal(t, "already", ToSnakeCase("already"))
}

func TestCountWords(t *testing.T) {
	assert.Equal(t, 3, CountWords("hello world test"))
	assert.Equal(t, 0, CountWords(""))
	assert.Equal(t, 1, CountWords("word"))
	assert.Equal(t, 2, CountWords("  spaced   words  "))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello...", Truncate("hello world", 8, "..."))
	assert.Equal(t, "short", Truncate("short", 10, "..."))
	assert.Equal(t, "...", Truncate("hello", 3, "..."))
}

func TestRemoveWhitespace(t *testing.T) {
	assert.Equal(t, "helloworld", RemoveWhitespace("hello world"))
	assert.Equal(t, "abc", RemoveWhitespace("  a b  c  "))
	assert.Equal(t, "notabs", RemoveWhitespace("no\ttabs"))
}

func TestIsEmpty(t *testing.T) {
	assert.True(t, IsEmpty(""))
	assert.True(t, IsEmpty("   "))
	assert.True(t, IsEmpty("\t\n"))
	assert.False(t, IsEmpty("hello"))
	assert.False(t, IsEmpty(" x "))
}

func TestContains(t *testing.T) {
	assert.True(t, Contains("Hello World", "world"))
	assert.True(t, Contains("hello", "HELLO"))
	assert.False(t, Contains("hello", "xyz"))
}

func TestCountOccurrences(t *testing.T) {
	assert.Equal(t, 2, CountOccurrences("abcabc", "ab"))
	assert.Equal(t, 0, CountOccurrences("hello", "xyz"))
	assert.Equal(t, 2, CountOccurrences("banana", "an"))
}

func TestReplaceAll(t *testing.T) {
	assert.Equal(t, "hXllo world", ReplaceAll("hello world", "e", "X"))
	assert.Equal(t, "hello", ReplaceAll("hello", "x", "y"))
}

func TestExtractEmails(t *testing.T) {
	text := "Contact us at test@example.com or admin@site.org"
	emails := ExtractEmails(text)

	require.Len(t, emails, 2)
	assert.Contains(t, emails, "test@example.com")
	assert.Contains(t, emails, "admin@site.org")
}

func TestExtractNumbers(t *testing.T) {
	nums := ExtractNumbers("abc123def456")
	require.Len(t, nums, 2)
	assert.Equal(t, "123", nums[0])
	assert.Equal(t, "456", nums[1])
}

func TestRepeat(t *testing.T) {
	assert.Equal(t, "abcabcabc", Repeat("abc", 3))
	assert.Equal(t, "", Repeat("abc", 0))
	assert.Equal(t, "", Repeat("abc", -1))
}

func TestPadLeft(t *testing.T) {
	assert.Equal(t, "00042", PadLeft("42", 5, '0'))
	assert.Equal(t, "hello", PadLeft("hello", 3, ' '))
}

func TestPadRight(t *testing.T) {
	assert.Equal(t, "42000", PadRight("42", 5, '0'))
	assert.Equal(t, "hi   ", PadRight("hi", 5, ' '))
}

func TestIsAlpha(t *testing.T) {
	assert.True(t, IsAlpha("hello"))
	assert.True(t, IsAlpha("ABC"))
	assert.False(t, IsAlpha("hello123"))
	assert.False(t, IsAlpha(""))
	assert.False(t, IsAlpha("hello world"))
}

func TestIsNumeric(t *testing.T) {
	assert.True(t, IsNumeric("123"))
	assert.False(t, IsNumeric("12.3"))
	assert.False(t, IsNumeric(""))
	assert.False(t, IsNumeric("abc"))
}

func TestIsAlphanumeric(t *testing.T) {
	assert.True(t, IsAlphanumeric("hello123"))
	assert.True(t, IsAlphanumeric("ABC"))
	assert.True(t, IsAlphanumeric("123"))
	assert.False(t, IsAlphanumeric(""))
	assert.False(t, IsAlphanumeric("hello world"))
	assert.False(t, IsAlphanumeric("hello!"))
}

// Test using require for critical assertions
func TestExtractEmailsWithRequire(t *testing.T) {
	text := "No emails here"
	emails := ExtractEmails(text)
	require.Len(t, emails, 0)
}

// Table-driven tests with subtests
func TestCapitalizeTableDriven(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"lowercase":   {"hello", "Hello"},
		"uppercase":   {"HELLO", "HELLO"},
		"mixed":       {"hELLO", "HELLO"},
		"empty":       {"", ""},
		"single char": {"x", "X"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := Capitalize(tc.input)
			assert.Equal(t, tc.expected, result, "Capitalize(%q) should be %q", tc.input, tc.expected)
		})
	}
}
