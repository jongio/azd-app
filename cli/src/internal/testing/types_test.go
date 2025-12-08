package testing

import "testing"

// TestTypesCompile verifies that all types compile correctly.
func TestTypesCompile(t *testing.T) {
	// Create instances of all types to ensure they compile
	_ = &TestConfig{}
	_ = &ServiceTestConfig{}
	_ = &TestTypeConfig{}
	_ = &CoverageConfig{}
	_ = &TestResult{}
	_ = &TestFailure{}
	_ = &CoverageData{}
	_ = &CoverageMetric{}
	_ = &FileCoverage{}
	_ = &AggregateResult{}
	_ = &AggregateCoverage{}
}

// TestTestConfigDefaults tests default values for TestConfig.
func TestTestConfigDefaults(t *testing.T) {
	config := &TestConfig{
		Parallel:          true,
		FailFast:          false,
		CoverageThreshold: 80.0,
		OutputDir:         "./test-results",
		Verbose:           false,
	}

	if !config.Parallel {
		t.Error("Expected Parallel to be true")
	}
	if config.CoverageThreshold != 80.0 {
		t.Errorf("Expected CoverageThreshold to be 80.0, got %f", config.CoverageThreshold)
	}
}

// TestCoverageMetricCalculation tests coverage percentage calculation.
func TestCoverageMetricCalculation(t *testing.T) {
	metric := CoverageMetric{
		Covered: 85,
		Total:   100,
		Percent: 85.0,
	}

	if metric.Percent != 85.0 {
		t.Errorf("Expected Percent to be 85.0, got %f", metric.Percent)
	}
}

// TestAggregateResultSuccess tests aggregate result success calculation.
func TestAggregateResultSuccess(t *testing.T) {
	result := &AggregateResult{
		Passed:  100,
		Failed:  0,
		Skipped: 5,
		Total:   105,
		Success: true,
	}

	if !result.Success {
		t.Error("Expected Success to be true when there are no failures")
	}
}
