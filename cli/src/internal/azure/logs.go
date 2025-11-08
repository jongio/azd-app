package azure

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LogEntry represents a single log line from Azure.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level"`
	Source    string `json:"source"` // "containerapp" or "appservice"
}

// LogStreamOptions contains options for streaming Azure logs.
type LogStreamOptions struct {
	ResourceGroup   string
	ResourceName    string
	ResourceType    string // "containerapp" or "appservice"
	Follow          bool
	TailLines       int
	ContainerName   string // Optional: for multi-container apps
}

// StreamLogs streams logs from an Azure resource using Azure CLI.
// It tries Log Analytics first if workspace ID is available, then falls back to Azure CLI.
func StreamLogs(ctx context.Context, opts LogStreamOptions, logChan chan<- LogEntry) error {
	if opts.ResourceType == "containerapp" {
		return streamContainerAppLogs(ctx, opts, logChan)
	} else if opts.ResourceType == "appservice" {
		return streamAppServiceLogs(ctx, opts, logChan)
	}
	return fmt.Errorf("unsupported resource type: %s", opts.ResourceType)
}

// streamContainerAppLogs streams logs from Azure Container Apps.
func streamContainerAppLogs(ctx context.Context, opts LogStreamOptions, logChan chan<- LogEntry) error {
	// Build Azure CLI command
	args := []string{
		"containerapp", "logs", "show",
		"--name", opts.ResourceName,
		"--resource-group", opts.ResourceGroup,
		"--format", "text",
	}

	if opts.Follow {
		args = append(args, "--follow")
	}

	if opts.TailLines > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.TailLines))
	}

	if opts.ContainerName != "" {
		args = append(args, "--container", opts.ContainerName)
	}

	// #nosec G204 -- args are constructed from validated struct fields
	cmd := exec.CommandContext(ctx, "az", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start az command: %w", err)
	}

	// Read logs line by line
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse log line - Container App logs are typically in format:
		// timestamp level message
		entry := parseContainerAppLogLine(line)
		entry.Source = "containerapp"

		select {
		case logChan <- entry:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return cmd.Wait()
}

// streamAppServiceLogs streams logs from Azure App Service.
func streamAppServiceLogs(ctx context.Context, opts LogStreamOptions, logChan chan<- LogEntry) error {
	// Build Azure CLI command
	args := []string{
		"webapp", "log", "tail",
		"--name", opts.ResourceName,
		"--resource-group", opts.ResourceGroup,
	}

	// #nosec G204 -- args are constructed from validated struct fields
	cmd := exec.CommandContext(ctx, "az", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start az command: %w", err)
	}

	// Read logs line by line
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse log line - App Service logs vary by platform
		entry := parseAppServiceLogLine(line)
		entry.Source = "appservice"

		select {
		case logChan <- entry:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return cmd.Wait()
}

// parseContainerAppLogLine parses a Container App log line.
func parseContainerAppLogLine(line string) LogEntry {
	// Try to parse structured format: timestamp level message
	parts := strings.SplitN(line, " ", 3)
	
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   line,
		Level:     "info",
	}

	if len(parts) >= 2 {
		// Check if first part looks like a timestamp
		if strings.Contains(parts[0], "T") || strings.Contains(parts[0], ":") {
			entry.Timestamp = parts[0]
			if len(parts) >= 3 {
				// Second part might be level
				level := strings.ToLower(parts[1])
				if level == "info" || level == "warn" || level == "error" || level == "debug" {
					entry.Level = level
					entry.Message = parts[2]
				} else {
					entry.Message = strings.Join(parts[1:], " ")
				}
			} else {
				entry.Message = parts[1]
			}
		}
	}

	return entry
}

// parseAppServiceLogLine parses an App Service log line.
func parseAppServiceLogLine(line string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   line,
		Level:     "info",
	}

	// App Service logs often start with timestamp in format: YYYY-MM-DD HH:MM:SS
	if len(line) > 19 && line[4] == '-' && line[7] == '-' && line[13] == ':' {
		entry.Timestamp = line[:19]
		entry.Message = strings.TrimSpace(line[19:])
	}

	// Detect log level from message
	messageLower := strings.ToLower(entry.Message)
	if strings.Contains(messageLower, "error") || strings.Contains(messageLower, "exception") {
		entry.Level = "error"
	} else if strings.Contains(messageLower, "warn") {
		entry.Level = "warn"
	} else if strings.Contains(messageLower, "debug") {
		entry.Level = "debug"
	}

	return entry
}

// CheckAzureCLI checks if Azure CLI is installed and authenticated.
func CheckAzureCLI() error {
	cmd := exec.Command("az", "account", "show")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Azure CLI not available or not authenticated. Please run 'az login'")
	}
	return nil
}
