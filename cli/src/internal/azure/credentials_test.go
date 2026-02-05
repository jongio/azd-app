package azure

import (
	"context"
	"os"
	"testing"
)

func TestNewAzureCredential(t *testing.T) {
	// Test with no credentials (should return error or fallback)
	// Clear any existing environment variables that might affect the test
	originalToken := os.Getenv("AZD_ACCESS_TOKEN")
	defer func() { _ = os.Setenv("AZD_ACCESS_TOKEN", originalToken) }()

	_ = os.Unsetenv("AZD_ACCESS_TOKEN")

	// This test verifies the function doesn't panic
	cred, err := NewAzureCredential()
	if err != nil {
		// Expected when no credentials are available
		t.Logf("NewAzureCredential returned error (expected without credentials): %v", err)
	} else {
		t.Log("NewAzureCredential returned credential successfully")
		if cred == nil {
			t.Error("Credential should not be nil when no error")
		}
	}
}

func TestAzdTokenCredential(t *testing.T) {
	// Test creating token credential with valid token
	testToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ.eyJzdWIiOiJ0ZXN0IiwiZXhwIjoxOTk5OTk5OTk5fQ.sig"

	cred, err := NewAzdTokenCredential(testToken)
	if err != nil {
		t.Logf("NewAzdTokenCredential returned error (may be expected for invalid token format): %v", err)
	} else if cred == nil {
		t.Error("Credential should not be nil when no error")
	}
}

func TestCredentialChain(t *testing.T) {
	// Clear env var for test
	originalToken := os.Getenv("AZD_ACCESS_TOKEN")
	defer func() { _ = os.Setenv("AZD_ACCESS_TOKEN", originalToken) }()
	_ = os.Unsetenv("AZD_ACCESS_TOKEN")

	chain, err := NewCredentialChain()
	if err != nil {
		t.Logf("NewCredentialChain returned error (expected without credentials): %v", err)
		return
	}

	if chain == nil {
		t.Error("CredentialChain should not be nil when no error")
		return
	}

	source := chain.Source()
	if source == "" {
		t.Error("CredentialChain.Source() should return non-empty string")
	}
	t.Logf("Credential source: %s", source)
}

func TestValidateCredentials(t *testing.T) {
	// Clear env var for test
	originalToken := os.Getenv("AZD_ACCESS_TOKEN")
	defer func() { _ = os.Setenv("AZD_ACCESS_TOKEN", originalToken) }()
	_ = os.Unsetenv("AZD_ACCESS_TOKEN")

	cred, err := NewAzureCredential()
	if err != nil {
		t.Logf("No credential available for validation test: %v", err)
		return
	}

	// Validate credentials (will likely fail without real Azure auth)
	ctx := context.Background()
	err = ValidateCredentials(ctx, cred)
	if err != nil {
		t.Logf("ValidateCredentials returned error (expected without real auth): %v", err)
	} else {
		t.Log("ValidateCredentials succeeded")
	}
}

func TestGetCredentialSource(t *testing.T) {
	cred, err := NewAzureCredential()
	if err != nil {
		t.Logf("No credential available: %v", err)
		return
	}

	source := GetCredentialSource(cred)
	if source == "" {
		t.Error("GetCredentialSource should return non-empty string")
	}
	t.Logf("Credential source: %s", source)
}

func TestCredentialErrors(t *testing.T) {
	// Test error types exist
	if ErrNoCredentials == nil {
		t.Error("ErrNoCredentials should not be nil")
	}
	if ErrTokenExpired == nil {
		t.Error("ErrTokenExpired should not be nil")
	}
	if ErrAuthNotConfigured == nil {
		t.Error("ErrAuthNotConfigured should not be nil")
	}
}
