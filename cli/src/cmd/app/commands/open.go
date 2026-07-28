package commands

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var (
	openPath  string
	openPrint bool
	openURL   = browser.OpenURL
)

// NewOpenCommand creates the open command.
func NewOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open <service>",
		Short: "Open a service URL in the default browser",
		Long: `Resolve a service URL from azure.yaml or the running app state and open it in the default browser.

Use --path to append a route such as /health. Use --print to write the URL without launching a browser.`,
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		RunE:              runOpen,
		ValidArgsFunction: completeServiceArgs,
	}
	cmd.Flags().StringVar(&openPath, "path", "", "Path to append to the service URL")
	cmd.Flags().BoolVar(&openPrint, "print", false, "Print the URL without opening a browser")
	return cmd
}

func runOpen(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	resolved, err := resolveOpenServiceURL(cwd, args[0], openPath)
	if err != nil {
		return err
	}
	if openPrint {
		if _, writeErr := fmt.Fprintln(cmd.OutOrStdout(), resolved); writeErr != nil {
			return fmt.Errorf("failed to write service URL: %w", writeErr)
		}
		return nil
	}
	if err := openURL(resolved); err != nil {
		return fmt.Errorf("failed to open %s: %w", resolved, err)
	}
	return nil
}

// resolveOpenServiceURL resolves the browser URL for serviceName. The running
// app state wins, because it knows the port a service actually bound to; the
// azure.yaml definition is the fallback for services that are not running.
func resolveOpenServiceURL(projectDir, serviceName, extraPath string) (string, error) {
	infos, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to load service information: %w", err)
	}

	names := make([]string, 0, len(infos))
	found := false
	for _, info := range infos {
		if info == nil {
			continue
		}
		names = append(names, info.Name)
		if !strings.EqualFold(info.Name, serviceName) {
			continue
		}
		found = true
		if base := bestOpenURL(info); base != "" {
			return joinOpenURLPath(base, extraPath)
		}
	}

	// GetServiceInfo suppresses azure.yaml errors, so parse it again here to
	// surface a malformed file instead of masking it as "service not found".
	azureYaml, parseErr := parseOpenAzureYaml(projectDir)
	if parseErr != nil {
		return "", parseErr
	}
	if azureYaml != nil {
		for name, svc := range azureYaml.Services {
			if !containsName(names, name) {
				names = append(names, name)
			}
			if !strings.EqualFold(name, serviceName) {
				continue
			}
			found = true
			if base := openURLFromService(svc); base != "" {
				return joinOpenURLPath(base, extraPath)
			}
		}
	}

	if found {
		return "", fmt.Errorf(
			"service %q has no known URL. Start it with 'azd app run', or set local.customUrl or ports in azure.yaml",
			serviceName)
	}

	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("service %q not found. No services are defined in azure.yaml", serviceName)
	}
	return "", fmt.Errorf("service %q not found. Available services: %s", serviceName, strings.Join(names, ", "))
}

// parseOpenAzureYaml parses azure.yaml, returning (nil, nil) when the file does
// not exist so a project without one still reports "service not found" rather
// than a parse failure. Any other failure is returned to the caller.
func parseOpenAzureYaml(projectDir string) (*service.AzureYaml, error) {
	azureYamlPath, findErr := detector.FindAzureYaml(projectDir)
	if findErr != nil {
		return nil, fmt.Errorf("failed to locate azure.yaml: %w", findErr)
	}
	if azureYamlPath == "" {
		return nil, nil
	}
	azureYaml, parseErr := service.ParseAzureYaml(projectDir)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", azureYamlPath, parseErr)
	}
	return azureYaml, nil
}

func bestOpenURL(info *serviceinfo.ServiceInfo) string {
	if info.Local != nil {
		if info.Local.CustomURL != "" {
			return info.Local.CustomURL
		}
		if info.Local.URL != "" {
			return info.Local.URL
		}
		if info.Local.Port > 0 {
			return fmt.Sprintf("http://localhost:%d", info.Local.Port)
		}
	}
	return ""
}

func openURLFromService(svc service.Service) string {
	if svc.Local != nil && svc.Local.CustomURL != "" {
		return svc.Local.CustomURL
	}
	if hostPort, ok := publishedHostPort(svc); ok {
		return fmt.Sprintf("http://localhost:%d", hostPort)
	}
	return ""
}

// publishedHostPort returns the first TCP port a service actually publishes on
// the host. It defers to the canonical port parser so Docker container-only
// ports (e.g. "80", whose host port is auto-assigned at runtime), bind IPs,
// IPv6 binds, protocol suffixes and malformed specs are all handled the same way
// the rest of the CLI handles them. A mapping without a published host port
// yields no URL rather than a URL pointing at the container port.
func publishedHostPort(svc service.Service) (int, bool) {
	mappings, _ := svc.GetPortMappings()
	for _, mapping := range mappings {
		if mapping.HostPort <= 0 {
			continue
		}
		if mapping.Protocol != "" && !strings.EqualFold(mapping.Protocol, "tcp") {
			continue
		}
		return mapping.HostPort, true
	}
	return 0, false
}

func joinOpenURLPath(base, extraPath string) (string, error) {
	if strings.TrimSpace(extraPath) == "" {
		return base, nil
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid service URL %q: %w", base, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid service URL %q: missing scheme or host", base)
	}
	joined := path.Join(parsed.Path, extraPath)
	if strings.HasSuffix(extraPath, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	parsed.Path = joined
	// RawPath is the encoding of the previous Path, so it is stale now. Clear it
	// to keep Path and RawPath consistent and force String() to re-encode.
	parsed.RawPath = ""
	return parsed.String(), nil
}

func containsName(names []string, name string) bool {
	for _, existing := range names {
		if strings.EqualFold(existing, name) {
			return true
		}
	}
	return false
}
