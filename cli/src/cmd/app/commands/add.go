package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/wellknown"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

// NewAddCommand creates the add command.
func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [service]",
		Short: "Add a well-known service to azure.yaml",
		Long: `Add a well-known container service to your azure.yaml configuration.

This command simplifies adding commonly-used services like Azure emulators,
databases, and caches. It automatically configures the Docker image, ports,
environment variables, and health checks.

Available services:
  azurite   - Azure Storage emulator (Blob, Queue, Table)
  cosmos    - Azure Cosmos DB emulator
  redis     - Redis in-memory cache
  postgres  - PostgreSQL database

Examples:
  # List available services
  azd app add --list

  # Add Azurite storage emulator
  azd app add azurite

  # Add PostgreSQL database
  azd app add postgres

  # Show connection string after adding
  azd app add redis --show-connection

  # JSON output
  azd app add azurite --output json`,
		SilenceUsage: true,
		RunE:         runAdd,
	}

	cmd.Flags().Bool("list", false, "List all available services")
	cmd.Flags().Bool("show-connection", false, "Show connection string after adding")

	return cmd
}

// AddResult represents the result of adding a service.
type AddResult struct {
	Service           string            `json:"service"`
	Added             bool              `json:"added"`
	Message           string            `json:"message,omitempty"`
	ConnectionStrings map[string]string `json:"connectionStrings,omitempty"`
}

func runAdd(cmd *cobra.Command, args []string) error {
	listServices, _ := cmd.Flags().GetBool("list")
	showConnection, _ := cmd.Flags().GetBool("show-connection")

	// Handle --list flag
	if listServices {
		return listAvailableServices()
	}

	// Require a service name
	if len(args) == 0 {
		return fmt.Errorf("specify a service name or use --list to see available services")
	}

	serviceName := strings.ToLower(args[0])

	// Look up the service definition
	def := wellknown.Get(serviceName)
	if def == nil {
		return fmt.Errorf("unknown service %q - use --list to see available services", serviceName)
	}

	output.CommandHeader("add", fmt.Sprintf("Add %s", def.DisplayName))

	// Find azure.yaml
	azureYamlPath, err := findAzureYamlForAdd()
	if err != nil {
		return err
	}

	// Check if service already exists
	exists, err := serviceExistsInYaml(azureYamlPath, serviceName)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}
	if exists {
		if output.IsJSON() {
			return output.PrintJSON(AddResult{
				Service: serviceName,
				Added:   false,
				Message: fmt.Sprintf("Service %q already exists in azure.yaml", serviceName),
			})
		}
		output.Warning("Service %q already exists in azure.yaml", serviceName)
		return nil
	}

	// Add the service to azure.yaml
	if err := addServiceToYaml(azureYamlPath, serviceName, def); err != nil {
		return fmt.Errorf("failed to add service: %w", err)
	}

	// Success output
	if output.IsJSON() {
		result := AddResult{
			Service: serviceName,
			Added:   true,
			Message: fmt.Sprintf("Added %s to azure.yaml", def.DisplayName),
		}
		if showConnection {
			result.ConnectionStrings = def.ConnectionStrings
		}
		return output.PrintJSON(result)
	}

	output.Success("Added %s to azure.yaml", def.DisplayName)
	output.Newline()

	// Show connection strings if requested
	if showConnection {
		output.Section(output.IconInfo, "Connection Strings")
		for name, connStr := range def.ConnectionStrings {
			output.Label(name, connStr)
		}
		output.Newline()
	}

	output.Info("%s Run 'azd app run' to start all services including %s", output.IconBulb, serviceName)

	return nil
}

func listAvailableServices() error {
	output.CommandHeader("add", "Available services")

	names := wellknown.Names()
	sort.Strings(names)

	if output.IsJSON() {
		type ServiceInfo struct {
			Name        string            `json:"name"`
			DisplayName string            `json:"displayName"`
			Description string            `json:"description"`
			Category    string            `json:"category"`
			Image       string            `json:"image"`
			Ports       []string          `json:"ports"`
			ConnStrings map[string]string `json:"connectionStrings"`
		}
		services := make([]ServiceInfo, 0, len(names))
		for _, name := range names {
			def := wellknown.Get(name)
			services = append(services, ServiceInfo{
				Name:        def.Name,
				DisplayName: def.DisplayName,
				Description: def.Description,
				Category:    def.Category,
				Image:       def.Image,
				Ports:       def.Ports,
				ConnStrings: def.ConnectionStrings,
			})
		}
		return output.PrintJSON(services)
	}

	// Group by category
	categories := wellknown.Categories()
	sort.Strings(categories)

	titleCaser := cases.Title(language.English)
	for _, cat := range categories {
		output.Section(output.IconFolder, titleCaser.String(cat))
		services := wellknown.ByCategory(cat)
		for _, def := range services {
			output.Bullet("%s - %s", def.Name, def.Description)
		}
		output.Newline()
	}

	output.Info("%s Use 'azd app add <service>' to add a service to azure.yaml", output.IconBulb)

	return nil
}

func findAzureYamlForAdd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Search current and parent directories
	dir := cwd
	for {
		path := filepath.Join(dir, "azure.yaml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no azure.yaml found - run from a project directory")
}

func serviceExistsInYaml(path string, serviceName string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, err
	}

	// Navigate to services section
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return false, nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return false, nil
	}

	// Find services key
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "services" {
			servicesNode := root.Content[i+1]
			if servicesNode.Kind != yaml.MappingNode {
				return false, nil
			}
			// Check each service name
			for j := 0; j < len(servicesNode.Content)-1; j += 2 {
				if servicesNode.Content[j].Value == serviceName {
					return true, nil
				}
			}
			return false, nil
		}
	}

	return false, nil
}

func addServiceToYaml(path string, serviceName string, def *wellknown.ServiceDefinition) error {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err = yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Ensure document structure
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("invalid azure.yaml structure")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("azure.yaml root must be a mapping")
	}

	// Find or create services section
	var servicesNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "services" {
			servicesNode = root.Content[i+1]
			break
		}
	}

	if servicesNode == nil {
		// Create services section
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "services",
			Tag:   "!!str",
		}
		servicesNode = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{},
		}
		root.Content = append(root.Content, keyNode, servicesNode)
	}

	// Build service config node
	serviceConfig := buildServiceNode(def)

	// Add service name and config to services
	nameNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: serviceName,
		Tag:   "!!str",
	}
	servicesNode.Content = append(servicesNode.Content, nameNode, serviceConfig)

	// Write back
	yamlOutput, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	// #nosec G306 -- azure.yaml needs to be readable
	return os.WriteFile(path, yamlOutput, 0644)
}

func buildServiceNode(def *wellknown.ServiceDefinition) *yaml.Node {
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	// Add image
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "image", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: def.Image, Tag: "!!str"},
	)

	// Add ports
	if len(def.Ports) > 0 {
		portsNode := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
		for _, port := range def.Ports {
			portsNode.Content = append(portsNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: port, Tag: "!!str"},
			)
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "ports", Tag: "!!str"},
			portsNode,
		)
	}

	// Add environment variables
	if len(def.Environment) > 0 {
		envNode := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		// Sort keys for consistent output
		keys := make([]string, 0, len(def.Environment))
		for k := range def.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			envNode.Content = append(envNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k, Tag: "!!str"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: def.Environment[k], Tag: "!!str"},
			)
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "environment", Tag: "!!str"},
			envNode,
		)
	}

	return node
}
