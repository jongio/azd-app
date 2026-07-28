package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-app/cli/src/internal/trust"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/yamlutil"

	"github.com/spf13/cobra"
)

const (
	runtimeModeAzd    = "azd"
	runtimeModeAspire = "aspire"
)

var (
	runServiceFilter     string
	runScale             []string
	runEnvFile           string
	runEnvInline         []string
	runVerbose           bool
	runDryRun            bool
	runDetach            bool
	runRuntime           string
	runWeb               bool
	runRestartContainers bool
	runForce             bool
	runTrust             bool
	runNoTiming          bool
	runSkipSecretScan    bool
	runSkipExposureCheck bool
)

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	commandOrchestrator := newCommandOrchestrator()

	cmd := &cobra.Command{
		Use:          "run",
		Short:        "Run the development environment (services from azure.yaml, Aspire, pnpm, or docker compose)",
		Long:         `Automatically detects and runs services defined in azure.yaml, or falls back to: Aspire (AppHost.cs), pnpm dev/start scripts, or docker compose from package.json`,
		SilenceUsage: true, // Don't print usage on errors - it makes error messages hard to read
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithServices(cmd.Context(), commandOrchestrator, cmd, args)
		},
	}

	// Add flags for service orchestration
	cmd.Flags().StringVarP(&runServiceFilter, "service", "s", "", "Run specific service(s) only (comma-separated)")
	cmd.Flags().StringSliceVar(&runScale, "scale", nil, "Run multiple instances of a service, e.g. --scale worker=3 (repeatable, comma-separated)")
	cmd.Flags().StringVar(&runEnvFile, "env-file", "", "Load environment variables from .env file")
	cmd.Flags().StringArrayVar(&runEnvInline, "env", nil, "Set an environment variable inline as KEY=VALUE (repeatable, overrides --env-file)")
	cmd.Flags().BoolVarP(&runVerbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Show what would be run without starting services")
	cmd.Flags().StringVar(&runRuntime, "runtime", runtimeModeAzd, "Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run)")
	cmd.Flags().BoolVarP(&runWeb, "web", "w", false, "Open dashboard in browser")
	cmd.Flags().BoolVar(&runRestartContainers, "restart-containers", false, "Restart containers even if they are already running")
	cmd.Flags().BoolVar(&runForce, "force", false, "Force clean dependency reinstall and auto-resolve port conflicts without prompting")
	cmd.Flags().BoolVarP(&runTrust, "trust", "y", false, "Trust this workspace for code execution and remember the decision")
	cmd.Flags().BoolVar(&runDetach, "detach", false, "Run the app in the background and return to the shell")
	cmd.Flags().BoolVar(&runNoTiming, "no-timing", false, "Hide the per-service startup timing summary shown after services are ready")
	cmd.Flags().BoolVar(&runSkipSecretScan, "skip-secret-scan", false, "Skip the advisory scan for hardcoded secrets in tracked config")
	cmd.Flags().BoolVar(&runSkipExposureCheck, "skip-exposure-check", false, "Skip the warning shown when a service binds to all network interfaces")

	registerServiceFlagCompletion(cmd, "service")

	return cmd
}

// runWithServices runs services from azure.yaml.
func runWithServices(ctx context.Context, commandOrchestrator *orchestrator.Orchestrator, _ *cobra.Command, _ []string) error {
	cliout.CommandHeader("run", "Run the development environment")
	if err := validateRuntimeMode(runRuntime); err != nil {
		return err
	}

	// Locate azure.yaml before spawning any subprocesses so we can compute
	// the project root for the trust check (CWE-78/94).
	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}

	// Trust gate: user must explicitly consent before any command defined in
	// azure.yaml is allowed to execute on the host.  This must happen before
	// commandOrchestrator.Run (which installs deps) and before service startup.
	if err := ensureWorkspaceTrusted(azureYamlPath); err != nil {
		return err
	}

	detachedResult, detached, err := maybeStartDetachedRun(filepath.Dir(azureYamlPath))
	if err != nil {
		return err
	}
	if detached {
		return printDetachedStartResult(detachedResult)
	}

	// Set deps options if --force specified
	if runForce {
		opts := GetDepsOptions()
		opts.Force = true
		setDepsOptions(opts)
	}

	// Execute dependencies first (reqs -> deps -> run)
	// The orchestrator automatically sets orchestrated mode for dependencies
	if err := commandOrchestrator.Run("run"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	return runServicesFromAzureYaml(ctx, azureYamlPath, runRuntime)
}

// ensureWorkspaceTrusted enforces the workspace trust gate before any
// subprocess spawning.  It short-circuits when the --trust flag or the
// AZD_APP_TRUST=1 environment variable is set (CI / scripted use-cases).
// In an interactive terminal the user is prompted for consent.  In a
// non-interactive environment (pipes, MCP, CI without the flag) an error
// is returned so callers can surface a clear message.
func ensureWorkspaceTrusted(azureYamlPath string) error {
	projectDir := filepath.Dir(azureYamlPath)

	store, err := trust.NewTrustStore()
	if err != nil {
		return fmt.Errorf("failed to initialise trust store: %w", err)
	}

	// --trust flag or AZD_APP_TRUST=1 → record trust and proceed.
	if runTrust || os.Getenv("AZD_APP_TRUST") == "1" {
		if err := store.TrustWorkspace(projectDir); err != nil {
			return fmt.Errorf("failed to record workspace trust: %w", err)
		}
		return nil
	}

	trusted, trustErr := store.IsWorkspaceTrusted(projectDir)
	if trusted {
		return nil
	}

	// Propagate internal errors (I/O failures, parse errors) immediately.
	if trustErr != nil && !errors.Is(trustErr, trust.ErrHashChanged) {
		return fmt.Errorf("failed to check workspace trust: %w", trustErr)
	}

	// Non-interactive: require explicit opt-in rather than trying to prompt.
	if !isInteractiveTerminal() {
		if errors.Is(trustErr, trust.ErrHashChanged) {
			return fmt.Errorf(
				"workspace at %s is no longer trusted: azure.yaml has changed; "+
					"re-run with --trust flag or set AZD_APP_TRUST=1 to trust the updated file",
				projectDir,
			)
		}
		return fmt.Errorf(
			"workspace at %s is not trusted; "+
				"re-run with --trust flag or set AZD_APP_TRUST=1 to trust this workspace",
			projectDir,
		)
	}

	// Interactive: present the security notice and prompt for consent.
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  ⚠️  Security Notice\n")
	fmt.Fprintf(os.Stderr, "  Running 'azd app run' will execute commands defined in:\n")
	fmt.Fprintf(os.Stderr, "    %s\n", azureYamlPath)
	fmt.Fprintf(os.Stderr, "  Only trust workspaces you own or have reviewed.\n\n")

	if errors.Is(trustErr, trust.ErrHashChanged) {
		fmt.Fprintf(os.Stderr, "  azure.yaml has changed since this workspace was last trusted.\n\n")
	}

	fmt.Fprintf(os.Stderr, "  Trust workspace at %s for code execution? (Y/n): ", projectDir)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "n" || response == "no" {
		return fmt.Errorf("workspace not trusted — exiting without executing any commands")
	}

	if err := store.TrustWorkspace(projectDir); err != nil {
		return fmt.Errorf("failed to record workspace trust: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  ✓ Workspace trusted\n\n")
	return nil
}

// isInteractiveTerminal reports whether os.Stdin is connected to a character
// device (i.e., an interactive terminal as opposed to a pipe or file).
// Returns false on any error so callers fall back to non-interactive behaviour.
func isInteractiveTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// validateRuntimeMode validates the runtime mode parameter.
func validateRuntimeMode(mode string) error {
	if mode != runtimeModeAzd && mode != runtimeModeAspire {
		return fmt.Errorf("invalid --runtime value: %s (must be '%s' or '%s')", mode, runtimeModeAzd, runtimeModeAspire)
	}
	return nil
}

// findAzureYaml locates the azure.yaml file.
func findAzureYaml() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return "", fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath == "" {
		return "", fmt.Errorf("azure.yaml not found - create one with 'services' section to define your development environment")
	}

	return azureYamlPath, nil
}

// runServicesFromAzureYaml orchestrates services defined in azure.yaml.
func runServicesFromAzureYaml(ctx context.Context, azureYamlPath string, runtimeMode string) error {
	azureYamlDir := filepath.Dir(azureYamlPath)

	// Aspire mode: run AppHost directly
	if runtimeMode == runtimeModeAspire {
		return runAspireMode(ctx, azureYamlDir)
	}

	// AZD mode: orchestrate services individually
	return runAzdMode(ctx, azureYamlPath, azureYamlDir)
}

// runAzdMode runs services in azd mode with individual service orchestration.
func runAzdMode(ctx context.Context, azureYamlPath, azureYamlDir string) error {
	// Parse azure.yaml
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Install any project-specific log redaction rules before services start
	// streaming, so both the live console and support bundles honor them.
	registerLogRedaction(azureYaml)

	// REMOVED: initializeAzureLogBuffer call - deprecated v1
	// Azure logs are now fetched on-demand via /api/azure/logs endpoint

	// Execute prerun hook before starting services
	if err = executePrerunHook(ctx, azureYaml, azureYamlDir); err != nil {
		return err
	}

	// Check if there are services defined
	if !service.HasServices(azureYaml) {
		return showNoServicesMessage()
	}

	// Filter and detect services
	services := filterServices(azureYaml)
	if len(services) == 0 {
		return fmt.Errorf("no services match filter: %s", runServiceFilter)
	}

	// Advisory security preflight: warn about literal secrets in tracked config.
	runSecretScan(azureYaml, services, azureYamlDir)

	// Advisory: warn when a service binds to every network interface.
	runExposureCheck(services, azureYamlDir, azureYaml)

	// Propagate --force to port manager so port conflicts auto-resolve without prompting
	if runForce {
		portmanager.GetPortManager(azureYamlDir).SetForceMode(true)
	}

	runtimes, err := detectServiceRuntimes(services, azureYamlDir, runtimeModeAzd)
	if err != nil {
		return err
	}

	scale, err := parseScaleFlags(runScale)
	if err != nil {
		return fmt.Errorf("invalid --scale value: %w", err)
	}

	pm := portmanager.GetPortManager(azureYamlDir)
	alloc := func(instanceName string, avoid map[int]bool) (int, error) {
		port, _, allocErr := pm.AssignPort(instanceName, 0, false)
		if allocErr != nil {
			return 0, fmt.Errorf("failed to assign port for %s: %w", instanceName, allocErr)
		}

		avoid[port] = true
		return port, nil
	}

	expandedRuntimes, expandedServices, err := service.ExpandScaledRuntimes(runtimes, azureYaml.Services, scale, alloc)
	if err != nil {
		return fmt.Errorf("failed to expand scaled services: %w", err)
	}

	runtimes = expandedRuntimes
	azureYaml.Services = expandedServices

	// Dry-run mode: show what would be executed
	if runDryRun {
		return showDryRun(runtimes)
	}

	// Execute and monitor services
	return executeAndMonitorServices(ctx, runtimes, azureYamlDir, azureYaml)
}

// showNoServicesMessage displays a message when no services are defined.
func showNoServicesMessage() error {
	cliout.Info("No services defined in azure.yaml")
	cliout.Item("Add a 'services' section to azure.yaml to use service orchestration")
	cliout.Item("or remove azure.yaml to use auto-detection (Aspire, pnpm, docker-compose)")
	return nil
}

// filterServices applies service filtering based on --service flag.
func filterServices(azureYaml *service.AzureYaml) map[string]service.Service {
	if runServiceFilter == "" {
		return azureYaml.Services
	}
	filterList := strings.Split(runServiceFilter, ",")
	return service.FilterServices(azureYaml, filterList)
}

// detectServiceRuntimes detects runtime information for all services.
//
// CONCURRENCY: This function is NOT thread-safe and must be called sequentially.
// The usedPorts map is shared across all service detections to prevent port conflicts,
// but it is not protected by a mutex. If parallel detection is needed in the future,
// the usedPorts map must be protected with a sync.Mutex.
//
// Current usage: Called sequentially from runAzdMode, so no protection needed.
func detectServiceRuntimes(services map[string]service.Service, azureYamlDir, runtimeMode string) ([]*service.ServiceRuntime, error) {
	usedPorts := make(map[int]bool) // WARNING: Not thread-safe, do not call this function concurrently
	runtimes := make([]*service.ServiceRuntime, 0, len(services))

	// Find azure.yaml path for updates
	azureYamlPath := filepath.Join(azureYamlDir, "azure.yaml")

	for name, svc := range services {
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts, azureYamlDir, runtimeMode)
		if err != nil {
			return nil, fmt.Errorf("failed to detect runtime for service %s: %w", name, err)
		}
		usedPorts[runtime.Port] = true

		// If we auto-assigned a port and user wants to save it, update azure.yaml
		if runtime.ShouldUpdateAzureYaml {
			if err := yamlutil.UpdateServicePort(azureYamlPath, name, runtime.Port); err != nil {
				cliout.Warning("Failed to update azure.yaml for service %s: %v", name, err)
				cliout.Info("   Please manually add 'ports: [\"%d\"]' to service '%s' in azure.yaml", runtime.Port, name)
			} else {
				cliout.Success("Updated azure.yaml: Added ports: [\"%d\"] for service '%s'", runtime.Port, name)
			}
		}

		runtimes = append(runtimes, runtime)
	}

	return runtimes, nil
}

func loadRunEnvironmentVariables() (map[string]string, map[string]string, error) {
	envVars := make(map[string]string)

	if runEnvFile != "" {
		loaded, err := service.LoadDotEnv(runEnvFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load env file: %w", err)
		}
		envVars = loaded
	}

	inlineEnvVars := make(map[string]string)
	if err := mergeInlineEnv(inlineEnvVars, runEnvInline); err != nil {
		return nil, nil, err
	}

	return envVars, inlineEnvVars, nil
}

// mergeInlineEnv parses KEY=VALUE entries and merges them into envVars,
// overriding any existing keys. Each entry must contain '=' and a non-empty
// key. The value may be empty or contain additional '=' characters.
func mergeInlineEnv(envVars map[string]string, entries []string) error {
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return fmt.Errorf("invalid --env value %q: expected KEY=VALUE", entry)
		}
		envVars[key] = value
	}
	return nil
}

// buildServiceSummaries composes URL summaries for display, preferring rich data from serviceinfo.
func buildServiceSummaries(projectDir string, azureYaml *service.AzureYaml, processes map[string]*service.ServiceProcess) []service.ServiceURLSummary {
	services, err := serviceinfo.GetServiceInfo(projectDir)
	if err == nil && len(services) > 0 {
		return convertServiceInfoToSummaries(services)
	}

	if err != nil {
		cliout.Warning("Failed to get service info: %v", err)
	}

	return buildFallbackSummaries(azureYaml, processes)
}

// convertServiceInfoToSummaries converts serviceinfo results into logger-friendly summaries.
func convertServiceInfoToSummaries(services []*serviceinfo.ServiceInfo) []service.ServiceURLSummary {
	summaries := make([]service.ServiceURLSummary, 0, len(services))

	for _, svc := range services {
		summary := service.ServiceURLSummary{Name: svc.Name}

		if svc.Local != nil {
			summary.LocalURL = svc.Local.URL
			if summary.LocalURL == "" && svc.Local.Port > 0 {
				summary.LocalURL = fmt.Sprintf("http://localhost:%d", svc.Local.Port)
			}
			summary.LocalCustomURL = svc.Local.CustomURL
		}

		if svc.Azure != nil {
			summary.AzureURL = svc.Azure.URL
			summary.AzureCustomURL = svc.Azure.CustomURL
			if svc.Azure.CustomDomain != "" {
				if svc.Azure.CustomDomainSource != "" {
					summary.AzureCustomDomain = fmt.Sprintf("%s (%s)", svc.Azure.CustomDomain, svc.Azure.CustomDomainSource)
				} else {
					summary.AzureCustomDomain = svc.Azure.CustomDomain
				}
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

type customEndpointConfig struct {
	localCustomURL          string
	azureCustomURL          string
	azureCustomDomain       string
	azureCustomDomainSource string
}

// buildFallbackSummaries is used when serviceinfo data is unavailable; it combines runtime ports with azure.yaml config.
func buildFallbackSummaries(azureYaml *service.AzureYaml, processes map[string]*service.ServiceProcess) []service.ServiceURLSummary {
	if azureYaml == nil {
		return nil
	}

	urls := service.GetServiceURLs(processes)
	customConfig := extractCustomConfig(azureYaml)

	names := make([]string, 0, len(azureYaml.Services))
	for name := range azureYaml.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]service.ServiceURLSummary, 0, len(names))
	for _, name := range names {
		normalized := strings.ToLower(name)
		summary := service.ServiceURLSummary{Name: name}

		if url, ok := urls[name]; ok {
			summary.LocalURL = url
		} else if url, ok := urls[normalized]; ok {
			summary.LocalURL = url
		}

		if cfg, ok := customConfig[normalized]; ok {
			summary.LocalCustomURL = cfg.localCustomURL
			summary.AzureCustomURL = cfg.azureCustomURL
			if cfg.azureCustomDomain != "" {
				if cfg.azureCustomDomainSource != "" {
					summary.AzureCustomDomain = fmt.Sprintf("%s (%s)", cfg.azureCustomDomain, cfg.azureCustomDomainSource)
				} else {
					summary.AzureCustomDomain = cfg.azureCustomDomain
				}
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

// extractCustomConfig captures custom URL/domain configuration from azure.yaml for summary display.
func extractCustomConfig(azureYaml *service.AzureYaml) map[string]customEndpointConfig {
	config := make(map[string]customEndpointConfig)
	if azureYaml == nil {
		return config
	}

	for name, svc := range azureYaml.Services {
		normalized := strings.ToLower(name)
		cfg := customEndpointConfig{}

		if svc.Local != nil && svc.Local.CustomURL != "" {
			cfg.localCustomURL = svc.Local.CustomURL
		}

		if svc.Azure != nil {
			if svc.Azure.CustomURL != "" {
				cfg.azureCustomURL = svc.Azure.CustomURL
			}
			if svc.Azure.CustomDomain != "" {
				cfg.azureCustomDomain = svc.Azure.CustomDomain
				cfg.azureCustomDomainSource = svc.Azure.CustomDomainSource
			}
		}

		// Include deprecated root-level url as a custom Azure URL for backward compatibility
		if cfg.azureCustomURL == "" && svc.URL != "" {
			cfg.azureCustomURL = svc.URL
		}

		config[normalized] = cfg
	}

	return config
}

// showDryRun displays what would be executed without starting services.
func showDryRun(runtimes []*service.ServiceRuntime) error {
	cliout.Section("🔍", "Dry-run mode: Showing execution plan")

	for _, runtime := range runtimes {
		cliout.Newline()
		cliout.Info("%s", runtime.Name)
		cliout.Label("Language", runtime.Language)
		cliout.Label("Framework", runtime.Framework)
		cliout.Label("Port", fmt.Sprintf("%d", runtime.Port))
		cliout.Label("Directory", runtime.WorkingDir)
		cliout.Label("Command", fmt.Sprintf("%s %v", runtime.Command, runtime.Args))
	}

	return nil
}

// REMOVED: initializeAzureLogBuffer - deprecated v1 polling/WebSocket implementation
// Azure logs are now fetched on-demand via /api/azure/logs endpoint (v2 request/response model)
