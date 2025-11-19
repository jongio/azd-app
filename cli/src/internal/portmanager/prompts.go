package portmanager

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// promptUserChoice prompts the user to choose from a list of options.
// Returns the user's choice (trimmed) and any error from reading input.
func promptUserChoice(message string, options []string) (string, error) {
	fmt.Fprint(os.Stderr, message)
	for i, opt := range options {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, opt)
	}
	fmt.Fprint(os.Stderr, "\nChoose (")
	for i := range options {
		if i > 0 {
			fmt.Fprint(os.Stderr, "/")
		}
		fmt.Fprintf(os.Stderr, "%d", i+1)
	}
	fmt.Fprint(os.Stderr, "): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// promptYesNo prompts the user with a yes/no question.
// Returns true for yes, false for no or error.
func promptYesNo(message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s (y/N): ", message)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// formatPortConflictMessage formats a port conflict error message with process info.
func formatPortConflictMessage(serviceName string, port int, processInfo string) {
	fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' requires port %d (configured in azure.yaml)\n", serviceName, port)
	fmt.Fprintf(os.Stderr, "This port is currently in use%s.\n\n", processInfo)
}

// formatPortAssignedMessage formats a success message for port assignment.
func formatPortAssignedMessage(port int, serviceName string) {
	fmt.Fprintf(os.Stderr, "\n✓ Assigned port %d to service '%s'\n", port, serviceName)
}

// formatUpdateAzureYamlWarning formats a warning about updating azure.yaml.
func formatUpdateAzureYamlWarning(port int) {
	fmt.Fprintf(os.Stderr, "\n⚠️  IMPORTANT: Update your application code to use port %d\n", port)
}
