package main

import (
	"fmt"
	"os"

	"github.com/jongio/azd-app/cli/src/internal/yamlutil"
)

func main() {
	// Path to the test azure.yaml file (absolute path)
	testFile := "c:\\code\\azd-app-2\\cli\\tests\\projects\\integration\\azure-logs-test\\azure.yaml"

	// Read original content
	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== ORIGINAL azure.yaml ===")
	fmt.Println(string(originalContent))
	fmt.Println()

	// Update containerapp-api service with new log tables
	tables := []string{"ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL", "AppRequests"}
	if updateErr := yamlutil.UpdateServiceLogsConfig(testFile, "containerapp-api", tables, ""); updateErr != nil {
		fmt.Printf("Error updating logs config: %v\n", updateErr)
		os.Exit(1)
	}

	// Read updated content
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("Error reading updated file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== UPDATED azure.yaml ===")
	fmt.Println(string(updatedContent))
	fmt.Println()

	// Verify critical elements are preserved
	updatedStr := string(updatedContent)
	checks := []struct {
		name  string
		value string
	}{
		{"Schema comment", "# yaml-language-server: $schema="},
		{"Project name", "name: azure-logs-test"},
		{"Global logs section", "logs:"},
		{"Global filters", "filters:"},
		{"Reqs section", "reqs:"},
		{"Services section", "services:"},
		{"containerapp-api service", "containerapp-api:"},
		{"appservice-web service", "appservice-web:"},
		{"functions-worker service", "functions-worker:"},
		{"azurite service", "azurite:"},
		{"New log table 1", "ContainerAppConsoleLogs_CL"},
		{"New log table 2", "ContainerAppSystemLogs_CL"},
		{"New log table 3", "AppRequests"},
	}

	allPassed := true
	for _, check := range checks {
		if !contains(updatedStr, check.value) {
			fmt.Printf("❌ FAILED: %s not found\n", check.name)
			allPassed = false
		} else {
			fmt.Printf("✅ PASSED: %s preserved\n", check.name)
		}
	}

	// Restore original content
	if err := os.WriteFile(testFile, originalContent, 0600); err != nil {
		fmt.Printf("Error restoring original file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n✅ Original file restored")

	if !allPassed {
		os.Exit(1)
	}

	fmt.Println("\n🎉 All checks passed! The fix works correctly.")
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		(haystack == needle || len(haystack) > len(needle) &&
			(haystack[:len(needle)] == needle ||
				haystack[len(haystack)-len(needle):] == needle ||
				findSubstring(haystack, needle) >= 0))
}

func findSubstring(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
