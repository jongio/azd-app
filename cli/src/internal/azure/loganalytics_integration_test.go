//go:build integration

package azure

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"
)

// TestLogAnalyticsQuery_Integration tests querying Log Analytics with real credentials.
// Run with: go test -tags=integration -v -run TestLogAnalyticsQuery_Integration
func TestLogAnalyticsQuery_Integration(t *testing.T) {
	workspaceGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	if workspaceGUID == "" {
		t.Skip("AZURE_LOG_ANALYTICS_WORKSPACE_GUID not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get credential using DefaultAzureCredential (which uses azd auth)
	cred, err := NewLogAnalyticsCredential()
	if err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	// Test that we can get a token for Log Analytics scope
	t.Run("TestCredential", func(t *testing.T) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://api.loganalytics.io/.default"},
		})
		if err != nil {
			t.Fatalf("failed to get token: %v", err)
		}
		t.Logf("Got token, expires: %v", token.ExpiresOn)
		t.Logf("Token length: %d", len(token.Token))
	})

	// Create raw azlogs client to test directly
	client, err := azlogs.NewClient(cred, nil)
	if err != nil {
		t.Fatalf("failed to create azlogs client: %v", err)
	}

	// Direct query test with longer timespan
	t.Run("QueryWithLongerTimespan", func(t *testing.T) {
		query := "ContainerAppConsoleLogs_CL | take 5"
		timespan := azlogs.TimeInterval("P1D") // 1 day instead of 1 hour

		t.Logf("Querying workspace: %s", workspaceGUID)
		t.Logf("Query: %s", query)
		t.Logf("Timespan: %s", timespan)

		resp, err := client.QueryWorkspace(ctx, workspaceGUID, azlogs.QueryBody{
			Query:    &query,
			Timespan: &timespan,
		}, nil)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		t.Logf("Response tables: %d", len(resp.Tables))
		for ti, table := range resp.Tables {
			t.Logf("Table %d: %d columns, %d rows", ti, len(table.Columns), len(table.Rows))

			// Print column names
			var colNames []string
			for _, col := range table.Columns {
				if col.Name != nil {
					colNames = append(colNames, *col.Name)
				}
			}
			t.Logf("  Columns: %v", colNames)

			// Print first few rows
			for ri, row := range table.Rows {
				if ri >= 3 {
					break
				}
				t.Logf("  Row %d: %v", ri, truncateRow(row))
			}
		}
	})
}

func truncateRow(row []any) []string {
	var result []string
	for _, v := range row {
		s := fmt.Sprintf("%v", v)
		if len(s) > 50 {
			s = s[:50] + "..."
		}
		result = append(result, s)
	}
	return result
}
