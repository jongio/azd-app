package azure

import (
	"os"
	"testing"
)

func TestGetServiceNameMapExcludesImageName(t *testing.T) {
	// Save and restore original env vars
	origAPI := os.Getenv("SERVICE_API_NAME")
	origAPIImage := os.Getenv("SERVICE_API_IMAGE_NAME")
	origWeb := os.Getenv("SERVICE_WEB_NAME")
	t.Cleanup(func() {
		restoreEnv(t, "SERVICE_API_NAME", origAPI)
		restoreEnv(t, "SERVICE_API_IMAGE_NAME", origAPIImage)
		restoreEnv(t, "SERVICE_WEB_NAME", origWeb)
	})

	t.Setenv("SERVICE_API_NAME", "api-prod")
	t.Setenv("SERVICE_API_IMAGE_NAME", "registry.io/img:v1")
	t.Setenv("SERVICE_WEB_NAME", "web-prod")

	result := getServiceNameMap("")

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(result), result)
	}

	if v, ok := result["api"]; !ok || v != "api-prod" {
		t.Errorf("expected api → api-prod, got %q (present=%v)", v, ok)
	}

	if v, ok := result["web"]; !ok || v != "web-prod" {
		t.Errorf("expected web → web-prod, got %q (present=%v)", v, ok)
	}

	if _, ok := result["api-image"]; ok {
		t.Errorf("expected api-image to NOT be present, but it was found in %v", result)
	}
}

func TestGetServiceNameMapEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantKeys map[string]string // key → expected value
		noKeys   []string          // keys that must NOT be present
	}{
		{
			name: "multi-segment service name",
			envVars: map[string]string{
				"SERVICE_MY_SVC_NAME": "my-svc-prod",
			},
			wantKeys: map[string]string{
				"my-svc": "my-svc-prod",
			},
		},
		{
			name: "service name with resource suffix is kept",
			envVars: map[string]string{
				"SERVICE_MY_RESOURCE_NAME": "my-resource-prod",
			},
			wantKeys: map[string]string{
				"my-resource": "my-resource-prod",
			},
		},
		{
			name: "service name with endpoint suffix is kept",
			envVars: map[string]string{
				"SERVICE_MY_ENDPOINT_NAME": "my-endpoint-prod",
			},
			wantKeys: map[string]string{
				"my-endpoint": "my-endpoint-prod",
			},
		},
		{
			name: "service name with identity suffix is kept",
			envVars: map[string]string{
				"SERVICE_AUTH_IDENTITY_NAME": "auth-identity-prod",
			},
			wantKeys: map[string]string{
				"auth-identity": "auth-identity-prod",
			},
		},
		{
			name: "compound IMAGE suffix is excluded",
			envVars: map[string]string{
				"SERVICE_API_IMAGE_NAME": "registry.io/img:v1",
				"SERVICE_API_NAME":       "api-prod",
			},
			wantKeys: map[string]string{
				"api": "api-prod",
			},
			noKeys: []string{"api-image"},
		},
		{
			name: "empty value is excluded",
			envVars: map[string]string{
				"SERVICE_API_NAME": "",
			},
			wantKeys: map[string]string{},
		},
		{
			name: "quoted value is unquoted",
			envVars: map[string]string{
				"SERVICE_API_NAME": `"api-prod"`,
			},
			wantKeys: map[string]string{
				"api": "api-prod",
			},
		},
		{
			name: "non-service env vars are ignored",
			envVars: map[string]string{
				"OTHER_VAR":        "ignored",
				"SERVICE_API_NAME": "api-prod",
			},
			wantKeys: map[string]string{
				"api": "api-prod",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore all env vars used in this test case
			originals := make(map[string]string)
			for k := range tc.envVars {
				originals[k] = os.Getenv(k)
			}
			t.Cleanup(func() {
				for k, orig := range originals {
					restoreEnv(t, k, orig)
				}
			})

			// Set test env vars
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			result := getServiceNameMap("")

			for wantKey, wantVal := range tc.wantKeys {
				got, ok := result[wantKey]
				if !ok {
					t.Errorf("expected key %q to be present, got map: %v", wantKey, result)
				} else if got != wantVal {
					t.Errorf("expected %q → %q, got %q", wantKey, wantVal, got)
				}
			}

			for _, noKey := range tc.noKeys {
				if _, ok := result[noKey]; ok {
					t.Errorf("expected key %q to NOT be present, but found in %v", noKey, result)
				}
			}
		})
	}
}

// restoreEnv restores an environment variable to its original value, or unsets it if it was empty.
func restoreEnv(t *testing.T, key, origValue string) {
	t.Helper()
	if origValue == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, origValue)
	}
}
