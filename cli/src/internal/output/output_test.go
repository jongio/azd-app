package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSetFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "default format",
			format:  "default",
			wantErr: false,
		},
		{
			name:    "json format",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "empty format (defaults to default)",
			format:  "",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFormat(t *testing.T) {
	// Set to default
	_ = SetFormat("default")
	if got := GetFormat(); got != FormatDefault {
		t.Errorf("GetFormat() = %v, want %v", got, FormatDefault)
	}

	// Set to JSON
	_ = SetFormat("json")
	if got := GetFormat(); got != FormatJSON {
		t.Errorf("GetFormat() = %v, want %v", got, FormatJSON)
	}

	// Reset to default for other tests
	_ = SetFormat("default")
}

func TestIsJSON(t *testing.T) {
	// Set to default
	_ = SetFormat("default")
	if IsJSON() {
		t.Errorf("IsJSON() = true, want false when format is default")
	}

	// Set to JSON
	_ = SetFormat("json")
	if !IsJSON() {
		t.Errorf("IsJSON() = false, want true when format is json")
	}

	// Reset to default for other tests
	_ = SetFormat("default")
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]interface{}{
		"name":    "test",
		"version": "1.0.0",
	}

	err := PrintJSON(data)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("PrintJSON() error = %v, want nil", err)
	}

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("PrintJSON() output is not valid JSON: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("PrintJSON() name = %v, want test", result["name"])
	}
}

func TestPrintDefault(t *testing.T) {
	// Set to default format
	_ = SetFormat("default")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	called := false
	PrintDefault(func() {
		called = true
		os.Stdout.WriteString("test output")
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if !called {
		t.Errorf("PrintDefault() formatter not called in default mode")
	}

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "test output") {
		t.Errorf("PrintDefault() output = %q, want to contain 'test output'", output)
	}
}

func TestPrintDefaultInJSONMode(t *testing.T) {
	// Set to JSON format
	_ = SetFormat("json")
	defer func() { _ = SetFormat("default") }() // Reset

	called := false
	PrintDefault(func() {
		called = true
	})

	if called {
		t.Errorf("PrintDefault() formatter called in JSON mode, should be skipped")
	}
}

func TestPrint(t *testing.T) {
	// Test default format
	_ = SetFormat("default")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	called := false
	err := Print(map[string]string{"test": "value"}, func() {
		called = true
		os.Stdout.WriteString("formatted output")
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Print() error = %v, want nil", err)
	}

	if !called {
		t.Errorf("Print() formatter not called in default mode")
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Reset for next test
	_ = SetFormat("default")
}

func TestPrintJSONMode(t *testing.T) {
	// Set to JSON format
	_ = SetFormat("json")
	defer func() { _ = SetFormat("default") }() // Reset

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	called := false
	err := Print(map[string]string{"test": "value"}, func() {
		called = true
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Print() error = %v, want nil", err)
	}

	if called {
		t.Errorf("Print() formatter called in JSON mode, should use PrintJSON")
	}

	// Read and verify JSON output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Print() in JSON mode output is not valid JSON: %v", err)
	}
}

func TestOutputFunctions(t *testing.T) {
	// Test that output functions don't panic
	// We can't easily test the actual output without capturing stdout

	tests := []struct {
		name string
		fn   func()
	}{
		{"Header", func() { Header("Test Header") }},
		{"Section", func() { Section("📦", "Test Section") }},
		{"Success", func() { Success("Test success") }},
		{"Error", func() { Error("Test error") }},
		{"Warning", func() { Warning("Test warning") }},
		{"Info", func() { Info("Test info") }},
		{"Step", func() { Step("🔧", "Test step") }},
		{"Item", func() { Item("Test item") }},
		{"ItemSuccess", func() { ItemSuccess("Test success item") }},
		{"ItemError", func() { ItemError("Test error item") }},
		{"ItemWarning", func() { ItemWarning("Test warning item") }},
		{"Divider", func() { Divider() }},
		{"Newline", func() { Newline() }},
		{"Label", func() { Label("Name", "Value") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s() panicked: %v", tt.name, r)
				}
			}()

			// Capture stdout to prevent output during tests
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			tt.fn()
		})
	}
}

func TestHighlight(t *testing.T) {
	result := Highlight("test")
	if !strings.Contains(result, "test") {
		t.Errorf("Highlight() = %q, want to contain 'test'", result)
	}

	// Test with format args
	result = Highlight("test %s", "value")
	if !strings.Contains(result, "test value") {
		t.Errorf("Highlight() = %q, want to contain 'test value'", result)
	}
}

func TestEmphasize(t *testing.T) {
	result := Emphasize("test")
	if !strings.Contains(result, "test") {
		t.Errorf("Emphasize() = %q, want to contain 'test'", result)
	}

	// Test with format args
	result = Emphasize("test %d", 123)
	if !strings.Contains(result, "test 123") {
		t.Errorf("Emphasize() = %q, want to contain 'test 123'", result)
	}
}

func TestMuted(t *testing.T) {
	result := Muted("test")
	if !strings.Contains(result, "test") {
		t.Errorf("Muted() = %q, want to contain 'test'", result)
	}

	// Test with format args
	result = Muted("test %s", "value")
	if !strings.Contains(result, "test value") {
		t.Errorf("Muted() = %q, want to contain 'test value'", result)
	}
}

func TestURL(t *testing.T) {
	url := "https://example.com"
	result := URL(url)
	if !strings.Contains(result, url) {
		t.Errorf("URL() = %q, want to contain %q", result, url)
	}
}

func TestCount(t *testing.T) {
	result := Count(42)
	if !strings.Contains(result, "42") {
		t.Errorf("Count() = %q, want to contain '42'", result)
	}
}

func TestSuccessWithArgs(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Success("Test %s %d", "message", 123)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Test message 123") {
		t.Errorf("Success() output = %q, want to contain 'Test message 123'", output)
	}
}

func TestErrorWithArgs(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Error("Error %s %d", "message", 456)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Error message 456") {
		t.Errorf("Error() output = %q, want to contain 'Error message 456'", output)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"success", "success"},
		{"running", "running"},
		{"error", "error"},
		{"warning", "warning"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := Status(tt.status)
			if !strings.Contains(result, tt.want) {
				t.Errorf("Status(%s) = %q, want to contain %q", tt.status, result, tt.want)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		width   int
		wantLen int
	}{
		{"empty", 0, 10, 10, 15}, // "[░░░░░░░░░░] 0%" (length 15)
		{"half", 5, 10, 10, 18}, // "[█████░░░░░] 50%"
		{"full", 10, 10, 10, 19}, // "[██████████] 100%"
		{"zero total", 5, 0, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProgressBar(tt.current, tt.total, tt.width)
			if tt.total == 0 {
				if result != "" {
					t.Errorf("ProgressBar() with zero total = %q, want empty string", result)
				}
			} else if len(result) == 0 && tt.wantLen > 0 {
				t.Errorf("ProgressBar() = empty, want non-empty")
			}
		})
	}
}

func TestTable(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	headers := []string{"Name", "Status"}
	rows := []TableRow{
		{"Name": "Service1", "Status": "running"},
		{"Name": "Service2", "Status": "stopped"},
	}

	Table(headers, rows)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	// Check if table contains expected content
	if !strings.Contains(output, "Name") {
		t.Errorf("Table() output = %q, want to contain 'Name'", output)
	}
	if !strings.Contains(output, "Service1") {
		t.Errorf("Table() output = %q, want to contain 'Service1'", output)
	}
}

func TestTableEmpty(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	headers := []string{"Name", "Status"}
	var rows []TableRow

	Table(headers, rows)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	// Empty table should produce no output
	if output != "" {
		t.Errorf("Table() with empty rows = %q, want empty output", output)
	}
}

func TestLabelColored(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	LabelColored("Status", "running", Green)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Status") {
		t.Errorf("LabelColored() output = %q, want to contain 'Status'", output)
	}
	if !strings.Contains(output, "running") {
		t.Errorf("LabelColored() output = %q, want to contain 'running'", output)
	}
}

func TestItemInfo(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ItemInfo("Test info item")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Test info item") {
		t.Errorf("ItemInfo() output = %q, want to contain 'Test info item'", output)
	}
}
