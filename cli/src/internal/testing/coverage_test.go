package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageAggregator(t *testing.T) {
	// Create temp directory for test output
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	// Add coverage for multiple services
	err = agg.AddCoverage("web", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	err = agg.AddCoverage("api", &CoverageData{
		Lines: CoverageMetric{
			Total:   200,
			Covered: 180,
			Percent: 90.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	// Test aggregation
	aggregate := agg.Aggregate()
	if aggregate.Aggregate.Lines.Total != 300 {
		t.Errorf("Expected total lines 300, got %d", aggregate.Aggregate.Lines.Total)
	}
	if aggregate.Aggregate.Lines.Covered != 265 {
		t.Errorf("Expected covered lines 265, got %d", aggregate.Aggregate.Lines.Covered)
	}
	expectedPercentage := (265.0 / 300.0) * 100.0
	actualPercentage := aggregate.Aggregate.Lines.Percent
	if actualPercentage < expectedPercentage-0.1 || actualPercentage > expectedPercentage+0.1 {
		t.Errorf("Expected line percentage %.2f, got %.2f", expectedPercentage, actualPercentage)
	}
}

func TestCheckThreshold(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	// Add coverage below threshold
	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 75,
			Percent: 75.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	meetsThreshold, percentage := agg.CheckThreshold()
	if meetsThreshold {
		t.Errorf("Expected threshold check to fail, but it passed with %.2f%%", percentage)
	}

	// Add more coverage to meet threshold
	err = agg.AddCoverage("service2", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 90,
			Percent: 90.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	meetsThreshold, percentage = agg.CheckThreshold()
	if !meetsThreshold {
		t.Errorf("Expected threshold check to pass, but it failed with %.2f%%", percentage)
	}
}

func TestGenerateJSONReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("json")
	if err != nil {
		t.Errorf("Failed to generate JSON report: %v", err)
	}

	// Check if file was created
	reportPath := filepath.Join(tmpDir, "coverage.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("JSON report was not created at %s", reportPath)
	}
}

func TestGenerateCoberturaReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("cobertura")
	if err != nil {
		t.Errorf("Failed to generate Cobertura report: %v", err)
	}

	// Check if file was created
	reportPath := filepath.Join(tmpDir, "coverage.xml")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Cobertura report was not created at %s", reportPath)
	}
}

func TestGenerateHTMLReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Errorf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("html")
	if err != nil {
		t.Errorf("Failed to generate HTML report: %v", err)
	}

	// Check if file was created
	reportPath := filepath.Join(tmpDir, "coverage.html")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("HTML report was not created at %s", reportPath)
	}
}

func TestAddNilCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", nil)
	if err == nil {
		t.Error("Expected error when adding nil coverage, but got nil")
	}
}

func TestSetSourceRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)
	agg.SetSourceRoot("/path/to/source")

	if agg.sourceRoot != "/path/to/source" {
		t.Errorf("Expected sourceRoot '/path/to/source', got '%s'", agg.sourceRoot)
	}
}

func TestGenerateAllReports(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateAllReports()
	if err != nil {
		t.Errorf("GenerateAllReports failed: %v", err)
	}

	// Verify all report files were created
	expectedFiles := []string{"coverage.json", "coverage.xml", "coverage.html"}
	for _, file := range expectedFiles {
		reportPath := filepath.Join(tmpDir, file)
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			t.Errorf("Expected %s to be created", file)
		}
	}
}

func TestGenerateReportUnsupportedFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("unknown")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestGenerateFileHTML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a source file
	sourceDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "main.go")
	sourceContent := `package main

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	agg := NewCoverageAggregator(80.0, tmpDir)
	agg.SetSourceRoot(tmpDir)

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   3,
			Covered: 2,
			Percent: 66.7,
		},
		Files: []*FileCoverage{
			{
				Path: "src/main.go",
				Lines: CoverageMetric{
					Total:   3,
					Covered: 2,
					Percent: 66.7,
				},
				LineHits: map[int]int{
					3: 1,
					4: 1,
					5: 0,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("html")
	if err != nil {
		t.Errorf("GenerateReport(html) failed: %v", err)
	}

	// Check if file-specific HTML was created
	fileHTMLPath := filepath.Join(tmpDir, "file-src-main.go.html")
	if _, err := os.Stat(fileHTMLPath); os.IsNotExist(err) {
		t.Errorf("Expected file HTML to be created at %s", fileHTMLPath)
	}
}

func TestGetCoverageColorFunctions(t *testing.T) {
	tests := []struct {
		name           string
		percentage     float64
		expectedColor  string
		expectedClass  string
		expectedPClass string
	}{
		{
			name:           "high coverage",
			percentage:     90.0,
			expectedColor:  "#059669",
			expectedClass:  "high",
			expectedPClass: "progress-high",
		},
		{
			name:           "medium coverage",
			percentage:     60.0,
			expectedColor:  "#d97706",
			expectedClass:  "medium",
			expectedPClass: "progress-medium",
		},
		{
			name:           "low coverage",
			percentage:     30.0,
			expectedColor:  "#dc2626",
			expectedClass:  "low",
			expectedPClass: "progress-low",
		},
		{
			name:           "edge case - exactly 80%",
			percentage:     80.0,
			expectedColor:  "#059669",
			expectedClass:  "high",
			expectedPClass: "progress-high",
		},
		{
			name:           "edge case - exactly 50%",
			percentage:     50.0,
			expectedColor:  "#d97706",
			expectedClass:  "medium",
			expectedPClass: "progress-medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := getCoverageColor(tt.percentage)
			if color != tt.expectedColor {
				t.Errorf("getCoverageColor(%.1f) = %s, want %s", tt.percentage, color, tt.expectedColor)
			}

			class := getCoverageClass(tt.percentage)
			if class != tt.expectedClass {
				t.Errorf("getCoverageClass(%.1f) = %s, want %s", tt.percentage, class, tt.expectedClass)
			}

			pClass := getProgressClass(tt.percentage)
			if pClass != tt.expectedPClass {
				t.Errorf("getProgressClass(%.1f) = %s, want %s", tt.percentage, pClass, tt.expectedPClass)
			}
		})
	}
}

func TestGetThresholdFunctions(t *testing.T) {
	tests := []struct {
		name            string
		met             bool
		expectedClass   string
		expectedMessage string
	}{
		{
			name:            "threshold met",
			met:             true,
			expectedClass:   "threshold-met",
			expectedMessage: "✓ Threshold met",
		},
		{
			name:            "threshold not met",
			met:             false,
			expectedClass:   "threshold-unmet",
			expectedMessage: "✗ Below threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := getThresholdClass(tt.met)
			if class != tt.expectedClass {
				t.Errorf("getThresholdClass(%v) = %s, want %s", tt.met, class, tt.expectedClass)
			}

			message := getThresholdMessage(tt.met)
			if message != tt.expectedMessage {
				t.Errorf("getThresholdMessage(%v) = %s, want %s", tt.met, message, tt.expectedMessage)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "src/main.go",
			expected: "src-main.go",
		},
		{
			name:     "windows path",
			input:    "src\\main.go",
			expected: "src-main.go",
		},
		{
			name:     "path with drive letter",
			input:    "C:/src/main.go",
			expected: "C--src-main.go",
		},
		{
			name:     "path with spaces",
			input:    "src/my file.go",
			expected: "src-my-file.go",
		},
		{
			name:     "leading slash",
			input:    "/src/main.go",
			expected: "src-main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAggregateWithFileCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	// Add coverage with file-level data
	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
		Files: []*FileCoverage{
			{
				Path: "main.go",
				Lines: CoverageMetric{
					Total:   50,
					Covered: 40,
					Percent: 80.0,
				},
				LineHits: map[int]int{
					1: 1, 2: 1, 3: 0,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	// Add coverage for another service with overlapping files
	err = agg.AddCoverage("service2", &CoverageData{
		Lines: CoverageMetric{
			Total:   50,
			Covered: 45,
			Percent: 90.0,
		},
		Files: []*FileCoverage{
			{
				Path: "main.go",
				Lines: CoverageMetric{
					Total:   50,
					Covered: 45,
					Percent: 90.0,
				},
				LineHits: map[int]int{
					1: 2, 2: 2, 3: 1,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	aggregate := agg.Aggregate()

	// Verify aggregate metrics
	if len(aggregate.Aggregate.Files) != 1 {
		t.Errorf("Expected 1 merged file, got %d", len(aggregate.Aggregate.Files))
	}

	// Verify file coverage was merged
	for _, file := range aggregate.Aggregate.Files {
		if file.Path == "main.go" {
			// Line hits should be combined
			if file.LineHits[1] != 3 {
				t.Errorf("Expected line 1 hits to be 3, got %d", file.LineHits[1])
			}
			if file.LineHits[3] != 1 {
				t.Errorf("Expected line 3 hits to be 1, got %d", file.LineHits[3])
			}
		}
	}
}

func TestCoberturaReportWithSourceRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)
	agg.SetSourceRoot("/path/to/src")

	err = agg.AddCoverage("service1", &CoverageData{
		Lines: CoverageMetric{
			Total:   100,
			Covered: 85,
			Percent: 85.0,
		},
		Files: []*FileCoverage{
			{
				Path: "main.go",
				Lines: CoverageMetric{
					Total:   100,
					Covered: 85,
					Percent: 85.0,
				},
				LineHits: map[int]int{
					10: 5,
					11: 0,
					12: 3,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add coverage: %v", err)
	}

	err = agg.GenerateReport("cobertura")
	if err != nil {
		t.Errorf("GenerateReport(cobertura) failed: %v", err)
	}

	// Read and verify the XML contains source root
	xmlPath := filepath.Join(tmpDir, "coverage.xml")
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("Failed to read coverage.xml: %v", err)
	}

	content := string(data)
	if !containsSubstr(content, "/path/to/src") {
		t.Error("Expected Cobertura XML to contain source root")
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstr(s[1:], substr)))
}

func TestAggregateWithNoCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agg := NewCoverageAggregator(80.0, tmpDir)

	aggregate := agg.Aggregate()
	if aggregate.Aggregate.Lines.Total != 0 {
		t.Errorf("Expected total lines 0, got %d", aggregate.Aggregate.Lines.Total)
	}
	if aggregate.Aggregate.Lines.Covered != 0 {
		t.Errorf("Expected covered lines 0, got %d", aggregate.Aggregate.Lines.Covered)
	}
	if aggregate.Aggregate.Lines.Percent != 0.0 {
		t.Errorf("Expected line percentage 0.0, got %.2f", aggregate.Aggregate.Lines.Percent)
	}
}
