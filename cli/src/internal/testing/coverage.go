package testing

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// CoverageAggregator collects and merges coverage data from multiple services
type CoverageAggregator struct {
	serviceCoverage map[string]*CoverageData
	threshold       float64
	outputDir       string
	sourceRoot      string // Root path for source file linking
}

// NewCoverageAggregator creates a new coverage aggregator
func NewCoverageAggregator(threshold float64, outputDir string) *CoverageAggregator {
	return &CoverageAggregator{
		serviceCoverage: make(map[string]*CoverageData),
		threshold:       threshold,
		outputDir:       outputDir,
	}
}

// SetSourceRoot sets the root path for source file linking in reports
func (a *CoverageAggregator) SetSourceRoot(root string) {
	a.sourceRoot = root
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

	// Merge file coverage across all services
	allFiles := make(map[string]*FileCoverage)

	for service, coverage := range a.serviceCoverage {
		totalLines += coverage.Lines.Total
		coveredLines += coverage.Lines.Covered
		services[service] = coverage

		// Merge file-level coverage
		for _, file := range coverage.Files {
			if existing, ok := allFiles[file.Path]; ok {
				// Merge line coverage
				existing.Lines.Total += file.Lines.Total
				existing.Lines.Covered += file.Lines.Covered
				for lineNum, hits := range file.LineHits {
					existing.LineHits[lineNum] += hits
				}
			} else {
				// Copy file coverage
				fileCopy := &FileCoverage{
					Path:     file.Path,
					Lines:    file.Lines,
					LineHits: make(map[int]int),
				}
				for k, v := range file.LineHits {
					fileCopy.LineHits[k] = v
				}
				allFiles[file.Path] = fileCopy
			}
		}
	}

	// Update percentages for merged files
	for _, file := range allFiles {
		if file.Lines.Total > 0 {
			file.Lines.Percent = (float64(file.Lines.Covered) / float64(file.Lines.Total)) * 100.0
		}
	}

	linePercentage := 0.0
	if totalLines > 0 {
		linePercentage = (float64(coveredLines) / float64(totalLines)) * 100.0
	}

	// Convert map to slice
	files := make([]*FileCoverage, 0, len(allFiles))
	for _, file := range allFiles {
		files = append(files, file)
	}

	aggregateData := &CoverageData{
		Lines: CoverageMetric{
			Total:   totalLines,
			Covered: coveredLines,
			Percent: linePercentage,
		},
		Files: files,
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
		if err := os.MkdirAll(a.outputDir, 0o755); err != nil {
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

// GenerateAllReports generates all coverage report formats
func (a *CoverageAggregator) GenerateAllReports() error {
	formats := []string{"json", "cobertura", "html"}
	for _, format := range formats {
		if err := a.GenerateReport(format); err != nil {
			return err
		}
	}
	return nil
}

// CoverageJSONReport is the JSON report structure
type CoverageJSONReport struct {
	Generated    string                      `json:"generated"`
	Threshold    float64                     `json:"threshold"`
	ThresholdMet bool                        `json:"threshold_met"`
	Summary      CoverageSummary             `json:"summary"`
	Services     map[string]*ServiceCoverage `json:"services"`
	Files        []*FileCoverageReport       `json:"files,omitempty"`
}

// CoverageSummary contains overall coverage statistics
type CoverageSummary struct {
	Lines         CoverageMetric `json:"lines"`
	TotalFiles    int            `json:"total_files"`
	TotalServices int            `json:"total_services"`
}

// ServiceCoverage contains coverage data for a single service
type ServiceCoverage struct {
	Lines CoverageMetric        `json:"lines"`
	Files []*FileCoverageReport `json:"files,omitempty"`
}

// FileCoverageReport contains coverage data for a single file
type FileCoverageReport struct {
	Path           string         `json:"path"`
	Lines          CoverageMetric `json:"lines"`
	UncoveredLines []int          `json:"uncovered_lines,omitempty"`
}

// generateJSONReport generates a JSON coverage report
func (a *CoverageAggregator) generateJSONReport(aggregate *AggregateCoverage) error {
	outputPath := filepath.Join(a.outputDir, "coverage.json")

	report := CoverageJSONReport{
		Generated:    time.Now().Format(time.RFC3339),
		Threshold:    aggregate.Threshold,
		ThresholdMet: aggregate.Met,
		Summary: CoverageSummary{
			Lines:         aggregate.Aggregate.Lines,
			TotalFiles:    len(aggregate.Aggregate.Files),
			TotalServices: len(aggregate.Services),
		},
		Services: make(map[string]*ServiceCoverage),
		Files:    make([]*FileCoverageReport, 0),
	}

	// Add per-service coverage
	for name, coverage := range aggregate.Services {
		serviceCov := &ServiceCoverage{
			Lines: coverage.Lines,
			Files: make([]*FileCoverageReport, 0),
		}

		for _, file := range coverage.Files {
			fileReport := &FileCoverageReport{
				Path:  file.Path,
				Lines: file.Lines,
			}

			// Add uncovered lines
			for lineNum, hits := range file.LineHits {
				if hits == 0 {
					fileReport.UncoveredLines = append(fileReport.UncoveredLines, lineNum)
				}
			}
			sort.Ints(fileReport.UncoveredLines)

			serviceCov.Files = append(serviceCov.Files, fileReport)
		}

		report.Services[name] = serviceCov
	}

	// Add aggregate file coverage
	for _, file := range aggregate.Aggregate.Files {
		fileReport := &FileCoverageReport{
			Path:  file.Path,
			Lines: file.Lines,
		}

		for lineNum, hits := range file.LineHits {
			if hits == 0 {
				fileReport.UncoveredLines = append(fileReport.UncoveredLines, lineNum)
			}
		}
		sort.Ints(fileReport.UncoveredLines)

		report.Files = append(report.Files, fileReport)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal coverage data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
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
	XMLName    xml.Name        `xml:"class"`
	Name       string          `xml:"name,attr"`
	Filename   string          `xml:"filename,attr"`
	LineRate   float64         `xml:"line-rate,attr"`
	BranchRate float64         `xml:"branch-rate,attr"`
	Complexity float64         `xml:"complexity,attr"`
	Lines      []CoberturaLine `xml:"lines>line,omitempty"`
}

// CoberturaLine represents a line in Cobertura format
type CoberturaLine struct {
	XMLName xml.Name `xml:"line"`
	Number  int      `xml:"number,attr"`
	Hits    int      `xml:"hits,attr"`
}

// CoberturaCoverage represents the root Cobertura XML structure
type CoberturaCoverage struct {
	XMLName    xml.Name           `xml:"coverage"`
	LineRate   float64            `xml:"line-rate,attr"`
	BranchRate float64            `xml:"branch-rate,attr"`
	Version    string             `xml:"version,attr"`
	Timestamp  int64              `xml:"timestamp,attr"`
	Sources    []string           `xml:"sources>source,omitempty"`
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
		Timestamp:  time.Now().Unix(),
		Sources:    []string{},
		Packages:   []CoberturaPackage{},
	}

	// Add source root if set
	if a.sourceRoot != "" {
		coverage.Sources = append(coverage.Sources, a.sourceRoot)
	}

	// Add packages for each service with full file details
	for service, serviceCoverage := range aggregate.Services {
		serviceLineRate := serviceCoverage.Lines.Percent / 100.0

		pkg := CoberturaPackage{
			Name:       service,
			LineRate:   serviceLineRate,
			BranchRate: serviceLineRate,
			Complexity: 0,
			Classes:    []CoberturaClass{},
		}

		// Add each file as a class
		for _, file := range serviceCoverage.Files {
			fileLineRate := file.Lines.Percent / 100.0
			class := CoberturaClass{
				Name:       filepath.Base(file.Path),
				Filename:   file.Path,
				LineRate:   fileLineRate,
				BranchRate: fileLineRate,
				Complexity: 0,
				Lines:      []CoberturaLine{},
			}

			// Add line-level coverage
			for lineNum, hits := range file.LineHits {
				class.Lines = append(class.Lines, CoberturaLine{
					Number: lineNum,
					Hits:   hits,
				})
			}

			// Sort lines by number
			sort.Slice(class.Lines, func(i, j int) bool {
				return class.Lines[i].Number < class.Lines[j].Number
			})

			pkg.Classes = append(pkg.Classes, class)
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

	if err := os.WriteFile(outputPath, xmlData, 0o644); err != nil {
		return fmt.Errorf("failed to write Cobertura report: %w", err)
	}

	return nil
}

// generateHTMLReport generates an HTML coverage report with source linking
func (a *CoverageAggregator) generateHTMLReport(aggregate *AggregateCoverage) error {
	// Generate main index page
	if err := a.generateHTMLIndex(aggregate); err != nil {
		return err
	}

	// Generate per-service pages
	for service, coverage := range aggregate.Services {
		if err := a.generateServiceHTML(service, coverage); err != nil {
			return err
		}
	}

	// Generate per-file pages with source highlighting
	for _, file := range aggregate.Aggregate.Files {
		if err := a.generateFileHTML(file); err != nil {
			// Don't fail if source file doesn't exist
			continue
		}
	}

	return nil
}

// generateHTMLIndex generates the main HTML index page
func (a *CoverageAggregator) generateHTMLIndex(aggregate *AggregateCoverage) error {
	outputPath := filepath.Join(a.outputDir, "coverage.html")

	linePercent := aggregate.Aggregate.Lines.Percent
	covered := aggregate.Aggregate.Lines.Covered
	total := aggregate.Aggregate.Lines.Total

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Code Coverage Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 0; background: #f9fafb; }
        .container { max-width: 1200px; margin: 0 auto; padding: 40px 20px; }
        h1 { color: #111827; margin-bottom: 8px; }
        .timestamp { color: #6b7280; font-size: 14px; margin-bottom: 24px; }
        .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 24px 0; }
        .summary-card { background: white; padding: 24px; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .metric { font-size: 36px; font-weight: bold; color: %s; }
        .label { color: #6b7280; font-size: 14px; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 8px; }
        .sub-label { color: #9ca3af; font-size: 12px; margin-top: 4px; }
        table { width: 100%%; border-collapse: collapse; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin-top: 24px; }
        th { background: #1f2937; color: white; padding: 16px; text-align: left; font-weight: 500; }
        td { padding: 16px; border-bottom: 1px solid #e5e7eb; }
        tr:hover { background: #f9fafb; }
        .high { color: #059669; }
        .medium { color: #d97706; }
        .low { color: #dc2626; }
        .progress-bar { width: 100px; height: 8px; background: #e5e7eb; border-radius: 4px; overflow: hidden; display: inline-block; vertical-align: middle; margin-left: 8px; }
        .progress-fill { height: 100%%; transition: width 0.3s; }
        .progress-high { background: #059669; }
        .progress-medium { background: #d97706; }
        .progress-low { background: #dc2626; }
        a { color: #2563eb; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .threshold { padding: 16px; border-radius: 8px; margin-top: 24px; }
        .threshold-met { background: #d1fae5; color: #065f46; }
        .threshold-unmet { background: #fee2e2; color: #991b1b; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Code Coverage Report</h1>
        <div class="timestamp">Generated: %s</div>
        
        <div class="summary-grid">
            <div class="summary-card">
                <div class="label">Line Coverage</div>
                <div class="metric">%.1f%%</div>
                <div class="sub-label">%d / %d lines covered</div>
            </div>
            <div class="summary-card">
                <div class="label">Services</div>
                <div class="metric">%d</div>
                <div class="sub-label">tested services</div>
            </div>
            <div class="summary-card">
                <div class="label">Files</div>
                <div class="metric">%d</div>
                <div class="sub-label">source files</div>
            </div>
        </div>

        <div class="threshold %s">
            <strong>Coverage Threshold: %.1f%%</strong> - %s
        </div>

        <h2>Coverage by Service</h2>
        <table>
            <tr>
                <th>Service</th>
                <th>Lines Covered</th>
                <th>Total Lines</th>
                <th>Coverage</th>
            </tr>
`, getCoverageColor(linePercent), time.Now().Format("2006-01-02 15:04:05"),
		linePercent, covered, total,
		len(aggregate.Services), len(aggregate.Aggregate.Files),
		getThresholdClass(aggregate.Met), aggregate.Threshold, getThresholdMessage(aggregate.Met))

	// Sort services by name
	serviceNames := make([]string, 0, len(aggregate.Services))
	for name := range aggregate.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	for _, service := range serviceNames {
		coverage := aggregate.Services[service]
		percentage := coverage.Lines.Percent
		progressClass := getProgressClass(percentage)

		// Security: HTML-escape service name to prevent XSS
		htmlContent += fmt.Sprintf(`        <tr>
            <td><a href="service-%s.html">%s</a></td>
            <td>%d</td>
            <td>%d</td>
            <td class="%s">%.1f%% <div class="progress-bar"><div class="progress-fill %s" style="width: %.1f%%"></div></div></td>
        </tr>
`, html.EscapeString(service), html.EscapeString(service),
			coverage.Lines.Covered, coverage.Lines.Total,
			getCoverageClass(percentage), percentage, progressClass, percentage)
	}

	htmlContent += `    </table>

        <h2>All Files</h2>
        <table>
            <tr>
                <th>File</th>
                <th>Lines Covered</th>
                <th>Total Lines</th>
                <th>Coverage</th>
            </tr>
`

	// Sort files by path
	files := make([]*FileCoverage, len(aggregate.Aggregate.Files))
	copy(files, aggregate.Aggregate.Files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	for _, file := range files {
		percentage := file.Lines.Percent
		progressClass := getProgressClass(percentage)
		fileName := filepath.Base(file.Path)
		fileLink := fmt.Sprintf("file-%s.html", sanitizeFilename(file.Path))

		htmlContent += fmt.Sprintf(`        <tr>
            <td><a href="%s" title="%s">%s</a></td>
            <td>%d</td>
            <td>%d</td>
            <td class="%s">%.1f%% <div class="progress-bar"><div class="progress-fill %s" style="width: %.1f%%"></div></div></td>
        </tr>
`, fileLink, html.EscapeString(file.Path), html.EscapeString(fileName),
			file.Lines.Covered, file.Lines.Total,
			getCoverageClass(percentage), percentage, progressClass, percentage)
	}

	htmlContent += `    </table>
    </div>
</body>
</html>`

	if err := os.WriteFile(outputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}

// generateServiceHTML generates an HTML page for a service
func (a *CoverageAggregator) generateServiceHTML(service string, coverage *CoverageData) error {
	outputPath := filepath.Join(a.outputDir, fmt.Sprintf("service-%s.html", sanitizeFilename(service)))

	linePercent := coverage.Lines.Percent

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s - Coverage Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 0; background: #f9fafb; }
        .container { max-width: 1200px; margin: 0 auto; padding: 40px 20px; }
        h1 { color: #111827; margin-bottom: 8px; }
        .breadcrumb { color: #6b7280; margin-bottom: 24px; }
        .breadcrumb a { color: #2563eb; text-decoration: none; }
        .summary { background: white; padding: 24px; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin: 24px 0; }
        .metric { font-size: 36px; font-weight: bold; color: %s; }
        .label { color: #6b7280; font-size: 14px; }
        table { width: 100%%; border-collapse: collapse; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        th { background: #1f2937; color: white; padding: 16px; text-align: left; }
        td { padding: 16px; border-bottom: 1px solid #e5e7eb; }
        tr:hover { background: #f9fafb; }
        .high { color: #059669; }
        .medium { color: #d97706; }
        .low { color: #dc2626; }
        a { color: #2563eb; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="breadcrumb"><a href="coverage.html">← Back to Overview</a></div>
        <h1>%s</h1>
        
        <div class="summary">
            <div class="label">Line Coverage</div>
            <div class="metric">%.1f%%</div>
            <div class="label">%d / %d lines covered</div>
        </div>

        <h2>Files</h2>
        <table>
            <tr>
                <th>File</th>
                <th>Lines Covered</th>
                <th>Total Lines</th>
                <th>Coverage</th>
            </tr>
`, html.EscapeString(service), getCoverageColor(linePercent),
		html.EscapeString(service), linePercent, coverage.Lines.Covered, coverage.Lines.Total)

	for _, file := range coverage.Files {
		percentage := file.Lines.Percent
		fileName := filepath.Base(file.Path)
		fileLink := fmt.Sprintf("file-%s.html", sanitizeFilename(file.Path))

		htmlContent += fmt.Sprintf(`        <tr>
            <td><a href="%s" title="%s">%s</a></td>
            <td>%d</td>
            <td>%d</td>
            <td class="%s">%.1f%%</td>
        </tr>
`, fileLink, html.EscapeString(file.Path), html.EscapeString(fileName),
			file.Lines.Covered, file.Lines.Total, getCoverageClass(percentage), percentage)
	}

	htmlContent += `    </table>
    </div>
</body>
</html>`

	if err := os.WriteFile(outputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("failed to write service HTML: %w", err)
	}

	return nil
}

// generateFileHTML generates an HTML page for a file with source code highlighting
func (a *CoverageAggregator) generateFileHTML(file *FileCoverage) error {
	outputPath := filepath.Join(a.outputDir, fmt.Sprintf("file-%s.html", sanitizeFilename(file.Path)))

	// Try to read source file
	sourceContent := ""
	sourcePath := file.Path
	if a.sourceRoot != "" {
		sourcePath = filepath.Join(a.sourceRoot, file.Path)
	}

	// Validate path before reading to prevent path traversal
	if err := security.ValidatePath(sourcePath); err != nil {
		// If validation fails, generate a simpler report
		sourceContent = "// Source file not available (path validation failed)"
	} else if sourceBytes, err := os.ReadFile(sourcePath); err != nil { // #nosec G304 -- sourcePath validated above
		// If we can't read the source, generate a simpler report
		sourceContent = "// Source file not available"
	} else {
		sourceContent = string(sourceBytes)
	}

	lines := strings.Split(sourceContent, "\n")
	linePercent := file.Lines.Percent

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s - Coverage</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 0; background: #f9fafb; }
        .container { max-width: 1400px; margin: 0 auto; padding: 40px 20px; }
        h1 { color: #111827; margin-bottom: 8px; font-size: 18px; }
        .breadcrumb { color: #6b7280; margin-bottom: 24px; }
        .breadcrumb a { color: #2563eb; text-decoration: none; }
        .summary { background: white; padding: 16px; border-radius: 8px; margin: 16px 0; display: inline-block; }
        .metric { font-size: 24px; font-weight: bold; color: %s; }
        .source { background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .source-header { background: #1f2937; color: white; padding: 12px 16px; font-family: monospace; font-size: 14px; }
        table { width: 100%%; border-collapse: collapse; }
        td { padding: 0; vertical-align: top; }
        .line-num { width: 60px; text-align: right; padding: 0 12px; background: #f3f4f6; color: #9ca3af; font-family: monospace; font-size: 12px; user-select: none; border-right: 1px solid #e5e7eb; }
        .line-hits { width: 40px; text-align: center; padding: 0 8px; font-family: monospace; font-size: 11px; color: #6b7280; }
        .line-code { padding: 0 16px; font-family: monospace; font-size: 13px; white-space: pre; overflow-x: auto; }
        tr { height: 20px; }
        .covered { background: #d1fae5; }
        .uncovered { background: #fee2e2; }
        .covered .line-num { background: #a7f3d0; }
        .uncovered .line-num { background: #fecaca; }
        .not-executable .line-num { background: #f3f4f6; }
    </style>
</head>
<body>
    <div class="container">
        <div class="breadcrumb"><a href="coverage.html">← Back to Overview</a></div>
        <h1>%s</h1>
        
        <div class="summary">
            <span class="metric">%.1f%%</span> coverage - %d / %d lines
        </div>

        <div class="source">
            <div class="source-header">%s</div>
            <table>
`, html.EscapeString(filepath.Base(file.Path)), getCoverageColor(linePercent),
		html.EscapeString(file.Path), linePercent, file.Lines.Covered, file.Lines.Total,
		html.EscapeString(file.Path))

	for i, line := range lines {
		lineNum := i + 1
		hits, hasHits := file.LineHits[lineNum]

		rowClass := "not-executable"
		hitsDisplay := ""
		if hasHits {
			if hits > 0 {
				rowClass = "covered"
				hitsDisplay = fmt.Sprintf("%dx", hits)
			} else {
				rowClass = "uncovered"
				hitsDisplay = "0x"
			}
		}

		htmlContent += fmt.Sprintf(`            <tr class="%s">
                <td class="line-num">%d</td>
                <td class="line-hits">%s</td>
                <td class="line-code">%s</td>
            </tr>
`, rowClass, lineNum, hitsDisplay, html.EscapeString(line))
	}

	htmlContent += `        </table>
        </div>
    </div>
</body>
</html>`

	if err := os.WriteFile(outputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("failed to write file HTML: %w", err)
	}

	return nil
}

// getCoverageColor returns the color for a coverage percentage
func getCoverageColor(percentage float64) string {
	if percentage >= CoverageThresholdHigh {
		return "#059669"
	} else if percentage >= CoverageThresholdMedium {
		return "#d97706"
	}
	return "#dc2626"
}

// getCoverageClass returns the CSS class for a coverage percentage
func getCoverageClass(percentage float64) string {
	if percentage >= CoverageThresholdHigh {
		return "high"
	} else if percentage >= CoverageThresholdMedium {
		return "medium"
	}
	return "low"
}

// getProgressClass returns the progress bar CSS class
func getProgressClass(percentage float64) string {
	if percentage >= CoverageThresholdHigh {
		return "progress-high"
	} else if percentage >= CoverageThresholdMedium {
		return "progress-medium"
	}
	return "progress-low"
}

// getThresholdClass returns the CSS class for threshold status
func getThresholdClass(met bool) string {
	if met {
		return "threshold-met"
	}
	return "threshold-unmet"
}

// getThresholdMessage returns the message for threshold status
func getThresholdMessage(met bool) string {
	if met {
		return "✓ Threshold met"
	}
	return "✗ Below threshold"
}

// sanitizeFilename creates a safe filename from a path
func sanitizeFilename(path string) string {
	// Replace path separators and special characters
	safe := strings.ReplaceAll(path, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	safe = strings.ReplaceAll(safe, ":", "-")
	safe = strings.ReplaceAll(safe, " ", "-")

	// Remove leading dashes
	safe = strings.TrimLeft(safe, "-")

	return safe
}
