package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

// NewPortsCommand creates the ports command.
func NewPortsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "List the host ports each service binds",
		Long: `List the host port each service binds, resolved from azure.yaml.

For every service this shows each port binding as "host -> container/protocol".
An explicit host port is shown as its number; a port left for the tool to assign
is shown as "auto". When two bindings claim the same explicit host port the
command reports the conflict and exits non-zero, which makes it useful in a
preflight check.

Examples:
  # Ports for every service
  azd app ports

  # JSON object keyed by service name
  azd app ports --output json`,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE:         runPorts,
	}

	return cmd
}

// portBinding describes one host-to-container port binding for a service.
type portBinding struct {
	Host      string `json:"host"`               // explicit host port number, or "auto"
	HostPort  int    `json:"hostPort,omitempty"` // numeric host port when explicit
	Container int    `json:"container,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Conflict  bool   `json:"conflict,omitempty"` // true when this explicit host port is claimed more than once
}

// servicePorts is the set of port bindings for a single service.
type servicePorts struct {
	Ports []portBinding `json:"ports"`
}

// portConflict records an explicit host port claimed by more than one binding.
type portConflict struct {
	Port   int
	Owners []string
}

// portReport is the resolved port view for every service.
type portReport struct {
	services  map[string]servicePorts
	order     []string
	conflicts []portConflict
}

func runPorts(_ *cobra.Command, _ []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYaml, err := service.ParseAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("failed to load azure.yaml: %w", err)
	}

	report := collectPortReport(azureYaml)

	if cliout.IsJSON() {
		if err := cliout.PrintJSON(report.services); err != nil {
			return err
		}
	} else {
		printPortReport(report)
	}

	if len(report.conflicts) > 0 {
		return fmt.Errorf("duplicate explicit host port(s): %s", conflictSummary(report.conflicts))
	}
	return nil
}

// collectPortReport resolves the port bindings for every service and flags any
// explicit host port that is claimed by more than one binding.
func collectPortReport(azureYaml *service.AzureYaml) portReport {
	names := sortedServiceNames(azureYaml.Services)
	services := make(map[string]servicePorts, len(names))
	owners := make(map[int][]string) // explicit host port -> service names that bind it

	for _, name := range names {
		svc := azureYaml.Services[name]
		mappings, _ := svc.GetPortMappings()
		bindings := make([]portBinding, 0, len(mappings))
		for _, m := range mappings {
			b := portBinding{
				Container: m.ContainerPort,
				Protocol:  protocolOrDefault(m.Protocol),
			}
			if m.HostPort > 0 {
				b.HostPort = m.HostPort
				b.Host = strconv.Itoa(m.HostPort)
				owners[m.HostPort] = append(owners[m.HostPort], name)
			} else {
				b.Host = "auto"
			}
			bindings = append(bindings, b)
		}
		services[name] = servicePorts{Ports: bindings}
	}

	conflictPorts := make(map[int]bool)
	var conflicts []portConflict
	for port, own := range owners {
		if len(own) > 1 {
			conflictPorts[port] = true
			conflicts = append(conflicts, portConflict{Port: port, Owners: dedupeStrings(own)})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Port < conflicts[j].Port })

	if len(conflictPorts) > 0 {
		for name, sp := range services {
			for i := range sp.Ports {
				if sp.Ports[i].HostPort > 0 && conflictPorts[sp.Ports[i].HostPort] {
					sp.Ports[i].Conflict = true
				}
			}
			services[name] = sp
		}
	}

	return portReport{services: services, order: names, conflicts: conflicts}
}

// printPortReport writes the port bindings for each service, then a warning line
// per conflicting host port.
func printPortReport(report portReport) {
	if len(report.order) == 0 {
		cliout.Info("No services are defined in azure.yaml")
		return
	}

	cliout.CommandHeader("ports", "Configured host ports")
	for i, name := range report.order {
		if i > 0 {
			cliout.Newline()
		}
		cliout.Info("%s", name)
		sp := report.services[name]
		if len(sp.Ports) == 0 {
			cliout.Item("no ports configured")
			continue
		}
		for _, b := range sp.Ports {
			line := fmt.Sprintf("%s -> %d/%s", b.Host, b.Container, b.Protocol)
			if b.Conflict {
				line += "  (conflict)"
			}
			cliout.Label("port", line)
		}
	}

	if len(report.conflicts) > 0 {
		cliout.Newline()
		for _, c := range report.conflicts {
			cliout.Warning("Host port %d is bound by more than one service: %s", c.Port, strings.Join(c.Owners, ", "))
		}
	}
}

// conflictSummary renders the conflicts for the returned error, e.g.
// "3000 (web, worker); 8080 (api, api2)".
func conflictSummary(conflicts []portConflict) string {
	parts := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		parts = append(parts, fmt.Sprintf("%d (%s)", c.Port, strings.Join(c.Owners, ", ")))
	}
	return strings.Join(parts, "; ")
}

// protocolOrDefault returns the protocol, defaulting to "tcp" when unset.
func protocolOrDefault(protocol string) string {
	if protocol == "" {
		return "tcp"
	}
	return protocol
}

// dedupeStrings returns the input with duplicates removed, preserving order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
