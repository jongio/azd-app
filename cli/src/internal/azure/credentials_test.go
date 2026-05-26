package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"
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

func TestExtractJWTExpiry(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		expectFuture bool   // expect expiry to be in the future
		expectApprox *int64 // if set, expect expiry unix timestamp near this value
	}{
		{
			name:         "valid JWT with future exp",
			token:        buildTestJWT(t, 1999999999), // ~2033
			expectFuture: true,
			expectApprox: ptrInt64(1999999999),
		},
		{
			name:         "valid JWT with past exp uses fallback",
			token:        buildTestJWT(t, 1000000000), // 2001 - in the past
			expectFuture: true,
			expectApprox: nil, // fallback ~1 hour from now
		},
		{
			name:         "not a JWT (no dots)",
			token:        "this-is-not-a-jwt",
			expectFuture: true,
			expectApprox: nil, // fallback
		},
		{
			name:         "JWT with no exp claim",
			token:        buildTestJWTNoClaim(t),
			expectFuture: true,
			expectApprox: nil, // fallback
		},
		{
			name:         "JWT with invalid base64 payload",
			token:        "header.!!!invalid-base64!!!.signature",
			expectFuture: true,
			expectApprox: nil, // fallback
		},
		{
			name:         "JWT with invalid JSON payload",
			token:        "header." + base64RawURL([]byte("{not json")) + ".signature",
			expectFuture: true,
			expectApprox: nil, // fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJWTExpiry(tt.token)

			if tt.expectFuture {
				if result.Before(time.Now()) {
					t.Errorf("expected expiry in the future, got %v", result)
				}
			}

			if tt.expectApprox != nil {
				expected := time.Unix(*tt.expectApprox, 0)
				diff := result.Sub(expected)
				if diff < -time.Second || diff > time.Second {
					t.Errorf("expected expiry near %v, got %v (diff: %v)", expected, result, diff)
				}
			}
		})
	}
}

func TestAzdTokenCredential_JWTExpiry(t *testing.T) {
	// Build a valid JWT with a known future expiry
	futureExp := time.Now().Add(2 * time.Hour).Unix()
	token := buildTestJWT(t, futureExp)

	cred, err := NewAzdTokenCredential(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The credential should use the JWT's exp, not the 1-hour fallback
	expectedExpiry := time.Unix(futureExp, 0)
	diff := cred.expiresOn.Sub(expectedExpiry)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected expiry from JWT (%v), got %v (diff: %v)", expectedExpiry, cred.expiresOn, diff)
	}
}

// buildTestJWT creates a minimal JWT token with the given exp claim for testing.
func buildTestJWT(t *testing.T, exp int64) string {
	t.Helper()
	header := base64RawURL([]byte(`{"typ":"JWT","alg":"none"}`))
	payload := base64RawURL([]byte(fmt.Sprintf(`{"sub":"test","exp":%d}`, exp)))
	return header + "." + payload + ".signature"
}

// buildTestJWTNoClaim creates a JWT with no exp claim.
func buildTestJWTNoClaim(t *testing.T) string {
	t.Helper()
	header := base64RawURL([]byte(`{"typ":"JWT","alg":"none"}`))
	payload := base64RawURL([]byte(`{"sub":"test"}`))
	return header + "." + payload + ".signature"
}

func base64RawURL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func ptrInt64(v int64) *int64 {
	return &v
}
