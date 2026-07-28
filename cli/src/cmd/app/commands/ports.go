package commands

import (
	"fmt"
	"net"
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
	BindIP    string `json:"bindIP,omitempty"`
	Container int    `json:"container,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Conflict  bool   `json:"conflict,omitempty"` // true when this binding overlaps another explicit host binding
}

// servicePorts is the set of port bindings for a single service.
type servicePorts struct {
	Ports []portBinding `json:"ports"`
}

// portConflict records explicit host bindings that overlap.
type portConflict struct {
	BindIP   string   `json:"bindIP,omitempty"`
	HostPort int      `json:"hostPort"`
	Protocol string   `json:"protocol"`
	Owners   []string `json:"owners"`
}

// portJSONReport is the JSON payload for the ports command.
type portJSONReport struct {
	Services  map[string]servicePorts `json:"services"`
	Conflicts []portConflict          `json:"conflicts"`
}

type portBindingKey struct {
	BindIP   string
	HostPort int
	Protocol string
}

type collectedPortBinding struct {
	Service string
	Index   int
	Key     portBindingKey
}

// portReport is the resolved port view for every service.
type portReport struct {
	services  map[string]servicePorts
	order     []string
	conflicts []portConflict
}

func runPorts(cmd *cobra.Command, _ []string) error {
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
		if err := cliout.PrintJSON(newPortJSONReport(report)); err != nil {
			return err
		}
	} else {
		printPortReport(report)
	}

	if len(report.conflicts) > 0 {
		if cliout.IsJSON() && cmd != nil {
			cmd.SilenceErrors = true
		}
		return fmt.Errorf("duplicate explicit host port(s): %s", conflictSummary(report.conflicts))
	}
	return nil
}

func newPortJSONReport(report portReport) portJSONReport {
	conflicts := report.conflicts
	if conflicts == nil {
		conflicts = []portConflict{}
	}
	return portJSONReport{Services: report.services, Conflicts: conflicts}
}

// collectPortReport resolves the port bindings for every service and flags any
// explicit host binding that overlaps another binding.
func collectPortReport(azureYaml *service.AzureYaml) portReport {
	names := sortedServiceNames(azureYaml.Services)
	services := make(map[string]servicePorts, len(names))
	explicit := make([]collectedPortBinding, 0)

	for _, name := range names {
		svc := azureYaml.Services[name]
		mappings, _ := svc.GetPortMappings()
		bindings := make([]portBinding, 0, len(mappings))
		for _, m := range mappings {
			protocol := protocolOrDefault(m.Protocol)
			b := portBinding{
				BindIP:    m.BindIP,
				Container: m.ContainerPort,
				Protocol:  protocol,
			}
			if m.HostPort > 0 {
				b.HostPort = m.HostPort
				b.Host = strconv.Itoa(m.HostPort)
				explicit = append(explicit, collectedPortBinding{
					Service: name,
					Index:   len(bindings),
					Key: portBindingKey{
						BindIP:   m.BindIP,
						HostPort: m.HostPort,
						Protocol: protocol,
					},
				})
			} else {
				b.Host = "auto"
			}
			bindings = append(bindings, b)
		}
		services[name] = servicePorts{Ports: bindings}
	}

	conflicts := detectPortConflicts(explicit, services)

	return portReport{services: services, order: names, conflicts: conflicts}
}

func detectPortConflicts(explicit []collectedPortBinding, services map[string]servicePorts) []portConflict {
	if len(explicit) < 2 {
		return nil
	}

	parent := make([]int, len(explicit))
	conflicting := make([]bool, len(explicit))
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(i, j int) {
		ri := find(i)
		rj := find(j)
		if ri != rj {
			parent[rj] = ri
		}
	}

	for i := range explicit {
		for j := i + 1; j < len(explicit); j++ {
			if portBindingsOverlap(explicit[i].Key, explicit[j].Key) {
				conflicting[i] = true
				conflicting[j] = true
				union(i, j)
			}
		}
	}

	groups := make(map[int][]int)
	for i, isConflict := range conflicting {
		if !isConflict {
			continue
		}
		ref := explicit[i]
		sp := services[ref.Service]
		sp.Ports[ref.Index].Conflict = true
		services[ref.Service] = sp
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	conflicts := make([]portConflict, 0, len(groups))
	for _, indexes := range groups {
		first := explicit[indexes[0]].Key
		owners := make([]string, 0, len(indexes))
		bindIPs := make([]string, 0, len(indexes))
		for _, idx := range indexes {
			owners = append(owners, explicit[idx].Service)
			bindIPs = append(bindIPs, explicit[idx].Key.BindIP)
		}
		conflicts = append(conflicts, portConflict{
			BindIP:   summarizeBindIPs(bindIPs),
			HostPort: first.HostPort,
			Protocol: first.Protocol,
			Owners:   dedupeStrings(owners),
		})
	}

	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].HostPort != conflicts[j].HostPort {
			return conflicts[i].HostPort < conflicts[j].HostPort
		}
		if conflicts[i].Protocol != conflicts[j].Protocol {
			return conflicts[i].Protocol < conflicts[j].Protocol
		}
		return conflicts[i].BindIP < conflicts[j].BindIP
	})
	return conflicts
}

func portBindingsOverlap(a, b portBindingKey) bool {
	if a.HostPort != b.HostPort || a.Protocol != b.Protocol {
		return false
	}
	return bindIPsOverlap(a.BindIP, b.BindIP)
}

func bindIPsOverlap(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return true
	}
	// An empty bind IP means all interfaces for both IP families. 0.0.0.0 only
	// overlaps IPv4 or hostname binds, and :: only overlaps IPv6 binds.
	if a == "" || b == "" {
		return true
	}
	if isIPv4Wildcard(a) {
		return bindIPFamily(b) != "ipv6"
	}
	if isIPv4Wildcard(b) {
		return bindIPFamily(a) != "ipv6"
	}
	if isIPv6Wildcard(a) {
		return bindIPFamily(b) == "ipv6"
	}
	if isIPv6Wildcard(b) {
		return bindIPFamily(a) == "ipv6"
	}
	return false
}

func bindIPFamily(bindIP string) string {
	ip := net.ParseIP(bindIP)
	if ip == nil {
		return "unknown"
	}
	if ip.To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}

func isIPv4Wildcard(bindIP string) bool {
	return bindIP == "0.0.0.0"
}

func isIPv6Wildcard(bindIP string) bool {
	return bindIP == "::"
}

func summarizeBindIPs(bindIPs []string) string {
	unique := dedupeStrings(bindIPs)
	if len(unique) == 1 {
		return unique[0]
	}
	labels := make([]string, 0, len(unique))
	for _, bindIP := range unique {
		if bindIP == "" {
			labels = append(labels, "all")
			continue
		}
		labels = append(labels, bindIP)
	}
	return strings.Join(labels, ", ")
}

// printPortReport writes the port bindings for each service, then a warning line
// per overlapping host binding.
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
			host := formatHostBinding(b.BindIP, b.Host)
			line := fmt.Sprintf("%s -> %d/%s", host, b.Container, b.Protocol)
			if b.Conflict {
				line += "  (conflict)"
			}
			cliout.Label("port", line)
		}
	}

	if len(report.conflicts) > 0 {
		cliout.Newline()
		for _, c := range report.conflicts {
			cliout.Warning("Host binding %s is bound by more than one service: %s", conflictBindingLabel(c), strings.Join(c.Owners, ", "))
		}
	}
}

// conflictSummary renders the conflicts for the returned error, e.g.
// "3000/tcp (web, worker); 127.0.0.1:8080/tcp (api, api2)".
func conflictSummary(conflicts []portConflict) string {
	parts := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		parts = append(parts, fmt.Sprintf("%s (%s)", conflictBindingLabel(c), strings.Join(c.Owners, ", ")))
	}
	return strings.Join(parts, "; ")
}

func conflictBindingLabel(c portConflict) string {
	return fmt.Sprintf("%s/%s", formatHostBinding(c.BindIP, strconv.Itoa(c.HostPort)), c.Protocol)
}

func formatHostBinding(bindIP, host string) string {
	if bindIP == "" {
		return host
	}
	if strings.Contains(bindIP, ":") {
		return fmt.Sprintf("[%s]:%s", bindIP, host)
	}
	return fmt.Sprintf("%s:%s", bindIP, host)
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
