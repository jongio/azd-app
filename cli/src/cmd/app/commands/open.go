package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

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

func runOpen(_ *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	resolved, err := resolveOpenServiceURL(context.Background(), cwd, args[0], openPath)
	if err != nil {
		return err
	}
	if openPrint {
		fmt.Println(resolved)
		return nil
	}
	if err := openURL(resolved); err != nil {
		return fmt.Errorf("failed to open %s: %w", resolved, err)
	}
	return nil
}

func resolveOpenServiceURL(_ context.Context, projectDir, serviceName, extraPath string) (string, error) {
	infos, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to load service information: %w", err)
	}

	var names []string
	foundService := false
	for _, info := range infos {
		if info == nil {
			continue
		}
		names = append(names, info.Name)
		if strings.EqualFold(info.Name, serviceName) {
			foundService = true
			if base := bestOpenURL(info); base != "" {
				return joinOpenURLPath(base, extraPath)
			}
			break
		}
	}

	azureYaml, parseErr := service.ParseAzureYaml(projectDir)
	if parseErr == nil {
		for name, svc := range azureYaml.Services {
			if !containsName(names, name) {
				names = append(names, name)
			}
			if strings.EqualFold(name, serviceName) {
				if base := openURLFromService(svc); base != "" {
					return joinOpenURLPath(base, extraPath)
				}
				return "", fmt.Errorf("service %q has no known URL. Set local.customUrl or ports in azure.yaml", serviceName)
			}
		}
	}
	if foundService {
		return "", fmt.Errorf("service %q has no known URL. Start it with 'azd app run' or set local.customUrl or ports in azure.yaml", serviceName)
	}

	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("service %q not found. No services are defined in azure.yaml", serviceName)
	}
	return "", fmt.Errorf("service %q not found. Available services: %s", serviceName, strings.Join(names, ", "))
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
	for _, port := range svc.Ports {
		if hostPort := hostPortFromMapping(port); hostPort != "" {
			return "http://localhost:" + hostPort
		}
	}
	return ""
}

func hostPortFromMapping(mapping string) string {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return ""
	}
	parts := strings.Split(mapping, ":")
	if len(parts) == 1 {
		return trimPortProtocol(parts[0])
	}
	return trimPortProtocol(parts[len(parts)-2])
}

func trimPortProtocol(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "/tcp")
	value = strings.TrimSuffix(value, "/udp")
	return value
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
	parsed.Path = path.Join(parsed.Path, extraPath)
	if strings.HasSuffix(extraPath, "/") && !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
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
