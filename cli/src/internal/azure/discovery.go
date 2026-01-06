package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ResourceType represents the type of Azure compute resource.
type ResourceType string

const (
	ResourceTypeContainerApp      ResourceType = "containerApp"
	ResourceTypeAppService        ResourceType = "appService"
	ResourceTypeFunction          ResourceType = "function"
	ResourceTypeAKS               ResourceType = "aks"
	ResourceTypeContainerInstance ResourceType = "containerInstance"
	ResourceTypeUnknown           ResourceType = "unknown"
)

// AzureResource represents a discovered Azure resource.
type AzureResource struct {
	ServiceName             string       `json:"serviceName"`
	ResourceID              string       `json:"resourceId"`
	ResourceType            ResourceType `json:"resourceType"`
	ResourceGroup           string       `json:"resourceGroup"`
	SubscriptionID          string       `json:"subscriptionId"`
	LogAnalyticsWorkspaceID string       `json:"logAnalyticsWorkspaceId,omitempty"`
	URL                     string       `json:"url,omitempty"`
	Name                    string       `json:"name"`
}

// DiscoveryResult holds the results of Azure resource discovery.
type DiscoveryResult struct {
	SubscriptionID          string                    `json:"subscriptionId"`
	ResourceGroup           string                    `json:"resourceGroup"`
	Environment             string                    `json:"environment"`
	LogAnalyticsWorkspaceID string                    `json:"logAnalyticsWorkspaceId,omitempty"`
	Resources               map[string]*AzureResource `json:"resources"` // keyed by service name
	DiscoveredAt            time.Time                 `json:"discoveredAt"`
}

// ResourceDiscovery handles discovering Azure resources for azd projects.
type ResourceDiscovery struct {
	credential    azcore.TokenCredential
	projectDir    string
	cache         *DiscoveryResult
	cacheMu       sync.RWMutex
	cacheDuration time.Duration
}

// NewResourceDiscovery creates a new resource discovery instance.
func NewResourceDiscovery(credential azcore.TokenCredential, projectDir string) *ResourceDiscovery {
	return &ResourceDiscovery{
		credential:    credential,
		projectDir:    projectDir,
		cacheDuration: 5 * time.Minute,
	}
}

// Discover finds Azure resources for the current azd project.
// Results are cached for 5 minutes to reduce API calls.
func (d *ResourceDiscovery) Discover(ctx context.Context) (*DiscoveryResult, error) {
	// Check cache first
	d.cacheMu.RLock()
	if d.cache != nil && time.Since(d.cache.DiscoveredAt) < d.cacheDuration {
		result := d.cache
		d.cacheMu.RUnlock()
		return result, nil
	}
	d.cacheMu.RUnlock()

	// Get azd environment values
	envValues, err := d.getAzdEnvValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get azd environment values: %w", err)
	}

	result := &DiscoveryResult{
		SubscriptionID: envValues["AZURE_SUBSCRIPTION_ID"],
		ResourceGroup:  envValues["AZURE_RESOURCE_GROUP_NAME"],
		Environment:    envValues["AZURE_ENV_NAME"],
		Resources:      make(map[string]*AzureResource),
		DiscoveredAt:   time.Now(),
	}

	// Extract service resources from environment values
	servicePattern := regexp.MustCompile(`^SERVICE_([A-Z0-9_]+)_(URL|NAME|IMAGE_NAME)$`)
	services := make(map[string]map[string]string)

	for key, value := range envValues {
		matches := servicePattern.FindStringSubmatch(key)
		if matches != nil {
			// Normalize service name: lowercase and replace underscores with hyphens
			// to match azure.yaml naming convention (e.g., containerapp-api)
			serviceName := strings.ToLower(strings.ReplaceAll(matches[1], "_", "-"))
			field := matches[2]
			if services[serviceName] == nil {
				services[serviceName] = make(map[string]string)
			}
			services[serviceName][field] = value
		}
	}

	// Build resource entries
	for serviceName, fields := range services {
		// Skip services that run locally (host: local or localhost in azure.yaml)
		// These don't have Azure resources and shouldn't be included in Azure logs
		if fields["URL"] == "" || strings.Contains(fields["URL"], "localhost") || strings.Contains(fields["URL"], "127.0.0.1") {
			slog.Debug("discovery: skipping local service", "serviceName", serviceName, "url", fields["URL"])
			continue
		}

		resource := &AzureResource{
			ServiceName:    serviceName,
			SubscriptionID: result.SubscriptionID,
			ResourceGroup:  result.ResourceGroup,
			URL:            fields["URL"],
			Name:           fields["NAME"],
		}

		slog.Debug("discovery: processing service", "serviceName", serviceName, "azureName", fields["NAME"], "url", fields["URL"])

		// Determine resource type from ARM if we have subscription and resource group
		if result.SubscriptionID != "" && result.ResourceGroup != "" && fields["NAME"] != "" {
			resourceType, resourceID := d.detectResourceType(ctx, result.SubscriptionID, result.ResourceGroup, fields["NAME"])
			resource.ResourceType = resourceType
			resource.ResourceID = resourceID
			slog.Debug("discovery: detected resource type", "serviceName", serviceName, "type", resourceType, "resourceID", resourceID)
		} else {
			// Infer from URL pattern
			resource.ResourceType = inferResourceTypeFromURL(fields["URL"])
			slog.Debug("discovery: inferred resource type from URL", "serviceName", serviceName, "type", resource.ResourceType)
		}

		result.Resources[serviceName] = resource
	}

	// Get Log Analytics workspace ID (GUID) for querying
	// Priority: GUID env var > resource ID env var (extract name) > auto-detection
	if wsGUID := envValues["AZURE_LOG_ANALYTICS_WORKSPACE_GUID"]; wsGUID != "" {
		// Prefer the GUID directly if available
		result.LogAnalyticsWorkspaceID = wsGUID
	} else if wsID := envValues["AZURE_LOG_ANALYTICS_WORKSPACE_ID"]; wsID != "" {
		// Fall back to resource ID (will need to extract workspace name or query for GUID)
		result.LogAnalyticsWorkspaceID = wsID
	} else if result.SubscriptionID != "" && result.ResourceGroup != "" {
		// Last resort: auto-detection from resource group
		workspaceID := d.detectLogAnalyticsWorkspace(ctx, result.SubscriptionID, result.ResourceGroup)
		result.LogAnalyticsWorkspaceID = workspaceID
	}

	// Update cache
	d.cacheMu.Lock()
	d.cache = result
	d.cacheMu.Unlock()

	return result, nil
}

// getAzdEnvValues runs 'azd env get-values' and parses the output.
func (d *ResourceDiscovery) getAzdEnvValues(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "azd", "env", "get-values", "--output", "json")
	if d.projectDir != "" {
		cmd.Dir = d.projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		// If azd env get-values fails, return empty map (not provisioned yet)
		return make(map[string]string), nil
	}

	var values map[string]string
	if err := json.Unmarshal(output, &values); err != nil {
		// Try parsing as key=value pairs (older azd versions)
		values = make(map[string]string)
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if idx := strings.Index(line, "="); idx > 0 {
				key := line[:idx]
				value := strings.Trim(line[idx+1:], `"'`)
				values[key] = value
			}
		}
	}

	return values, nil
}

// detectResourceType queries ARM to determine the resource type.
func (d *ResourceDiscovery) detectResourceType(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (ResourceType, string) {
	if d.credential == nil {
		slog.Debug("detectResourceType: no credential available")
		return ResourceTypeUnknown, ""
	}

	client, err := armresources.NewClient(subscriptionID, d.credential, nil)
	if err != nil {
		slog.Debug("detectResourceType: failed to create ARM client", "error", err)
		return ResourceTypeUnknown, ""
	}

	// List resources in the resource group and find matching name
	pager := client.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Debug("detectResourceType: pager error", "error", err)
			break
		}

		for _, resource := range page.Value {
			if resource.Name != nil && strings.EqualFold(*resource.Name, resourceName) {
				var kindStr string
				if resource.Kind != nil {
					kindStr = *resource.Kind
				}
				slog.Debug("detectResourceType: found resource", "name", *resource.Name, "type", *resource.Type, "kind", kindStr)
				resourceType := mapARMTypeToResourceType(*resource.Type, resource.Kind)
				return resourceType, *resource.ID
			}
		}
	}

	slog.Debug("detectResourceType: resource not found in ARM", "resourceName", resourceName)
	return ResourceTypeUnknown, ""
}

// detectLogAnalyticsWorkspace tries to find a Log Analytics workspace in the resource group.
func (d *ResourceDiscovery) detectLogAnalyticsWorkspace(ctx context.Context, subscriptionID, resourceGroup string) string {
	if d.credential == nil {
		return ""
	}

	client, err := armresources.NewClient(subscriptionID, d.credential, nil)
	if err != nil {
		return ""
	}

	pager := client.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			break
		}

		for _, resource := range page.Value {
			if resource.Type != nil && strings.EqualFold(*resource.Type, "Microsoft.OperationalInsights/workspaces") {
				return *resource.ID
			}
		}
	}

	return ""
}

// ClearCache clears the discovery cache, forcing a refresh on next Discover call.
func (d *ResourceDiscovery) ClearCache() {
	d.cacheMu.Lock()
	d.cache = nil
	d.cacheMu.Unlock()
}

// GetResource returns a specific resource by service name, or nil if not found.
func (d *ResourceDiscovery) GetResource(ctx context.Context, serviceName string) (*AzureResource, error) {
	result, err := d.Discover(ctx)
	if err != nil {
		return nil, err
	}

	resource, ok := result.Resources[strings.ToLower(serviceName)]
	if !ok {
		return nil, fmt.Errorf("service '%s' not found in Azure", serviceName)
	}

	return resource, nil
}

// mapARMTypeToResourceType maps Azure ARM resource types to our ResourceType enum.
// The kind parameter is used to differentiate between similar resource types (e.g., App Service vs Function App).
func mapARMTypeToResourceType(armType string, kind *string) ResourceType {
	armType = strings.ToLower(armType)
	switch {
	case strings.Contains(armType, "microsoft.app/containerapps"):
		return ResourceTypeContainerApp
	case strings.Contains(armType, "microsoft.web/sites"):
		// Differentiate between App Service and Function App using the "kind" property
		// Function Apps have kind values like "functionapp", "functionapp,linux", "functionapp,workflowapp"
		if kind != nil && isFunctionAppKind(*kind) {
			return ResourceTypeFunction
		}
		return ResourceTypeAppService
	case strings.Contains(armType, "microsoft.containerservice/managedclusters"):
		return ResourceTypeAKS
	case strings.Contains(armType, "microsoft.containerinstance/containergroups"):
		return ResourceTypeContainerInstance
	default:
		return ResourceTypeUnknown
	}
}

// isFunctionAppKind checks if the kind string indicates a Function App.
// Function Apps have kind values like "functionapp", "functionapp,linux",
// "functionapp,workflowapp", "functionapp,linux,container", etc.
func isFunctionAppKind(kind string) bool {
	kindLower := strings.ToLower(kind)
	// Check if "functionapp" is present anywhere in the kind value
	return strings.Contains(kindLower, "functionapp")
}

// inferResourceTypeFromURL tries to determine resource type from the URL pattern.
func inferResourceTypeFromURL(url string) ResourceType {
	if url == "" {
		return ResourceTypeUnknown
	}

	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, ".azurecontainerapps.io"):
		return ResourceTypeContainerApp
	case strings.Contains(url, ".azurewebsites.net"):
		return ResourceTypeAppService
	case strings.Contains(url, ".azurefunctions.net"):
		return ResourceTypeFunction
	default:
		return ResourceTypeUnknown
	}
}

// GetAzureEnvInfo returns environment variables needed for Azure operations.
// This is a convenience method for getting subscription and resource group info.
func GetAzureEnvInfo(projectDir string) (subscriptionID, resourceGroup, envName string, err error) {
	discovery := &ResourceDiscovery{projectDir: projectDir}
	values, err := discovery.getAzdEnvValues(context.Background())
	if err != nil {
		return "", "", "", err
	}

	return values["AZURE_SUBSCRIPTION_ID"], values["AZURE_RESOURCE_GROUP_NAME"], values["AZURE_ENV_NAME"], nil
}
