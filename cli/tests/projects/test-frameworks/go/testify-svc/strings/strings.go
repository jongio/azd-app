// Package strings provides string manipulation utilities
package strings

import (
	"regexp"
	"strings"
	"unicode"
)

// Reverse returns the reversed string
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPalindrome checks if string is a palindrome
func IsPalindrome(s string) bool {
	s = strings.ToLower(s)
	return s == Reverse(s)
}

// Capitalize capitalizes first letter
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ToCamelCase converts snake_case to camelCase
func ToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = Capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

// ToSnakeCase converts camelCase to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// CountWords counts words in a string
func CountWords(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}

// Truncate truncates string to maxLen with suffix
func Truncate(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}
	return s[:maxLen-len(suffix)] + suffix
}

// RemoveWhitespace removes all whitespace
func RemoveWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// IsEmpty checks if string is empty or whitespace
func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// Contains checks if string contains substring (case insensitive)
func Contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// CountOccurrences counts occurrences of substr
func CountOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}

// ReplaceAll replaces all occurrences
func ReplaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// ExtractEmails extracts email addresses from text
func ExtractEmails(s string) []string {
	re := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	return re.FindAllString(s, -1)
}

// ExtractNumbers extracts numbers from text
func ExtractNumbers(s string) []string {
	re := regexp.MustCompile(`\d+`)
	return re.FindAllString(s, -1)
}

// Repeat repeats string n times
func Repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// PadLeft pads string on left to reach length
func PadLeft(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(string(pad), length-len(s))
	return padding + s
}

// PadRight pads string on right to reach length
func PadRight(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(string(pad), length-len(s))
	return s + padding
}

// IsAlpha checks if string contains only letters
func IsAlpha(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsNumeric checks if string contains only digits
func IsNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlphanumeric checks if string is alphanumeric
func IsAlphanumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
