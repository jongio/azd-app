package authserver

import (
	"testing"
	"time"
)

func TestJWTManager_CreateToken(t *testing.T) {
	secret := "test-secret-key"
	manager := NewJWTManager(secret)

	tests := []struct {
		name        string
		azureToken  string
		scope       string
		expiresIn   time.Duration
		expectError bool
	}{
		{
			name:        "valid token",
			azureToken:  "valid-azure-token",
			scope:       "https://management.azure.com/.default",
			expiresIn:   15 * time.Minute,
			expectError: false,
		},
		{
			name:        "empty azure token",
			azureToken:  "",
			scope:       "https://management.azure.com/.default",
			expiresIn:   15 * time.Minute,
			expectError: true,
		},
		{
			name:        "custom scope",
			azureToken:  "valid-azure-token",
			scope:       "https://storage.azure.com/.default",
			expiresIn:   30 * time.Minute,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.CreateToken(tt.azureToken, tt.scope, tt.expiresIn)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token == "" {
				t.Error("expected non-empty token")
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	secret := "test-secret-key"
	manager := NewJWTManager(secret)

	// Create a valid token
	azureToken := "test-azure-token"
	scope := "https://management.azure.com/.default"
	expiresIn := 15 * time.Minute

	token, err := manager.CreateToken(azureToken, scope, expiresIn)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Validate the token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Errorf("failed to validate token: %v", err)
	}

	if claims.AccessToken != azureToken {
		t.Errorf("expected access token %s, got %s", azureToken, claims.AccessToken)
	}

	if claims.Scope != scope {
		t.Errorf("expected scope %s, got %s", scope, claims.Scope)
	}
}

func TestJWTManager_ValidateToken_InvalidSecret(t *testing.T) {
	secret1 := "secret-1"
	secret2 := "secret-2"

	manager1 := NewJWTManager(secret1)
	manager2 := NewJWTManager(secret2)

	// Create token with manager1
	token, err := manager1.CreateToken("azure-token", "scope", 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Try to validate with manager2 (different secret)
	_, err = manager2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with different secret")
	}
}

func TestJWTManager_ValidateToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	manager := NewJWTManager(secret)

	// Create a token that expires immediately
	token, err := manager.CreateToken("azure-token", "scope", -1*time.Second)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Wait a bit to ensure expiration
	time.Sleep(100 * time.Millisecond)

	// Try to validate
	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating expired token")
	}
}

func TestJWTManager_ValidateToken_MalformedToken(t *testing.T) {
	secret := "test-secret"
	manager := NewJWTManager(secret)

	// Try to validate malformed tokens
	malformedTokens := []string{
		"not-a-jwt-token",
		"",
		"header.payload", // Missing signature
		"a.b.c.d",        // Too many parts
	}

	for _, token := range malformedTokens {
		_, err := manager.ValidateToken(token)
		if err == nil {
			t.Errorf("expected error for malformed token: %s", token)
		}
	}
}
