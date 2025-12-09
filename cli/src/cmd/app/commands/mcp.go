package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// Timeout constants
const (
	defaultCommandTimeout    = 30 * time.Second
	dependencyInstallTimeout = 15 * time.Minute // Increased to handle large projects
	maxLogTailLines          = 10000            // Maximum number of log lines to retrieve
)

// Rate limiting constants
const (
	maxToolCallsPerMinute = 60 // Prevent MCP client abuse
	burstSize             = 10 // Allow burst of 10 calls
)

// Command constants
const (
	azdCommand     = "azd"
	appSubcommand  = "app"
	jsonOutputFlag = "--output"
	jsonOutputVal  = "json"
	cwdFlag        = "--cwd"
)

// Allowed values for validation
var (
	allowedLogLevels = map[string]bool{"info": true, "warn": true, "error": true, "debug": true, "all": true}
	allowedRuntimes  = map[string]bool{"azd": true, "aspire": true, "pnpm": true, "docker-compose": true}
	// safeNamePattern validates service names and other identifiers to prevent injection
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
)

// NewMCPCommand creates the mcp command with subcommands.
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp",
		Short:  "Model Context Protocol server operations",
		Long:   `Manage the Model Context Protocol (MCP) server for AI assistant integration`,
		Hidden: true, // Hidden from help - primarily used by azd internally
	}

	cmd.AddCommand(newMCPServeCommand())

	return cmd
}

// newMCPServeCommand creates the mcp serve subcommand.
func newMCPServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long:  `Starts the Model Context Protocol server to expose azd app functionality to AI assistants`,
		RunE:  runMCPServe,
	}
}

// runMCPServe starts the MCP server using Go implementation.
func runMCPServe(cmd *cobra.Command, args []string) error {
	return runMCPServer(cmd.Context())
}

// runMCPServer implements the MCP server logic
func runMCPServer(ctx context.Context) error {
	// System instructions to guide AI on how to use the tools
	// This server is part of the azd extension framework and provides runtime operations
	instructions := `This MCP server is provided by the azd app extension and focuses on runtime operations for azd projects.

**Extension Role:**
This server complements azd's core MCP capabilities:
- azd core MCP: Project planning, architecture discovery, infrastructure generation
- azd app MCP (this): Runtime operations, service monitoring, log analysis

**Best Practices:**
1. Always use get_services to check current state before starting/stopping services
2. Use check_requirements before installing dependencies to see what's needed
3. Use get_service_errors FIRST when debugging - it returns only errors with context
4. Use get_service_logs for full log history when you need more detail
5. Read azure://project/azure.yaml resource to understand project structure before operations

**Debugging Workflow:**
1. get_service_errors: Start here - returns errors with surrounding context for quick diagnosis
2. get_service_logs: Use if you need full log history or non-error messages
3. restart_service: After fixing issues, restart the affected service

**Tool Categories:**
- Observability: get_services, get_service_errors, get_service_logs, get_project_info
- Operations: run_services, stop_services, start_service, restart_service, install_dependencies
- Configuration: check_requirements, get_environment_variables, set_environment_variable

**Service Lifecycle:**
- run_services: Start all services (background process, use get_services to check status)
- start_service: Start a specific stopped service
- stop_services: Stop all or a specific running service (graceful shutdown)
- restart_service: Stop and start a specific service

**Integration Notes:**
- Works with projects created by azd init or azd templates
- Monitors services started by azd app run
- Complements azd's built-in deployment and provisioning workflows`

	// Create MCP server with all capabilities
	// Server name follows azd extension naming convention: {namespace}-mcp-server
	s := server.NewMCPServer(
		"app-mcp-server", Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true), // subscribe=false, listChanged=true
		server.WithPromptCapabilities(false),         // listChanged=false
		server.WithInstructions(instructions),
	)

	// Add tools
	tools := []server.ServerTool{
		// Observability tools
		newGetServicesTool(),
		newGetServiceLogsTool(),
		newGetServiceErrorsTool(),
		newGetProjectInfoTool(),
		// Operational tools
		newRunServicesTool(),
		newStopServicesTool(),
		newStartServiceTool(),
		newRestartServiceTool(),
		newInstallDependenciesTool(),
		newCheckRequirementsTool(),
		// Configuration tools
		newGetEnvironmentVariablesTool(),
		newSetEnvironmentVariableTool(),
	}

	s.AddTools(tools...)

	// Add resources
	resources := []server.ServerResource{
		newAzureYamlResource(),
		newServiceConfigResource(),
	}

	s.AddResources(resources...)

	// Start the server using stdio transport
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		return err
	}

	return nil
}

// executeAzdAppCommand executes an azd app command and returns JSON output
func executeAzdAppCommand(ctx context.Context, command string, args []string) (map[string]interface{}, error) {
	return executeAzdAppCommandWithTimeout(ctx, command, args, defaultCommandTimeout)
}

// executeAzdAppCommandWithTimeout executes an azd app command with a custom timeout
func executeAzdAppCommandWithTimeout(ctx context.Context, command string, args []string, timeout time.Duration) (map[string]interface{}, error) {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("command cancelled before execution: %w", err)
	}

	cmdArgs := append([]string{command}, args...)
	cmdArgs = append(cmdArgs, jsonOutputFlag, jsonOutputVal)

	// Use context with timeout for command execution
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, azdCommand, append([]string{appSubcommand}, cmdArgs...)...)

	// Set up process group to allow killing child processes
	// This is platform-specific but helps ensure cleanup
	setupProcessGroup(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if parent context was cancelled
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, fmt.Errorf("command cancelled: %w", ctx.Err())
		}
		// Check if command context timed out
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("command timed out after %v", timeout)
		}
		return nil, fmt.Errorf("failed to execute azd app %s: %w\nOutput: %s", command, err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w\nRaw output: %s", err, string(output))
	}

	return result, nil
}

// Helper functions for common MCP patterns

// getArgsMap safely extracts the arguments map from a request
func getArgsMap(request mcp.CallToolRequest) map[string]interface{} {
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		return args
	}
	return make(map[string]interface{})
}

// getStringParam safely extracts a string parameter from request arguments
func getStringParam(args map[string]interface{}, key string) (string, bool) {
	if val, ok := args[key].(string); ok && val != "" {
		return val, true
	}
	return "", false
}

// getFloat64Param safely extracts a float64 parameter from request arguments
func getFloat64Param(args map[string]interface{}, key string) (float64, bool) {
	if val, ok := args[key].(float64); ok {
		return val, true
	}
	return 0, false
}

// marshalToolResult marshals data to JSON and returns an MCP tool result
func marshalToolResult(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// checkRateLimitWithName checks if the operation is allowed under rate limiting
// Returns an error result if rate limit is exceeded
// operationName is used for logging purposes
func checkRateLimitWithName(operationName string) *mcp.CallToolResult {
	if !globalRateLimiter.Allow() {
		logRateLimitEvent(operationName)
		return mcp.NewToolResultError("Rate limit exceeded. Please wait before making more requests.")
	}
	return nil
}

// extractProjectDirArg extracts projectDir argument and returns command args with --cwd flag
// Returns the args and any validation error
func extractProjectDirArg(args map[string]interface{}) ([]string, error) {
	var cmdArgs []string
	if projectDir, ok := getStringParam(args, "projectDir"); ok {
		validatedPath, err := validateProjectDir(projectDir)
		if err != nil {
			return nil, err
		}
		cmdArgs = append(cmdArgs, cwdFlag, validatedPath)
	}
	return cmdArgs, nil
}

// extractValidatedProjectDir extracts and validates projectDir from args, returning the path.
// Falls back to getProjectDir() if not provided in args.
func extractValidatedProjectDir(args map[string]interface{}) (string, error) {
	projectDir := getProjectDir()
	if pd, ok := getStringParam(args, "projectDir"); ok {
		validatedPath, err := validateProjectDir(pd)
		if err != nil {
			return "", err
		}
		projectDir = validatedPath
	}
	return projectDir, nil
}

// validateRequiredParam validates that a required parameter exists and returns the value
func validateRequiredParam(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("%s parameter is required", key)
	}
	return val, nil
}

// validateEnumParam validates that a parameter value is in allowed set
func validateEnumParam(value string, allowed map[string]bool, paramName string) error {
	if value == "" {
		return nil // Empty is OK for optional params
	}
	if !allowed[value] {
		var validValues []string
		for k := range allowed {
			validValues = append(validValues, k)
		}
		return fmt.Errorf("invalid %s: '%s'. Valid values: %s", paramName, value, strings.Join(validValues, ", "))
	}
	return nil
}

// isValidDuration validates a duration string format (e.g., "5m", "1h", "30s")
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	// Simple validation: must end with s, m, or h and have a numeric prefix
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	suffix := s[len(s)-1]
	if suffix != 's' && suffix != 'm' && suffix != 'h' {
		return false
	}
	prefix := s[:len(s)-1]
	for _, c := range prefix {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(prefix) > 0
}

// validateProjectDir validates that the project directory path is safe
// Prevents path traversal attacks and ensures the directory exists
func validateProjectDir(dir string) (string, error) {
	if dir == "" || dir == "." {
		// Get current working directory for "." reference
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		return cwd, nil
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(dir)

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid project directory path: %w", err)
	}

	// Resolve symbolic links to prevent symlink-based attacks
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist, that's an error for project directories
		if os.IsNotExist(err) {
			return "", fmt.Errorf("project directory does not exist: %s", absPath)
		}
		return "", fmt.Errorf("cannot resolve project directory path: %w", err)
	}

	// Verify the resolved path is a directory
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("cannot access project directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", resolvedPath)
	}

	// Security check: Ensure the resolved path doesn't contain suspicious patterns
	// This catches attempts to use .. after symlink resolution
	cleanResolved := filepath.Clean(resolvedPath)
	if strings.Contains(cleanResolved, "..") {
		return "", fmt.Errorf("project directory path contains parent directory traversal")
	}

	// Get current working directory to establish a baseline for allowed paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get user's home directory as an allowed boundary
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, just ensure we're not in system directories
		homeDir = ""
	}

	// Security: Only allow project directories under:
	// 1. Current working directory tree
	// 2. User's home directory tree
	// 3. Explicitly disallow system directories (/etc, /usr, /bin, /sbin, etc.)
	isUnderCwd := strings.HasPrefix(cleanResolved, cwd)
	isUnderHome := homeDir != "" && strings.HasPrefix(cleanResolved, homeDir)

	// Check for system directory access attempts (Unix-like systems)
	systemDirs := []string{"/etc", "/usr", "/bin", "/sbin", "/var", "/sys", "/proc", "/dev", "/root"}
	if strings.HasPrefix(cleanResolved, "/") { // Unix-like path
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanResolved, sysDir) {
				return "", fmt.Errorf("access to system directories not allowed: %s", cleanResolved)
			}
		}
	}

	// Check for Windows system directory access attempts
	if len(cleanResolved) >= 3 && cleanResolved[1] == ':' { // Windows path (e.g., C:\)
		lowerPath := strings.ToLower(cleanResolved)
		windowsSystemDirs := []string{
			`c:\windows`,
			`c:\program files`,
			`c:\program files (x86)`,
			`c:\programdata`,
			`c:\users\public`,
			`c:\users\default`,
			`c:\recovery`,
			`c:\$recycle.bin`,
			`c:\system volume information`,
		}
		for _, sysDir := range windowsSystemDirs {
			if strings.HasPrefix(lowerPath, sysDir) {
				return "", fmt.Errorf("access to system directories not allowed: %s", cleanResolved)
			}
		}
	}

	// Allow if under CWD or home directory
	if !isUnderCwd && !isUnderHome {
		return "", fmt.Errorf("project directory must be under current directory or home directory")
	}

	return cleanResolved, nil
}

// getProjectDir gets the project directory from AZD_APP_PROJECT_DIR environment variable or defaults to current directory
// This environment variable is set by azd when invoking the extension's MCP server
// The returned path is validated for security
func getProjectDir() string {
	// First try AZD_APP_PROJECT_DIR (set via extension.yaml mcp.serve.env)
	projectDir := os.Getenv("AZD_APP_PROJECT_DIR")
	if projectDir == "" {
		// Fall back to PROJECT_DIR for backwards compatibility
		projectDir = os.Getenv("PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir = "."
	}

	// Validate the environment variable value to prevent injection attacks
	// If validation fails, fall back to current directory
	validated, err := validateProjectDir(projectDir)
	if err != nil {
		// Log the validation failure but don't expose details to client
		// Use stderr since this is a server process
		fmt.Fprintf(os.Stderr, "Warning: Invalid project directory from environment: %v, using current directory\n", err)
		return "."
	}

	return validated
}

// ServiceInfo represents the output schema for get_services tool
type ServiceInfo struct {
	Project  map[string]interface{} `json:"project" jsonschema:"description=Project metadata including name and directory"`
	Services []ServiceDetails       `json:"services" jsonschema:"description=List of services with their status and configuration"`
}

// ServiceDetails represents individual service information
type ServiceDetails struct {
	Name      string            `json:"name" jsonschema:"description=Service name"`
	Language  string            `json:"language" jsonschema:"description=Programming language (e.g. python, javascript, dotnet)"`
	Framework string            `json:"framework" jsonschema:"description=Framework used (e.g. flask, express, aspnet)"`
	Project   string            `json:"project" jsonschema:"description=Path to the project directory"`
	Status    string            `json:"status,omitempty" jsonschema:"description=Current running status"`
	Health    string            `json:"health,omitempty" jsonschema:"description=Health check status"`
	URL       string            `json:"url,omitempty" jsonschema:"description=Local URL where service is running"`
	Port      int               `json:"port,omitempty" jsonschema:"description=Port number the service is listening on"`
	PID       int               `json:"pid,omitempty" jsonschema:"description=Process ID of the running service"`
	Env       map[string]string `json:"env,omitempty" jsonschema:"description=Environment variables configured for the service"`
}

// ProjectInfo represents the output schema for get_project_info tool
type ProjectInfo struct {
	Project  map[string]interface{}  `json:"project" jsonschema:"description=Project metadata"`
	Services []ProjectServiceSummary `json:"services" jsonschema:"description=Summary of services defined in the project"`
}

// ProjectServiceSummary represents a simplified service summary
type ProjectServiceSummary struct {
	Name      string `json:"name" jsonschema:"description=Service name"`
	Language  string `json:"language" jsonschema:"description=Programming language"`
	Framework string `json:"framework" jsonschema:"description=Framework used"`
	Project   string `json:"project" jsonschema:"description=Project directory path"`
}

// RequirementsResult represents the output schema for check_requirements tool
type RequirementsResult struct {
	Requirements []RequirementStatus `json:"requirements" jsonschema:"description=List of requirements and their status"`
	AllMet       bool                `json:"allMet" jsonschema:"description=Whether all requirements are satisfied"`
}

// RequirementStatus represents individual requirement check result
type RequirementStatus struct {
	Name           string `json:"name" jsonschema:"description=Requirement name (e.g. node, python, docker)"`
	Required       string `json:"required" jsonschema:"description=Required version"`
	Installed      string `json:"installed,omitempty" jsonschema:"description=Installed version if found"`
	Met            bool   `json:"met" jsonschema:"description=Whether requirement is satisfied"`
	InstallCommand string `json:"installCommand,omitempty" jsonschema:"description=Command to install if missing"`
}
