package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironmentWithSourcesPrecedence(t *testing.T) {
	t.Setenv("PROV_LAYER_KEY", "from-os")

	azureEnv := map[string]string{
		"PROV_LAYER_KEY": "from-azd",
		"PROV_AZD_ONLY":  "azd-value",
	}
	svc := Service{Environment: Environment{"PROV_LAYER_KEY": "from-service"}}

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), svc, azureEnv, "", nil, nil)
	require.NoError(t, err)

	// Without inline env, azure.yaml wins the layered key.
	assert.Equal(t, "from-service", env["PROV_LAYER_KEY"])
	p := prov["PROV_LAYER_KEY"]
	assert.Equal(t, EnvSourceService, p.Source)
	assert.Equal(t, []EnvSource{EnvSourceOS, EnvSourceAzd}, p.Overrides)

	// A key set only by azd has no overrides.
	azdOnly := prov["PROV_AZD_ONLY"]
	assert.Equal(t, EnvSourceAzd, azdOnly.Source)
	assert.Empty(t, azdOnly.Overrides)
}

func TestResolveEnvironmentWithSourcesDotEnvOverridesAzd(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("PROV_DOTENV_KEY=from-dotenv\n"), 0o600))

	azureEnv := map[string]string{"PROV_DOTENV_KEY": "from-azd"}

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), Service{}, azureEnv, envPath, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "from-dotenv", env["PROV_DOTENV_KEY"])
	p := prov["PROV_DOTENV_KEY"]
	assert.Equal(t, EnvSourceDotEnv, p.Source)
	assert.Equal(t, []EnvSource{EnvSourceAzd}, p.Overrides)
}

func TestResolveEnvironmentWithSourcesServiceURLOverridesDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("PROV_URL_KEY=from-dotenv\n"), 0o600))

	serviceURLs := map[string]string{"PROV_URL_KEY": "http://localhost:3000"}

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), Service{}, nil, envPath, serviceURLs, nil)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:3000", env["PROV_URL_KEY"])
	p := prov["PROV_URL_KEY"]
	assert.Equal(t, EnvSourceServiceURL, p.Source)
	assert.Equal(t, []EnvSource{EnvSourceDotEnv}, p.Overrides)
}

func TestResolveEnvironmentWithSourcesInlineOverridesService(t *testing.T) {
	svc := Service{Environment: Environment{"PROV_INLINE_KEY": "from-service"}}
	inlineEnv := map[string]string{"PROV_INLINE_KEY": "from-inline"}

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), svc, nil, "", nil, inlineEnv)
	require.NoError(t, err)

	assert.Equal(t, "from-inline", env["PROV_INLINE_KEY"])
	p := prov["PROV_INLINE_KEY"]
	assert.Equal(t, EnvSourceInline, p.Source)
	assert.Equal(t, []EnvSource{EnvSourceService}, p.Overrides)
}

func TestResolveEnvironmentWithSourcesInlineOverridesDotEnvAndEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("PROV_INLINE_CHAIN=from-dotenv\n"), 0o600))

	envFileVars := map[string]string{"PROV_INLINE_CHAIN": "from-env-file"}
	inlineEnv := map[string]string{"PROV_INLINE_CHAIN": "from-inline"}

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), Service{}, nil, envPath, envFileVars, inlineEnv)
	require.NoError(t, err)

	assert.Equal(t, "from-inline", env["PROV_INLINE_CHAIN"])
	p := prov["PROV_INLINE_CHAIN"]
	assert.Equal(t, EnvSourceInline, p.Source)
	assert.Equal(t, []EnvSource{EnvSourceDotEnv, EnvSourceServiceURL}, p.Overrides)
}

func TestResolveEnvironmentMatchesWithSources(t *testing.T) {
	t.Setenv("PROV_PARITY_KEY", "os-value")
	azureEnv := map[string]string{"PROV_PARITY_EXTRA": "azd-value"}
	svc := Service{Environment: Environment{"PROV_PARITY_SVC": "svc-value"}}

	plain, err := ResolveEnvironment(context.Background(), svc, azureEnv, "", nil, nil)
	require.NoError(t, err)

	withSources, _, err := ResolveEnvironmentWithSources(context.Background(), svc, azureEnv, "", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, plain, withSources)
}

func TestResolveEnvironmentWithSourcesDotEnvLoadError(t *testing.T) {
	// A missing .env path makes LoadDotEnv fail, which must surface as an error
	// with nil results rather than a partial environment.
	missing := filepath.Join(t.TempDir(), "missing.env")

	env, prov, err := ResolveEnvironmentWithSources(context.Background(), Service{}, nil, missing, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load .env file")
	assert.Nil(t, env)
	assert.Nil(t, prov)
}
