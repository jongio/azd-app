package testing

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CoverageAggregator collects and merges coverage data from multiple services
type CoverageAggregator struct {
	serviceCoverage map[string]*CoverageData
	threshold       float64
	outputDir       string
}

// NewCoverageAggregator creates a new coverage aggregator
func NewCoverageAggregator(threshold float64, outputDir string) *CoverageAggregator {
	return &CoverageAggregator{
		serviceCoverage: make(map[string]*CoverageData),
		threshold:       threshold,
		outputDir:       outputDir,
	}
}

// AddCoverage adds coverage data for a service
func (a *CoverageAggregator) AddCoverage(service string, data *CoverageData) error {
	if data == nil {
		return fmt.Errorf("coverage data is nil for service %s", service)
	}
	a.serviceCoverage[service] = data
	return nil
}

// Aggregate calculates aggregate coverage metrics across all services
func (a *CoverageAggregator) Aggregate() *AggregateCoverage {
	if len(a.serviceCoverage) == 0 {
		return &AggregateCoverage{
			Services:  make(map[string]*CoverageData),
			Aggregate: &CoverageData{},
			Threshold: a.threshold,
			Met:       false,
		}
	}

	totalLines := 0
	coveredLines := 0
	services := make(map[string]*CoverageData)

	for service, coverage := range a.serviceCoverage {
		totalLines += coverage.Lines.Total
		coveredLines += coverage.Lines.Covered
		services[service] = coverage
	}

	linePercentage := 0.0
	if totalLines > 0 {
		linePercentage = (float64(coveredLines) / float64(totalLines)) * 100.0
	}

	aggregateData := &CoverageData{
		Lines: CoverageMetric{
			Total:   totalLines,
			Covered: coveredLines,
			Percent: linePercentage,
		},
	}

	return &AggregateCoverage{
		Services:  services,
		Aggregate: aggregateData,
		Threshold: a.threshold,
		Met:       linePercentage >= a.threshold,
	}
}

// CheckThreshold checks if aggregate coverage meets the threshold
func (a *CoverageAggregator) CheckThreshold() (bool, float64) {
	aggregate := a.Aggregate()
	return aggregate.Met, aggregate.Aggregate.Lines.Percent
}

// GenerateReport generates coverage reports in various formats
func (a *CoverageAggregator) GenerateReport(format string) error {
	aggregate := a.Aggregate()

	if a.outputDir != "" {
		// Ensure output directory exists
		if err := os.MkdirAll(a.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	switch strings.ToLower(format) {
	case "json":
		return a.generateJSONReport(aggregate)
	case "cobertura", "xml":
		return a.generateCoberturaReport(aggregate)
	case "html":
		return a.generateHTMLReport(aggregate)
	default:
		return fmt.Errorf("unsupported coverage format: %s", format)
	}
}

// generateJSONReport generates a JSON coverage report
func (a *CoverageAggregator) generateJSONReport(aggregate *AggregateCoverage) error {
	outputPath := filepath.Join(a.outputDir, "coverage.json")

	data, err := json.MarshalIndent(aggregate, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal coverage data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	return nil
}

// CoberturaPackage represents a package in Cobertura format
type CoberturaPackage struct {
	XMLName    xml.Name         `xml:"package"`
	Name       string           `xml:"name,attr"`
	LineRate   float64          `xml:"line-rate,attr"`
	BranchRate float64          `xml:"branch-rate,attr"`
	Complexity float64          `xml:"complexity,attr"`
	Classes    []CoberturaClass `xml:"classes>class"`
}

// CoberturaClass represents a class in Cobertura format
type CoberturaClass struct {
	XMLName    xml.Name `xml:"class"`
	Name       string   `xml:"name,attr"`
	Filename   string   `xml:"filename,attr"`
	LineRate   float64  `xml:"line-rate,attr"`
	BranchRate float64  `xml:"branch-rate,attr"`
	Complexity float64  `xml:"complexity,attr"`
}

// CoberturaCoverage represents the root Cobertura XML structure
type CoberturaCoverage struct {
	XMLName    xml.Name           `xml:"coverage"`
	LineRate   float64            `xml:"line-rate,attr"`
	BranchRate float64            `xml:"branch-rate,attr"`
	Version    string             `xml:"version,attr"`
	Timestamp  int64              `xml:"timestamp,attr"`
	Packages   []CoberturaPackage `xml:"packages>package"`
}

// generateCoberturaReport generates a Cobertura XML coverage report
func (a *CoverageAggregator) generateCoberturaReport(aggregate *AggregateCoverage) error {
	outputPath := filepath.Join(a.outputDir, "coverage.xml")

	// Create Cobertura structure
	lineRate := aggregate.Aggregate.Lines.Percent / 100.0
	coverage := CoberturaCoverage{
		LineRate:   lineRate,
		BranchRate: lineRate, // Simplified - same as line rate
		Version:    "1.0",
		Timestamp:  0,
		Packages:   []CoberturaPackage{},
	}

	// Add packages for each service
	for service, serviceCoverage := range aggregate.Services {
		serviceLineRate := serviceCoverage.Lines.Percent / 100.0

		pkg := CoberturaPackage{
			Name:       service,
			LineRate:   serviceLineRate,
			BranchRate: serviceLineRate,
			Complexity: 0,
			Classes:    []CoberturaClass{},
		}

		coverage.Packages = append(coverage.Packages, pkg)
	}

	// Marshal to XML
	data, err := xml.MarshalIndent(coverage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Cobertura data: %w", err)
	}

	// Add XML header
	xmlData := append([]byte(xml.Header), data...)

	if err := os.WriteFile(outputPath, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write Cobertura report: %w", err)
	}

	return nil
}

// generateHTMLReport generates an HTML coverage report
func (a *CoverageAggregator) generateHTMLReport(aggregate *AggregateCoverage) error {
	outputPath := filepath.Join(a.outputDir, "coverage.html")

	linePercent := aggregate.Aggregate.Lines.Percent
	covered := aggregate.Aggregate.Lines.Covered
	total := aggregate.Aggregate.Lines.Total

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Code Coverage Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .summary { background: #f5f5f5; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .metric { font-size: 48px; font-weight: bold; color: %s; }
        .label { color: #666; font-size: 14px; text-transform: uppercase; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th { background: #333; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #ddd; }
        .high { color: #28a745; }
        .medium { color: #ffc107; }
        .low { color: #dc3545; }
    </style>
</head>
<body>
    <h1>Code Coverage Report</h1>
    <div class="summary">
        <div class="label">Overall Coverage</div>
        <div class="metric">%.1f%%</div>
        <div>%d / %d lines covered</div>
    </div>
    <h2>Coverage by Service</h2>
    <table>
        <tr>
            <th>Service</th>
            <th>Lines Covered</th>
            <th>Total Lines</th>
            <th>Coverage</th>
        </tr>
`, getCoverageColor(linePercent), linePercent, covered, total)

	for service, coverage := range aggregate.Services {
		percentage := coverage.Lines.Percent

		html += fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%d</td>
            <td>%d</td>
            <td class="%s">%.1f%%</td>
        </tr>
`, service, coverage.Lines.Covered, coverage.Lines.Total, getCoverageClass(percentage), percentage)
	}

	html += `    </table>
</body>
</html>`

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}

// getCoverageColor returns the color for a coverage percentage
func getCoverageColor(percentage float64) string {
	if percentage >= 80 {
		return "#28a745"
	} else if percentage >= 50 {
		return "#ffc107"
	}
	return "#dc3545"
}

// getCoverageClass returns the CSS class for a coverage percentage
func getCoverageClass(percentage float64) string {
	if percentage >= 80 {
		return "high"
	} else if percentage >= 50 {
		return "medium"
	}
	return "low"
}
