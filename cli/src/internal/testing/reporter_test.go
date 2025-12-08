package testing

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewReportGenerator(t *testing.T) {
	gen := NewReportGenerator("json", "/output")

	if gen.format != "json" {
		t.Errorf("Expected format 'json', got '%s'", gen.format)
	}
	if gen.outputDir != "/output" {
		t.Errorf("Expected outputDir '/output', got '%s'", gen.outputDir)
	}
}

func TestReportGenerator_generateJSONReport(t *testing.T) {
	tempDir := t.TempDir()
	gen := NewReportGenerator("json", tempDir)

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "api",
				Passed:  5,
				Failed:  1,
				Total:   6,
				Success: false,
			},
		},
		Passed:   5,
		Failed:   1,
		Total:    6,
		Duration: 2.5,
		Success:  false,
	}

	err := gen.generateJSONReport(results)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(tempDir, "test-results.json")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	// Verify JSON is valid
	var parsed AggregateResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}

	if parsed.Total != 6 {
		t.Errorf("Expected total 6, got %d", parsed.Total)
	}
	if parsed.Passed != 5 {
		t.Errorf("Expected passed 5, got %d", parsed.Passed)
	}
}

func TestReportGenerator_generateJUnitReport(t *testing.T) {
	tempDir := t.TempDir()
	gen := NewReportGenerator("junit", tempDir)

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "api",
				Passed:  4,
				Failed:  1,
				Skipped: 1,
				Total:   6,
				Success: false,
				Failures: []TestFailure{
					{
						Name:       "TestFailing",
						Message:    "Expected 5, got 4",
						StackTrace: "at TestFailing (test.go:15)",
						File:       "test.go",
						Line:       15,
					},
				},
			},
		},
		Passed:   4,
		Failed:   1,
		Skipped:  1,
		Total:    6,
		Duration: 2.5,
		Success:  false,
	}

	err := gen.generateJUnitReport(results)
	if err != nil {
		t.Fatalf("Failed to generate JUnit report: %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(tempDir, "test-results.xml")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read JUnit report: %v", err)
	}

	// Verify XML is valid
	var parsed JUnitTestSuites
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JUnit report: %v", err)
	}

	if parsed.Tests != 6 {
		t.Errorf("Expected tests 6, got %d", parsed.Tests)
	}
	if parsed.Failures != 1 {
		t.Errorf("Expected failures 1, got %d", parsed.Failures)
	}
	if parsed.Skipped != 1 {
		t.Errorf("Expected skipped 1, got %d", parsed.Skipped)
	}
	if len(parsed.TestSuites) != 1 {
		t.Errorf("Expected 1 test suite, got %d", len(parsed.TestSuites))
	}
	if parsed.TestSuites[0].Name != "api" {
		t.Errorf("Expected suite name 'api', got '%s'", parsed.TestSuites[0].Name)
	}
}

func TestReportGenerator_generateJUnitReport_MultipleServices(t *testing.T) {
	tempDir := t.TempDir()
	gen := NewReportGenerator("junit", tempDir)

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "api",
				Passed:  5,
				Total:   5,
				Success: true,
			},
			{
				Service: "web",
				Passed:  3,
				Failed:  2,
				Total:   5,
				Success: false,
				Failures: []TestFailure{
					{Name: "TestA", Message: "Error A"},
					{Name: "TestB", Message: "Error B"},
				},
			},
		},
		Passed:  8,
		Failed:  2,
		Total:   10,
		Success: false,
	}

	err := gen.generateJUnitReport(results)
	if err != nil {
		t.Fatalf("Failed to generate JUnit report: %v", err)
	}

	outputPath := filepath.Join(tempDir, "test-results.xml")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read JUnit report: %v", err)
	}

	var parsed JUnitTestSuites
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JUnit report: %v", err)
	}

	if len(parsed.TestSuites) != 2 {
		t.Errorf("Expected 2 test suites, got %d", len(parsed.TestSuites))
	}
}

func TestReportGenerator_generateGitHubSummary(t *testing.T) {
	gen := NewReportGenerator("github", "")

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "api",
				Passed:  5,
				Failed:  0,
				Total:   5,
				Success: true,
			},
			{
				Service: "web",
				Passed:  3,
				Failed:  2,
				Total:   5,
				Success: false,
				Failures: []TestFailure{
					{
						Name:       "TestFailing",
						Message:    "Expected 5, got 4",
						StackTrace: "at line 15",
					},
				},
			},
		},
		Passed:   8,
		Failed:   2,
		Total:    10,
		Duration: 5.5,
		Success:  false,
		Coverage: &AggregateCoverage{
			Services: map[string]*CoverageData{
				"api": {Lines: CoverageMetric{Percent: 85.0}},
				"web": {Lines: CoverageMetric{Percent: 72.5}},
			},
			Aggregate: &CoverageData{
				Lines: CoverageMetric{Percent: 78.75},
			},
			Threshold: 80.0,
			Met:       false,
		},
	}

	summary := gen.generateGitHubSummary(results)

	// Check for key content
	if !strings.Contains(summary, "## 🧪 Test Results") {
		t.Error("Summary missing header")
	}
	if !strings.Contains(summary, "Some tests failed") {
		t.Error("Summary missing failure indicator")
	}
	if !strings.Contains(summary, "| api |") {
		t.Error("Summary missing api service row")
	}
	if !strings.Contains(summary, "| web |") {
		t.Error("Summary missing web service row")
	}
	if !strings.Contains(summary, "78.8%") { // formatted as %.1f
		t.Error("Summary missing overall coverage")
	}
	if !strings.Contains(summary, "TestFailing") {
		t.Error("Summary missing failed test details")
	}
}

func TestReportGenerator_generateGitHubSummary_AllPassed(t *testing.T) {
	gen := NewReportGenerator("github", "")

	results := &AggregateResult{
		Services: []*TestResult{
			{
				Service: "api",
				Passed:  5,
				Total:   5,
				Success: true,
			},
		},
		Passed:  5,
		Total:   5,
		Success: true,
	}

	summary := gen.generateGitHubSummary(results)

	if !strings.Contains(summary, "All tests passed") {
		t.Error("Summary missing success indicator")
	}
}

func TestReportGenerator_GenerateTestReport_DefaultFormat(t *testing.T) {
	tempDir := t.TempDir()
	gen := NewReportGenerator("default", tempDir)

	results := &AggregateResult{
		Passed:  5,
		Total:   5,
		Success: true,
	}

	err := gen.GenerateTestReport(results)
	if err != nil {
		t.Fatalf("Default format should not return error: %v", err)
	}

	// No file should be created for default format
	files, _ := os.ReadDir(tempDir)
	if len(files) > 0 {
		t.Error("Default format should not create files")
	}
}

func TestReportGenerator_GenerateTestReport_CreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "nested", "output")
	gen := NewReportGenerator("json", outputDir)

	results := &AggregateResult{
		Passed:  5,
		Total:   5,
		Success: true,
	}

	err := gen.GenerateTestReport(results)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}
}

func TestJUnitStructures(t *testing.T) {
	// Test marshaling JUnit structures
	suites := JUnitTestSuites{
		Name:  "test",
		Tests: 1,
		Time:  1.0,
		TestSuites: []JUnitTestSuite{
			{
				Name:  "suite1",
				Tests: 1,
				TestCases: []JUnitTestCase{
					{
						Name:      "test1",
						ClassName: "TestClass",
						Time:      0.5,
						Failure: &JUnitFailure{
							Message: "failure message",
							Type:    "AssertionError",
							Content: "stack trace",
						},
					},
				},
			},
		},
	}

	data, err := xml.Marshal(suites)
	if err != nil {
		t.Fatalf("Failed to marshal JUnit structure: %v", err)
	}

	if !strings.Contains(string(data), "AssertionError") {
		t.Error("JUnit XML missing failure type")
	}
	if !strings.Contains(string(data), "failure message") {
		t.Error("JUnit XML missing failure message")
	}
}

func TestJUnitSkippedTest(t *testing.T) {
	testCase := JUnitTestCase{
		Name:      "skipped_test",
		ClassName: "TestClass",
		Skipped:   &JUnitSkipped{Message: "Skipped for reason"},
	}

	data, err := xml.Marshal(testCase)
	if err != nil {
		t.Fatalf("Failed to marshal skipped test: %v", err)
	}

	if !strings.Contains(string(data), "skipped") {
		t.Error("XML missing skipped element")
	}
	if !strings.Contains(string(data), "Skipped for reason") {
		t.Error("XML missing skip message")
	}
}

func TestJUnitErrorTest(t *testing.T) {
	testCase := JUnitTestCase{
		Name:      "error_test",
		ClassName: "TestClass",
		Error: &JUnitError{
			Message: "error message",
			Type:    "RuntimeError",
			Content: "error stack",
		},
	}

	data, err := xml.Marshal(testCase)
	if err != nil {
		t.Fatalf("Failed to marshal error test: %v", err)
	}

	if !strings.Contains(string(data), "RuntimeError") {
		t.Error("XML missing error type")
	}
}
