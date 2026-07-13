package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewRemoveCommand creates the remove command.
func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [service]",
		Short: "Remove a service from azure.yaml",
		Long: `Remove a service from the services section of azure.yaml.

This is the inverse of "azd app add". It deletes the named service entry and
leaves every other service in the file untouched. Use it to undo an add or to
drop a service you no longer run.

Examples:
  # Remove the redis service
  azd app remove redis

  # JSON output
  azd app remove redis --output json`,
		SilenceUsage:      true,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runRemove,
		ValidArgsFunction: completeServiceArgs,
	}

	return cmd
}

// RemoveResult represents the result of removing a service, mirroring AddResult.
type RemoveResult struct {
	Service string `json:"service"`
	Removed bool   `json:"removed"`
	Message string `json:"message,omitempty"`
}

func runRemove(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("specify a service name to remove")
	}

	serviceName := args[0]

	azureYamlPath, err := findAzureYamlForAdd()
	if err != nil {
		return fmt.Errorf("find azure.yaml: %w", err)
	}

	cliout.CommandHeader("remove", fmt.Sprintf("Remove %s", serviceName))

	removed, remaining, err := removeServiceFromYaml(azureYamlPath, serviceName)
	if err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}

	if !removed {
		return fmt.Errorf("%s", serviceNotFoundMessage(serviceName, remaining))
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(RemoveResult{
			Service: serviceName,
			Removed: true,
			Message: fmt.Sprintf("Removed %s from azure.yaml", serviceName),
		})
	}

	cliout.Success("Removed %s from azure.yaml", serviceName)
	return nil
}

// removeServiceFromYaml deletes serviceName from the services mapping in the
// azure.yaml at path. It returns whether the service was found and removed, plus
// the sorted names of the services that remain (used to build a helpful error
// when the service was not found). Every other part of the file is preserved by
// editing the parsed node tree in place.
func removeServiceFromYaml(path, serviceName string) (bool, []string, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return false, nil, err
	}

	var doc yaml.Node
	if err = yaml.Unmarshal(data, &doc); err != nil {
		return false, nil, err
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return false, nil, fmt.Errorf("invalid azure.yaml structure")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return false, nil, fmt.Errorf("azure.yaml root must be a mapping")
	}

	// Find the services mapping.
	var servicesNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "services" {
			servicesNode = root.Content[i+1]
			break
		}
	}
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return false, nil, nil
	}

	// Locate the service key/value pair.
	removeIdx := -1
	for j := 0; j < len(servicesNode.Content)-1; j += 2 {
		if servicesNode.Content[j].Value == serviceName {
			removeIdx = j
			break
		}
	}

	if removeIdx < 0 {
		return false, serviceNamesFromMapping(servicesNode), nil
	}

	// Drop the key node and its value node (two consecutive entries).
	servicesNode.Content = append(servicesNode.Content[:removeIdx], servicesNode.Content[removeIdx+2:]...)

	yamlOutput, err := yaml.Marshal(&doc)
	if err != nil {
		return false, nil, err
	}

	// #nosec G306 -- azure.yaml needs to be readable
	if err := os.WriteFile(path, yamlOutput, 0o644); err != nil {
		return false, nil, err
	}

	return true, serviceNamesFromMapping(servicesNode), nil
}

// serviceNamesFromMapping returns the sorted service names in a services mapping
// node.
func serviceNamesFromMapping(servicesNode *yaml.Node) []string {
	names := make([]string, 0, len(servicesNode.Content)/2)
	for j := 0; j < len(servicesNode.Content)-1; j += 2 {
		names = append(names, servicesNode.Content[j].Value)
	}
	sort.Strings(names)
	return names
}

// serviceNotFoundMessage builds the error text shown when a service to remove is
// not present, listing the current service names when there are any.
func serviceNotFoundMessage(serviceName string, remaining []string) string {
	if len(remaining) == 0 {
		return fmt.Sprintf("service %q not found in azure.yaml", serviceName)
	}
	return fmt.Sprintf("service %q not found in azure.yaml. Current services: %s",
		serviceName, strings.Join(remaining, ", "))
}
