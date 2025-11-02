package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Format represents the output format.
type Format string

const (
	// FormatDefault is the default human-readable format.
	FormatDefault Format = "default"
	// FormatJSON is JSON format.
	FormatJSON Format = "json"
)

// Global output format setting
var globalFormat Format = FormatDefault

// SetFormat sets the global output format.
func SetFormat(format string) error {
	switch format {
	case "default", "":
		globalFormat = FormatDefault
	case "json":
		globalFormat = FormatJSON
	default:
		return fmt.Errorf("invalid output format: %s (valid options: default, json)", format)
	}
	return nil
}

// GetFormat returns the current output format.
func GetFormat() Format {
	return globalFormat
}

// IsJSON returns true if the output format is JSON.
func IsJSON() bool {
	return globalFormat == FormatJSON
}

// PrintJSON prints data as JSON to stdout.
func PrintJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintDefault prints data in default format using a custom formatter function.
func PrintDefault(formatter func()) {
	if globalFormat == FormatDefault {
		formatter()
	}
}

// Print outputs data in the configured format.
// For default format, uses the formatter function.
// For JSON format, marshals the data object.
func Print(data interface{}, formatter func()) error {
	if globalFormat == FormatJSON {
		return PrintJSON(data)
	}
	formatter()
	return nil
}
