// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/logging"
)

// ReportGenerator generates test reports in various formats.
type ReportGenerator struct {
	format    string
	outputDir string
}

// NewReportGenerator creates a new report generator.
func NewReportGenerator(format, outputDir string) *ReportGenerator {
	return &ReportGenerator{
		format:    format,
		outputDir: outputDir,
	}
}

// GenerateTestReport generates a test report based on the configured format.
func (g *ReportGenerator) GenerateTestReport(results *AggregateResult) error {
	if g.outputDir != "" {
		if err := os.MkdirAll(g.outputDir, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	switch strings.ToLower(g.format) {
	case "json":
		return g.generateJSONReport(results)
	case "junit":
		return g.generateJUnitReport(results)
	case "github":
		return g.generateGitHubReport(results)
	default:
		return nil // default format is console output, handled separately
	}
}

// generateJSONReport generates a JSON test report.
func (g *ReportGenerator) generateJSONReport(results *AggregateResult) error {
	outputPath := filepath.Join(g.outputDir, "test-results.json")

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test results: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	return nil
}

// JUnitTestSuites represents the root element of JUnit XML.
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr,omitempty"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Skipped    int              `xml:"skipped,attr"`
	Time       float64          `xml:"time,attr"`
	Timestamp  string           `xml:"timestamp,attr,omitempty"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite in JUnit XML.
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr,omitempty"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a test case in JUnit XML.
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
	SystemErr string        `xml:"system-err,omitempty"`
}

// JUnitFailure represents a test failure in JUnit XML.
type JUnitFailure struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr,omitempty"`
	Content string `xml:",chardata"`
}

// JUnitError represents a test error in JUnit XML.
type JUnitError struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr,omitempty"`
	Content string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test in JUnit XML.
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// generateJUnitReport generates a JUnit XML test report.
func (g *ReportGenerator) generateJUnitReport(results *AggregateResult) error {
	outputPath := filepath.Join(g.outputDir, "test-results.xml")

	suites := JUnitTestSuites{
		Name:       "azd app test",
		Tests:      results.Total,
		Failures:   results.Failed,
		Errors:     0,
		Skipped:    results.Skipped,
		Time:       results.Duration,
		Timestamp:  time.Now().Format(time.RFC3339),
		TestSuites: make([]JUnitTestSuite, 0, len(results.Services)),
	}

	for _, svcResult := range results.Services {
		suite := JUnitTestSuite{
			Name:      svcResult.Service,
			Tests:     svcResult.Total,
			Failures:  svcResult.Failed,
			Errors:    0,
			Skipped:   svcResult.Skipped,
			Time:      svcResult.Duration,
			Timestamp: time.Now().Format(time.RFC3339),
			TestCases: make([]JUnitTestCase, 0),
		}

		// Add test cases for failures
		for _, failure := range svcResult.Failures {
			testCase := JUnitTestCase{
				Name:      failure.Name,
				ClassName: svcResult.Service,
				Time:      0, // Individual test time not available
				Failure: &JUnitFailure{
					Message: failure.Message,
					Type:    "AssertionError",
					Content: failure.StackTrace,
				},
			}
			suite.TestCases = append(suite.TestCases, testCase)
		}

		// Add placeholder test cases for passed tests
		passedCount := svcResult.Passed
		for i := 0; i < passedCount; i++ {
			var testTime float64
			if svcResult.Total > 0 {
				testTime = svcResult.Duration / float64(svcResult.Total)
			}
			testCase := JUnitTestCase{
				Name:      fmt.Sprintf("test_%d", i+1),
				ClassName: svcResult.Service,
				Time:      testTime,
			}
			suite.TestCases = append(suite.TestCases, testCase)
		}

		// Add placeholder test cases for skipped tests
		skippedCount := svcResult.Skipped
		for i := 0; i < skippedCount; i++ {
			testCase := JUnitTestCase{
				Name:      fmt.Sprintf("skipped_test_%d", i+1),
				ClassName: svcResult.Service,
				Time:      0,
				Skipped:   &JUnitSkipped{Message: "Test skipped"},
			}
			suite.TestCases = append(suite.TestCases, testCase)
		}

		suites.TestSuites = append(suites.TestSuites, suite)
	}

	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JUnit report: %w", err)
	}

	// Add XML header
	xmlData := append([]byte(xml.Header), data...)

	if err := os.WriteFile(outputPath, xmlData, 0o644); err != nil {
		return fmt.Errorf("failed to write JUnit report: %w", err)
	}

	return nil
}

// generateGitHubReport generates GitHub Actions specific output.
func (g *ReportGenerator) generateGitHubReport(results *AggregateResult) error {
	// Output annotations for failed tests
	for _, svcResult := range results.Services {
		for _, failure := range svcResult.Failures {
			// GitHub Actions annotation format:
			// ::error file={name},line={line},endLine={endLine},title={title}::{message}
			if failure.File != "" {
				fmt.Printf("::error file=%s,line=%d,title=Test Failed: %s::%s\n",
					failure.File, failure.Line, failure.Name, failure.Message)
			} else {
				fmt.Printf("::error title=Test Failed: %s [%s]::%s\n",
					failure.Name, svcResult.Service, failure.Message)
			}
		}
	}

	// Generate job summary
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile != "" {
		summary := g.generateGitHubSummary(results)
		if err := os.WriteFile(summaryFile, []byte(summary), 0644); err != nil {
			// Don't fail if we can't write summary
			log := logging.NewLogger("test")
			log.Warn("failed to write GitHub summary", "error", err.Error())
		}
	}

	// Set output variables
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile != "" {
		outputs := []string{
			fmt.Sprintf("tests_total=%d", results.Total),
			fmt.Sprintf("tests_passed=%d", results.Passed),
			fmt.Sprintf("tests_failed=%d", results.Failed),
			fmt.Sprintf("tests_skipped=%d", results.Skipped),
			fmt.Sprintf("tests_success=%t", results.Success),
		}

		// Add coverage if available
		if results.Coverage != nil && results.Coverage.Aggregate != nil {
			outputs = append(outputs,
				fmt.Sprintf("coverage_percent=%.1f", results.Coverage.Aggregate.Lines.Percent))
		}

		f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			for _, output := range outputs {
				_, _ = f.WriteString(output + "\n")
			}
		}
	}

	return nil
}

// generateGitHubSummary generates a markdown summary for GitHub Actions.
func (g *ReportGenerator) generateGitHubSummary(results *AggregateResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("## 🧪 Test Results\n\n")

	// Overall status
	if results.Success {
		sb.WriteString("### ✅ All tests passed!\n\n")
	} else {
		sb.WriteString("### ❌ Some tests failed\n\n")
	}

	// Summary table
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Tests | %d |\n", results.Total))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", results.Passed))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", results.Failed))
	sb.WriteString(fmt.Sprintf("| Skipped | %d |\n", results.Skipped))
	sb.WriteString(fmt.Sprintf("| Duration | %.2fs |\n", results.Duration))
	sb.WriteString("\n")

	// Service breakdown
	sb.WriteString("### Services\n\n")
	sb.WriteString("| Service | Status | Passed | Failed | Coverage |\n")
	sb.WriteString("|---------|--------|--------|--------|----------|\n")

	for _, svcResult := range results.Services {
		status := "✅"
		if !svcResult.Success {
			status = "❌"
		}

		coverageStr := "N/A"
		if results.Coverage != nil {
			if svcCov, ok := results.Coverage.Services[svcResult.Service]; ok {
				coverageStr = fmt.Sprintf("%.1f%%", svcCov.Lines.Percent)
			}
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n",
			svcResult.Service, status, svcResult.Passed, svcResult.Failed, coverageStr))
	}
	sb.WriteString("\n")

	// Coverage summary
	if results.Coverage != nil && results.Coverage.Aggregate != nil {
		sb.WriteString("### 📊 Coverage Summary\n\n")
		sb.WriteString(fmt.Sprintf("**Overall Coverage:** %.1f%%\n\n", results.Coverage.Aggregate.Lines.Percent))

		if results.Coverage.Threshold > 0 {
			if results.Coverage.Met {
				sb.WriteString(fmt.Sprintf("✅ Coverage meets threshold of %.0f%%\n", results.Coverage.Threshold))
			} else {
				sb.WriteString(fmt.Sprintf("❌ Coverage below threshold of %.0f%%\n", results.Coverage.Threshold))
			}
		}
	}

	// Failures section
	hasFailures := false
	for _, svcResult := range results.Services {
		if len(svcResult.Failures) > 0 {
			hasFailures = true
			break
		}
	}

	if hasFailures {
		sb.WriteString("\n### ❌ Failed Tests\n\n")
		for _, svcResult := range results.Services {
			for _, failure := range svcResult.Failures {
				sb.WriteString(fmt.Sprintf("#### %s: %s\n\n", svcResult.Service, failure.Name))
				sb.WriteString(fmt.Sprintf("**Message:** %s\n\n", failure.Message))
				if failure.StackTrace != "" {
					sb.WriteString("```\n")
					sb.WriteString(failure.StackTrace)
					sb.WriteString("\n```\n\n")
				}
			}
		}
	}

	return sb.String()
}
