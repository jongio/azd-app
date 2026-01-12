// Package output re-exports cliout from azd-core for backward compatibility.
// New code should import github.com/jongio/azd-core/cliout directly.
// This package exists to support progress tracking (progress.go) which uses cliout.
package output

import "github.com/jongio/azd-core/cliout"

// Re-export all types
type Format = cliout.Format
type TableRow = cliout.TableRow

// Re-export all constants
const (
	FormatDefault = cliout.FormatDefault
	FormatJSON    = cliout.FormatJSON

	// ANSI color codes
	Reset = cliout.Reset
	Bold  = cliout.Bold
	Dim   = cliout.Dim

	Black   = cliout.Black
	Red     = cliout.Red
	Green   = cliout.Green
	Yellow  = cliout.Yellow
	Blue    = cliout.Blue
	Magenta = cliout.Magenta
	Cyan    = cliout.Cyan
	White   = cliout.White
	Gray    = cliout.Gray

	BrightRed     = cliout.BrightRed
	BrightGreen   = cliout.BrightGreen
	BrightYellow  = cliout.BrightYellow
	BrightBlue    = cliout.BrightBlue
	BrightMagenta = cliout.BrightMagenta
	BrightCyan    = cliout.BrightCyan

	// Symbols
	SymbolCheck   = cliout.SymbolCheck
	SymbolCross   = cliout.SymbolCross
	SymbolWarning = cliout.SymbolWarning
	SymbolInfo    = cliout.SymbolInfo
	SymbolArrow   = cliout.SymbolArrow
	SymbolDot     = cliout.SymbolDot

	ASCIICheck   = cliout.ASCIICheck
	ASCIICross   = cliout.ASCIICross
	ASCIIWarning = cliout.ASCIIWarning
	ASCIIInfo    = cliout.ASCIIInfo
	ASCIIArrow   = cliout.ASCIIArrow
	ASCIIDot     = cliout.ASCIIDot
)

// Re-export all variables
var (
	IconSearch  = cliout.IconSearch
	IconTool    = cliout.IconTool
	IconRefresh = cliout.IconRefresh
	IconPackage = cliout.IconPackage
	IconPython  = cliout.IconPython
	IconDotnet  = cliout.IconDotnet
	IconDocker  = cliout.IconDocker
	IconCheck   = cliout.IconCheck
	IconBulb    = cliout.IconBulb
	IconRocket  = cliout.IconRocket
	IconWarning = cliout.IconWarning
	IconError   = cliout.IconError
	IconInfo    = cliout.IconInfo
	IconFolder  = cliout.IconFolder
)

// Re-export all functions

func SetFormat(format string) error    { return cliout.SetFormat(format) }
func GetFormat() Format                { return cliout.GetFormat() }
func IsJSON() bool                     { return cliout.IsJSON() }
func PrintJSON(data interface{}) error { return cliout.PrintJSON(data) }
func PrintDefault(formatter func())    { cliout.PrintDefault(formatter) }
func Print(data interface{}, formatter func()) error {
	return cliout.Print(data, formatter)
}

func SetOrchestrated(value bool)                 { cliout.SetOrchestrated(value) }
func IsOrchestrated() bool                       { return cliout.IsOrchestrated() }
func Header(text string)                         { cliout.Header(text) }
func CommandHeader(cmd, desc string)             { cliout.CommandHeader(cmd, desc) }
func Section(icon, text string)                  { cliout.Section(icon, text) }
func Success(format string, args ...interface{}) { cliout.Success(format, args...) }
func Error(format string, args ...interface{})   { cliout.Error(format, args...) }
func Warning(format string, args ...interface{}) { cliout.Warning(format, args...) }
func Info(format string, args ...interface{})    { cliout.Info(format, args...) }
func Step(icon, format string, args ...interface{}) {
	cliout.Step(icon, format, args...)
}
func Item(format string, args ...interface{})        { cliout.Item(format, args...) }
func Bullet(format string, args ...interface{})      { cliout.Bullet(format, args...) }
func ItemSuccess(format string, args ...interface{}) { cliout.ItemSuccess(format, args...) }
func ItemError(format string, args ...interface{})   { cliout.ItemError(format, args...) }
func ItemWarning(format string, args ...interface{}) { cliout.ItemWarning(format, args...) }
func ItemInfo(format string, args ...interface{})    { cliout.ItemInfo(format, args...) }
func Divider()                                       { cliout.Divider() }
func Newline()                                       { cliout.Newline() }
func Hint(hints ...string)                           { cliout.Hint(hints...) }
func Phase(label string)                             { cliout.Phase(label) }
func Plain(format string, args ...interface{})       { cliout.Plain(format, args...) }
func Confirm(message string) bool                    { return cliout.Confirm(message) }
func Label(label, value string)                      { cliout.Label(label, value) }
func LabelColored(label, value, color string)        { cliout.LabelColored(label, value, color) }

func Highlight(format string, args ...interface{}) string {
	return cliout.Highlight(format, args...)
}
func Emphasize(format string, args ...interface{}) string {
	return cliout.Emphasize(format, args...)
}
func Muted(format string, args ...interface{}) string { return cliout.Muted(format, args...) }
func URL(url string) string                           { return cliout.URL(url) }
func Count(n int) string                              { return cliout.Count(n) }
func Status(status string) string                     { return cliout.Status(status) }
func ProgressBar(current, total int, width int) string {
	return cliout.ProgressBar(current, total, width)
}
func Table(headers []string, rows []TableRow) { cliout.Table(headers, rows) }
