package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

// NewConfigCommand creates the config command.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [service]",
		Short: "Show the effective configuration for each service",
		Long: `Print the configuration azd app resolved from azure.yaml for each service.

For every service this shows the host, the effective service type (explicit or
inferred), language, project path, run command, image, ports, and dependencies
(uses), plus which optional blocks are configured (docker, healthcheck, restart,
resources, logs, local, azure).

Run without arguments to print every service, or pass a service name to print
just that one. Use --output json for a machine-readable object keyed by service
name.

Examples:
  # Configuration for every service
  azd app config

  # Configuration for a single service
  azd app config api

  # JSON object keyed by service name
  azd app config --output json`,
		SilenceUsage:      true,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runConfig,
		ValidArgsFunction: completeServiceArgs,
	}

	return cmd
}

// serviceConfig is the resolved configuration for a single service. In JSON
// output it is the value of a map keyed by service name, so the name itself is
// not repeated here.
type serviceConfig struct {
	Host       string   `json:"host,omitempty"`
	Type       string   `json:"type"`
	TypeSource string   `json:"typeSource"` // "explicit" when set in azure.yaml, otherwise "inferred"
	Language   string   `json:"language,omitempty"`
	Project    string   `json:"project,omitempty"`
	Command    string   `json:"command,omitempty"`
	Image      string   `json:"image,omitempty"`
	Ports      []string `json:"ports,omitempty"`
	Uses       []string `json:"uses,omitempty"`
	Blocks     []string `json:"blocks,omitempty"` // optional blocks configured on this service
}

func runConfig(_ *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYaml, err := service.ParseAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("failed to load azure.yaml: %w", err)
	}

	configs, order, err := selectServiceConfigs(azureYaml, args)
	if err != nil {
		return err
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(configs)
	}

	if len(order) == 0 {
		cliout.Info("No services are defined in azure.yaml")
		return nil
	}

	cliout.CommandHeader("config", "Effective service configuration")
	for i, name := range order {
		if i > 0 {
			cliout.Newline()
		}
		printServiceConfig(name, configs[name])
	}
	return nil
}

// selectServiceConfigs resolves the configs to display. With no arguments it
// returns every service in name order; with a single argument it returns just
// that service, or an error listing the available services when the name is
// unknown. The returned slice is the display order for text output.
func selectServiceConfigs(azureYaml *service.AzureYaml, args []string) (map[string]serviceConfig, []string, error) {
	names := make([]string, 0, len(azureYaml.Services))
	for name := range azureYaml.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	order := names
	if len(args) == 1 {
		name := args[0]
		if _, ok := azureYaml.Services[name]; !ok {
			if len(names) == 0 {
				return nil, nil, fmt.Errorf("service %q not found. No services are defined in azure.yaml", name)
			}
			return nil, nil, fmt.Errorf("service %q not found. Available services: %s", name, strings.Join(names, ", "))
		}
		order = []string{name}
	}

	configs := make(map[string]serviceConfig, len(order))
	for _, name := range order {
		configs[name] = buildServiceConfig(azureYaml.Services[name])
	}
	return configs, order, nil
}

// buildServiceConfig captures the resolved configuration for a service, using
// GetServiceType to report the effective type when it is not set explicitly.
func buildServiceConfig(svc service.Service) serviceConfig {
	cfg := serviceConfig{
		Host:     svc.Host,
		Type:     svc.GetServiceType(),
		Language: svc.Language,
		Project:  svc.Project,
		Command:  svc.Command,
		Image:    svc.Image,
		Ports:    svc.Ports,
		Uses:     svc.Uses,
		Blocks:   configuredBlocks(svc),
	}
	if svc.Type != "" {
		cfg.TypeSource = "explicit"
	} else {
		cfg.TypeSource = "inferred"
	}
	return cfg
}

// configuredBlocks returns the names of the optional service blocks that are set
// in azure.yaml, in a stable order.
func configuredBlocks(svc service.Service) []string {
	var blocks []string
	if svc.Docker != nil {
		blocks = append(blocks, "docker")
	}
	if svc.Healthcheck != nil {
		blocks = append(blocks, "healthcheck")
	}
	if svc.Restart != nil {
		blocks = append(blocks, "restart")
	}
	if svc.Resources != nil {
		blocks = append(blocks, "resources")
	}
	if svc.Logs != nil {
		blocks = append(blocks, "logs")
	}
	if svc.Local != nil {
		blocks = append(blocks, "local")
	}
	if svc.Azure != nil {
		blocks = append(blocks, "azure")
	}
	return blocks
}

// printServiceConfig writes one service's resolved configuration as aligned
// label lines, omitting fields that are not set.
func printServiceConfig(name string, cfg serviceConfig) {
	cliout.Info("%s", name)
	if cfg.Host != "" {
		cliout.Label("host", cfg.Host)
	}
	cliout.Label("type", fmt.Sprintf("%s (%s)", cfg.Type, cfg.TypeSource))
	if cfg.Language != "" {
		cliout.Label("language", cfg.Language)
	}
	if cfg.Project != "" {
		cliout.Label("project", cfg.Project)
	}
	if cfg.Command != "" {
		cliout.Label("command", cfg.Command)
	}
	if cfg.Image != "" {
		cliout.Label("image", cfg.Image)
	}
	if len(cfg.Ports) > 0 {
		cliout.Label("ports", strings.Join(cfg.Ports, ", "))
	}
	if len(cfg.Uses) > 0 {
		cliout.Label("uses", strings.Join(cfg.Uses, ", "))
	}
	if len(cfg.Blocks) > 0 {
		cliout.Label("configured", strings.Join(cfg.Blocks, ", "))
	}
}
