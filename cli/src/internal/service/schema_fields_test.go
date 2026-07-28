package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestV11SchemaDocumentsContainerFields verifies the shipped v1.1 JSON schema
// declares the new container fields (volumes, pull_policy, array-form command)
// so editor tooling validates and completes them. It is a structural check of
// the schema file itself (no external json-schema dependency).
func TestV11SchemaDocumentsContainerFields(t *testing.T) {
	// service package -> ../../../.. == repo root
	schemaPath := filepath.Join("..", "..", "..", "..", "schemas", "v1.1", "azure.yaml.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Skipf("schema file not found at %s: %v", schemaPath, err)
	}

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc), "schema must be valid JSON")

	props := serviceProps(t, doc)

	// volumes: array of strings
	vol, ok := props["volumes"].(map[string]any)
	require.True(t, ok, "schema is missing service.volumes")
	assert.Equal(t, "array", vol["type"])

	// pull_policy: enum missing/always/never
	pp, ok := props["pull_policy"].(map[string]any)
	require.True(t, ok, "schema is missing service.pull_policy")
	enum, ok := pp["enum"].([]any)
	require.True(t, ok, "pull_policy must have an enum")
	assert.ElementsMatch(t, []any{"missing", "always", "never"}, enum)

	// command: oneOf string | array of strings
	cmd, ok := props["command"].(map[string]any)
	require.True(t, ok, "schema is missing service.command")
	_, hasOneOf := cmd["oneOf"]
	assert.True(t, hasOneOf, "command must accept string or array (oneOf)")
}

func serviceProps(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	defs, ok := doc["definitions"].(map[string]any)
	require.True(t, ok, "schema missing definitions")
	svc, ok := defs["service"].(map[string]any)
	require.True(t, ok, "schema missing definitions.service")
	props, ok := svc["properties"].(map[string]any)
	require.True(t, ok, "schema missing definitions.service.properties")
	return props
}
