// Package azure provides Azure cloud integration for log streaming and resource discovery.
package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

func isProbablyGUID(value string) bool {
	// Cheap check: customerId is a GUID. Avoid regex dependency.
	// Example: 12345678-1234-1234-1234-1234567890ab
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	for i, ch := range value {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
				continue
			}
			return false
		}
	}
	return true
}

func isWorkspaceResourceID(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "/subscriptions/") {
		return false
	}
	return strings.Contains(strings.ToLower(value), "/providers/microsoft.operationalinsights/workspaces/")
}

// NormalizeWorkspaceID returns the Log Analytics workspace customerId GUID.
//
// Input can be:
// - a GUID customerId
// - a workspace ARM resource ID (Microsoft.OperationalInsights/workspaces)
func NormalizeWorkspaceID(ctx context.Context, credential azcore.TokenCredential, workspaceID string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return "", fmt.Errorf("workspace ID is empty")
	}
	if isProbablyGUID(workspaceID) {
		return workspaceID, nil
	}
	if isWorkspaceResourceID(workspaceID) {
		return ResolveWorkspaceCustomerID(ctx, credential, workspaceID)
	}
	return "", fmt.Errorf("workspace ID must be a GUID customerId or a workspace resource ID")
}

// ResolveWorkspaceCustomerID looks up the workspace customerId (GUID) using ARM.
func ResolveWorkspaceCustomerID(ctx context.Context, credential azcore.TokenCredential, workspaceResourceID string) (string, error) {
	if credential == nil {
		return "", fmt.Errorf("credential is required")
	}
	workspaceResourceID = strings.TrimSpace(workspaceResourceID)
	if !isWorkspaceResourceID(workspaceResourceID) {
		return "", fmt.Errorf("not a workspace resource ID")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tok, err := credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
	if err != nil {
		return "", fmt.Errorf("failed to get ARM token: %w", err)
	}

	// Use a stable ARM API version for Operational Insights workspaces.
	// If Azure changes versions, this can be updated; the payload shape is stable for customerId.
	url := "https://management.azure.com" + workspaceResourceID + "?api-version=2022-10-01"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call ARM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("ARM returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Properties struct {
			CustomerID string `json:"customerId"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode ARM response: %w", err)
	}

	customerID := strings.TrimSpace(payload.Properties.CustomerID)
	if !isProbablyGUID(customerID) {
		return "", fmt.Errorf("workspace customerId missing or invalid")
	}

	return customerID, nil
}
