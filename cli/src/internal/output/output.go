package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Format represents the output format.
type Format string

const (
	// FormatDefault is the default human-readable format.
	FormatDefault Format = "default"
	// FormatJSON is JSON format.
	FormatJSON Format = "json"
)

// ANSI color codes for consistent styling
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Bright foreground colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
)

// Unicode symbols for modern CLI output
const (
	SymbolCheck   = "✓"
	SymbolCross   = "✗"
	SymbolWarning = "⚠"
	SymbolInfo    = "ℹ"
	SymbolArrow   = "→"
	SymbolDot     = "•"
	SymbolSpinner = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏" // Spinner frames
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

// Modern CLI output functions with consistent styling

// Header prints a bold header with a divider
func Header(text string) {
	fmt.Printf("\n%s%s%s\n", Bold, text, Reset)
	fmt.Println(strings.Repeat("=", len(text)))
}

// Section prints a section header
func Section(icon, text string) {
	fmt.Printf("\n%s%s %s%s\n", Cyan, icon, text, Reset)
}

// Success prints a success message with green checkmark
func Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s %s\n", BrightGreen, SymbolCheck, Reset, msg)
}

// Error prints an error message with red X
func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s %s\n", BrightRed, SymbolCross, Reset, msg)
}

// Warning prints a warning message with yellow triangle
func Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s  %s\n", BrightYellow, SymbolWarning, Reset, msg)
}

// Info prints an info message with blue info icon
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s  %s\n", BrightBlue, SymbolInfo, Reset, msg)
}

// Step prints a step message with an icon
func Step(icon, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s %s\n", Cyan, icon, Reset, msg)
}

// Item prints an indented item
func Item(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("   %s\n", msg)
}

// ItemSuccess prints an indented success item
func ItemSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("   %s%s%s %s\n", Green, SymbolCheck, Reset, msg)
}

// ItemError prints an indented error item
func ItemError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("   %s%s%s %s\n", Red, SymbolCross, Reset, msg)
}

// ItemWarning prints an indented warning item
func ItemWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("   %s%s%s  %s\n", Yellow, SymbolWarning, Reset, msg)
}

// ItemInfo prints an indented info item
func ItemInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("   %s%s%s  %s\n", Cyan, SymbolInfo, Reset, msg)
}

// Divider prints a horizontal divider
func Divider() {
	fmt.Printf("\n%s%s%s\n", Dim, strings.Repeat("─", 75), Reset)
}

// Newline prints a blank line
func Newline() {
	fmt.Println()
}

// Label prints a label and value pair
func Label(label, value string) {
	fmt.Printf("   %s%-12s%s %s\n", Dim, label+":", Reset, value)
}

// LabelColored prints a label and colored value pair
func LabelColored(label, value, color string) {
	fmt.Printf("   %s%-12s%s %s%s%s\n", Dim, label+":", Reset, color, value, Reset)
}

// Highlight prints highlighted text
func Highlight(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return Bold + Cyan + msg + Reset
}

// Emphasize prints emphasized text
func Emphasize(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return Bold + msg + Reset
}

// Muted prints muted/dim text
func Muted(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return Dim + msg + Reset
}

// URL prints a URL in bright blue
func URL(url string) string {
	return BrightBlue + url + Reset
}

// Count prints a count badge
func Count(n int) string {
	return Bold + fmt.Sprintf("%d", n) + Reset
}

// Status prints a status badge with appropriate color
func Status(status string) string {
	switch strings.ToLower(status) {
	case "success", "ok", "running", "healthy":
		return BrightGreen + status + Reset
	case "warning", "pending", "starting":
		return BrightYellow + status + Reset
	case "error", "failed", "unhealthy":
		return BrightRed + status + Reset
	case "info", "unknown":
		return BrightBlue + status + Reset
	default:
		return status
	}
}

// ProgressBar prints a simple progress bar
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %d%%", bar, int(percent*100))
}

// Table prints a simple table
type TableRow map[string]string

func Table(headers []string, rows []TableRow) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make(map[string]int)
	for _, header := range headers {
		widths[header] = len(header)
	}
	for _, row := range rows {
		for _, header := range headers {
			if len(row[header]) > widths[header] {
				widths[header] = len(row[header])
			}
		}
	}

	// Print header
	fmt.Print("   ")
	for _, header := range headers {
		fmt.Printf("%s%-*s%s  ", Bold, widths[header], header, Reset)
	}
	fmt.Println()

	// Print separator
	fmt.Print("   ")
	for _, header := range headers {
		fmt.Print(strings.Repeat("─", widths[header]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		fmt.Print("   ")
		for _, header := range headers {
			fmt.Printf("%-*s  ", widths[header], row[header])
		}
		fmt.Println()
	}
}
