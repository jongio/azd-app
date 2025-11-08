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
