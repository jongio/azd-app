package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AutoFixRequest represents a request to auto-fix Bicep configuration.
type AutoFixRequest struct {
	Action string `json:"action"` // "add-bicep-outputs"
}

// AutoFixResponse represents the response from an auto-fix operation.
type AutoFixResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	BicepFile  string `json:"bicepFile,omitempty"`
	Changes    string `json:"changes,omitempty"`
	Applied    bool   `json:"applied"`
	PreviewURL string `json:"previewUrl,omitempty"`
}

// handleAzureSetupAutoFix handles POST /api/azure/setup/auto-fix
// This endpoint auto-fixes common Bicep configuration issues.
func (s *Server) handleAzureSetupAutoFix(w http.ResponseWriter, r *http.Request) {
	var req AutoFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	switch req.Action {
	case "add-bicep-outputs":
		s.handleAddBicepOutputs(w, r)
	default:
		writeJSONError(w, http.StatusBadRequest, "Unknown action: "+req.Action, nil)
	}
}

// handleAddBicepOutputs adds missing Log Analytics outputs to the main.bicep file.
func (s *Server) handleAddBicepOutputs(w http.ResponseWriter, r *http.Request) {
	// Find the main Bicep file
	bicepFile, err := s.findMainBicepFile()
	if err != nil {
		WriteJSONSuccess(w, AutoFixResponse{
			Success: false,
			Message: fmt.Sprintf("Could not find main Bicep file: %v", err),
		})
		return
	}

	// Read current content
	content, err := os.ReadFile(bicepFile)
	if err != nil {
		WriteJSONSuccess(w, AutoFixResponse{
			Success: false,
			Message: fmt.Sprintf("Could not read Bicep file: %v", err),
		})
		return
	}

	// Check if outputs already exist
	contentStr := string(content)
	if strings.Contains(contentStr, "AZURE_LOG_ANALYTICS_WORKSPACE_ID") &&
		strings.Contains(contentStr, "AZURE_LOG_ANALYTICS_WORKSPACE_GUID") {
		WriteJSONSuccess(w, AutoFixResponse{
			Success:   true,
			Message:   "Bicep outputs already configured",
			BicepFile: bicepFile,
			Applied:   false,
		})
		return
	}

	// Detect the Log Analytics module name
	moduleName := detectLogAnalyticsModuleName(contentStr)
	if moduleName == "" {
		WriteJSONSuccess(w, AutoFixResponse{
			Success: false,
			Message: "Could not detect Log Analytics module in Bicep file. Please add outputs manually.",
		})
		return
	}

	// Generate the outputs to add
	outputs := generateLogAnalyticsOutputs(moduleName)

	// Find insertion point (after last output or at end of file)
	newContent, insertionPoint := insertBicepOutputs(contentStr, outputs)

	// Write the updated content
	if err := os.WriteFile(bicepFile, []byte(newContent), 0644); err != nil {
		WriteJSONSuccess(w, AutoFixResponse{
			Success: false,
			Message: fmt.Sprintf("Could not write Bicep file: %v", err),
		})
		return
	}

	WriteJSONSuccess(w, AutoFixResponse{
		Success:   true,
		Message:   fmt.Sprintf("Added Log Analytics outputs to %s at line %d. Run 'azd provision' to apply.", filepath.Base(bicepFile), insertionPoint),
		BicepFile: bicepFile,
		Changes:   outputs,
		Applied:   true,
	})
}

// findMainBicepFile locates the main Bicep file in the project.
func (s *Server) findMainBicepFile() (string, error) {
	// Common locations to check
	candidates := []string{
		filepath.Join(s.projectDir, "infra", "main.bicep"),
		filepath.Join(s.projectDir, "infra", "main.bicepparam"),
		filepath.Join(s.projectDir, "main.bicep"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no main.bicep found in infra/ directory")
}

// detectLogAnalyticsModuleName finds the module name used for Log Analytics workspace.
func detectLogAnalyticsModuleName(content string) string {
	// Pattern to match module declarations for Log Analytics
	// e.g., module logAnalytics 'br/public:avm/res/operational-insights/workspace:...'
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`module\s+(\w+)\s+'br/public:avm/res/operational-insights/workspace`),
		regexp.MustCompile(`module\s+(\w+)\s+'.*/operational-insights/workspace`),
		regexp.MustCompile(`module\s+(\w+)\s+'.*log.*analytics.*\.bicep'`),
		regexp.MustCompile(`resource\s+(\w+)\s+'Microsoft\.OperationalInsights/workspaces@`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// generateLogAnalyticsOutputs creates the Bicep output declarations.
func generateLogAnalyticsOutputs(moduleName string) string {
	// Check if it's a module (has .outputs) or a resource (has .id, .name, .properties)
	isResource := strings.HasPrefix(moduleName, "resource")

	if isResource || !strings.Contains(moduleName, "module") {
		// Direct resource reference
		return fmt.Sprintf(`
// Log Analytics Workspace Outputs (auto-added by azd-app)
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = %s.id
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = %s.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = %s.properties.customerId
`, moduleName, moduleName, moduleName)
	}

	// Module reference - check for AVM outputs
	return fmt.Sprintf(`
// Log Analytics Workspace Outputs (auto-added by azd-app)
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = %s.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = %s.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = %s.outputs.logAnalyticsWorkspaceId
`, moduleName, moduleName, moduleName)
}

// insertBicepOutputs adds new outputs to the Bicep content.
// Returns the new content and the line number where outputs were inserted.
func insertBicepOutputs(content, outputs string) (string, int) {
	lines := strings.Split(content, "\n")
	lastOutputLine := -1

	// Find the last output line
	outputPattern := regexp.MustCompile(`^\s*output\s+`)
	for i, line := range lines {
		if outputPattern.MatchString(line) {
			lastOutputLine = i
		}
	}

	if lastOutputLine >= 0 {
		// Insert after the last output
		insertLine := lastOutputLine + 1
		// Skip any blank lines after the last output
		for insertLine < len(lines) && strings.TrimSpace(lines[insertLine]) == "" {
			insertLine++
		}
		// Insert before any following content
		before := strings.Join(lines[:insertLine], "\n")
		after := strings.Join(lines[insertLine:], "\n")
		return before + outputs + "\n" + after, insertLine + 1
	}

	// No outputs found - append at end of file
	return content + outputs, countLines(content) + 1
}

// countLines counts the number of lines in a string.
func countLines(s string) int {
	scanner := bufio.NewScanner(strings.NewReader(s))
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}
