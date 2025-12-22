package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

const decodeResponseErrFmt = "failed to decode response: %v"

type azureLogsResponseBody struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error,omitempty"`
}

type fakeTokenCredential struct{}

func (fakeTokenCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(1 * time.Hour)}, nil
}

func TestHandleAzureLogsDefaultsAndBounds(t *testing.T) {
	oldFetch := fetchAzureLogsStandalone
	t.Cleanup(func() { fetchAzureLogsStandalone = oldFetch })

	var captured azure.StandaloneLogsConfig
	fetchAzureLogsStandalone = func(_ context.Context, cfg azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		captured = cfg
		return []azure.LogEntry{}, nil
	}

	srv := GetServer(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/azure/logs?tail=200000", nil)
	w := httptest.NewRecorder()
	srv.handleAzureLogs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if captured.ProjectDir != srv.projectDir {
		t.Fatalf("expected ProjectDir %q, got %q", srv.projectDir, captured.ProjectDir)
	}
	if captured.Limit != 10000 {
		t.Fatalf("expected tail clamped to 10000, got %d", captured.Limit)
	}
	if captured.Since != time.Hour {
		t.Fatalf("expected default since 1h, got %v", captured.Since)
	}
	if len(captured.Services) != 0 {
		t.Fatalf("expected no services filter when service is omitted, got %v", captured.Services)
	}

	var body azureLogsResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf(decodeResponseErrFmt, err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}
}

func TestHandleAzureLogsServiceFilterPassedThrough(t *testing.T) {
	oldFetch := fetchAzureLogsStandalone
	t.Cleanup(func() { fetchAzureLogsStandalone = oldFetch })

	var captured azure.StandaloneLogsConfig
	fetchAzureLogsStandalone = func(_ context.Context, cfg azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		captured = cfg
		return []azure.LogEntry{}, nil
	}

	srv := GetServer(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/azure/logs?service=api&since=30m&tail=123", nil)
	w := httptest.NewRecorder()
	srv.handleAzureLogs(w, req)

	if captured.Since != 30*time.Minute {
		t.Fatalf("expected since=30m, got %v", captured.Since)
	}
	if captured.Limit != 123 {
		t.Fatalf("expected tail=123, got %d", captured.Limit)
	}
	if len(captured.Services) != 1 || captured.Services[0] != "api" {
		t.Fatalf("expected Services=[api], got %v", captured.Services)
	}
}

func TestHandleAzureLogsErrorMappingSetsHttpStatus(t *testing.T) {
	oldFetch := fetchAzureLogsStandalone
	t.Cleanup(func() { fetchAzureLogsStandalone = oldFetch })

	fetchAzureLogsStandalone = func(_ context.Context, _ azure.StandaloneLogsConfig) ([]azure.LogEntry, error) {
		return nil, &azure.AzureLogsError{Code: "AUTH_REQUIRED", Message: "auth required", Action: "login", Command: "azd auth login"}
	}

	srv := GetServer(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/azure/logs", nil)
	w := httptest.NewRecorder()
	srv.handleAzureLogs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}

	var body azureLogsResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf(decodeResponseErrFmt, err)
	}
	if body.Status != "error" {
		t.Fatalf("expected status error, got %q", body.Status)
	}
	if body.Error == nil || body.Error.Code != "AUTH_REQUIRED" {
		t.Fatalf("expected error code AUTH_REQUIRED, got %+v", body.Error)
	}
}

func runAzureLogsHealthCase(t *testing.T, srv *Server, workspaceID, wantStatus string) {
	getWorkspaceIDFromEnv = func(string) string { return workspaceID }

	req := httptest.NewRequest(http.MethodGet, "/api/azure/logs/health", nil)
	w := httptest.NewRecorder()
	srv.handleAzureLogsHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body struct {
		Status  string `json:"status"`
		DocsURL string `json:"docsUrl"`
		Checks  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf(decodeResponseErrFmt, err)
	}
	if body.Status != wantStatus {
		t.Fatalf("expected overall status %q, got %q", wantStatus, body.Status)
	}
	if body.DocsURL == "" {
		t.Fatalf("expected docsUrl to be set")
	}
	if len(body.Checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(body.Checks))
	}
}

func TestHandleAzureLogsHealthStatus(t *testing.T) {
	oldCred := newLogAnalyticsCredential
	oldValidate := validateCredentials
	oldWorkspace := getWorkspaceIDFromEnv
	oldClient := getOrCreateLogAnalyticsClient
	t.Cleanup(func() {
		newLogAnalyticsCredential = oldCred
		validateCredentials = oldValidate
		getWorkspaceIDFromEnv = oldWorkspace
		getOrCreateLogAnalyticsClient = oldClient
		_ = os.Unsetenv("SERVICE_API_NAME")
	})

	newLogAnalyticsCredential = func() (azcore.TokenCredential, error) { return fakeTokenCredential{}, nil }
	validateCredentials = func(context.Context, azcore.TokenCredential) error { return nil }
	getOrCreateLogAnalyticsClient = func(context.Context, azcore.TokenCredential, string) (*azure.LogAnalyticsClient, error) {
		return nil, nil
	}
	_ = os.Setenv("SERVICE_API_NAME", "api-prod")

	srv := GetServer(t.TempDir())

	cases := []struct {
		name        string
		workspaceID string
		wantStatus  string
	}{
		{name: "healthy", workspaceID: "00000000-0000-0000-0000-000000000000", wantStatus: "healthy"},
		{name: "missing workspace", workspaceID: "", wantStatus: "error"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runAzureLogsHealthCase(t, srv, tc.workspaceID, tc.wantStatus)
		})
	}
}
