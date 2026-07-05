package netexposure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectFlagsWildcardBinds(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"host ipv4 wildcard", "HOST", "0.0.0.0"},
		{"host ipv4 with port", "HOST", "0.0.0.0:8080"},
		{"bind ipv6 wildcard", "BIND_ADDRESS", "::"},
		{"bind ipv6 bracket port", "LISTEN_ADDR", "[::]:9000"},
		{"star host", "SERVER_HOST", "*"},
		{"aspnet plus form", "ASPNETCORE_URLS", "http://+:5000"},
		{"aspnet star form", "ASPNETCORE_URLS", "http://*:5000"},
		{"aspnet ipv4 wildcard", "ASPNETCORE_URLS", "http://0.0.0.0:5000"},
		{"aspnet multi url one wildcard", "ASPNETCORE_URLS", "http://localhost:5000;http://0.0.0.0:5001"},
		{"generic url wildcard", "PUBLIC_ENDPOINT", "https://0.0.0.0:443/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, Inspect(tt.key, tt.value), "expected %q=%q to be flagged", tt.key, tt.value)
		})
	}
}

func TestInspectIgnoresSafeBinds(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"empty", "HOST", ""},
		{"loopback ipv4", "HOST", "127.0.0.1"},
		{"loopback ipv4 port", "HOST", "127.0.0.1:8080"},
		{"localhost", "HOST", "localhost"},
		{"loopback ipv6", "BIND_ADDRESS", "::1"},
		{"specific ip", "HOST", "192.168.1.10"},
		{"non bind key wildcard value", "REGION", "0.0.0.0"},
		{"aspnet localhost only", "ASPNETCORE_URLS", "http://localhost:5000"},
		{"url localhost", "PUBLIC_ENDPOINT", "https://localhost:443/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, Inspect(tt.key, tt.value), "expected %q=%q to be ignored", tt.key, tt.value)
		})
	}
}

func TestScanEnvReturnsSortedFindings(t *testing.T) {
	env := map[string]string{
		"HOST":         "0.0.0.0",
		"ADMIN_HOST":   "0.0.0.0",
		"SERVICE_NAME": "orders-api",
		"BIND_ADDRESS": "127.0.0.1",
	}

	findings := ScanEnv("azure.yaml (service: api)", env)

	require.Len(t, findings, 2)
	assert.Equal(t, "ADMIN_HOST", findings[0].Key)
	assert.Equal(t, "HOST", findings[1].Key)
	assert.Equal(t, "0.0.0.0", findings[0].Value)
	assert.Equal(t, "azure.yaml (service: api)", findings[0].Source)
}

func TestScanEnvEmpty(t *testing.T) {
	assert.Empty(t, ScanEnv("x", nil))
	assert.Empty(t, ScanEnv("x", map[string]string{}))
}
