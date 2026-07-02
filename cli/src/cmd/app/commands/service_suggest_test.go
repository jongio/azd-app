package commands

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"both empty", "", "", 0},
		{"empty a", "", "abc", 3},
		{"empty b", "abc", "", 3},
		{"identical", "api", "api", 0},
		{"one insertion", "web", "webb", 1},
		{"one deletion", "backendd", "backend", 1},
		{"one substitution", "web", "web", 0},
		{"substitution", "cat", "bat", 1},
		{"multiple edits", "kitten", "sitting", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, levenshtein(tt.a, tt.b))
			// Distance is symmetric.
			assert.Equal(t, tt.want, levenshtein(tt.b, tt.a))
		})
	}
}

func TestSuggestServiceName(t *testing.T) {
	available := []string{"api", "web", "worker"}
	tests := []struct {
		name      string
		requested string
		available []string
		want      string
	}{
		{"exact match returns itself", "api", available, "api"},
		{"case-only mismatch suggests correct casing", "API", available, "api"},
		{"close typo suggests match", "webb", available, "web"},
		{"short typo suggests match", "ap", available, "api"},
		{"far-off name has no suggestion", "database", available, ""},
		{"unrelated short name has no suggestion", "xyz", available, ""},
		{"empty request has no suggestion", "", available, ""},
		{"empty available has no suggestion", "api", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, suggestServiceName(tt.requested, tt.available))
		})
	}
}

func TestResolveServiceName(t *testing.T) {
	available := []string{"api", "web", "worker"}

	t.Run("exact match returns name without error", func(t *testing.T) {
		got, err := resolveServiceName("web", available)
		require.NoError(t, err)
		assert.Equal(t, "web", got)
	})

	t.Run("case-only mismatch is not found but suggests", func(t *testing.T) {
		_, err := resolveServiceName("Web", available)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, `service "Web" not found`)
		assert.Contains(t, msg, "Available services: api, web, worker")
		assert.Contains(t, msg, `Did you mean "web"?`)
	})

	t.Run("close typo includes suggestion", func(t *testing.T) {
		_, err := resolveServiceName("webb", available)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `Did you mean "web"?`)
	})

	t.Run("far-off name omits suggestion but lists services", func(t *testing.T) {
		_, err := resolveServiceName("database", available)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, `service "database" not found`)
		assert.Contains(t, msg, "Available services: api, web, worker")
		assert.NotContains(t, msg, "Did you mean")
	})

	t.Run("empty service list reports no services defined", func(t *testing.T) {
		_, err := resolveServiceName("api", nil)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, `service "api" not found`)
		assert.Contains(t, msg, "No services are defined")
		assert.NotContains(t, msg, "Did you mean")
	})

	t.Run("available names keep original casing", func(t *testing.T) {
		_, err := resolveServiceName("frontent", []string{"Frontend", "Backend"})
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "Frontend")
		assert.Contains(t, msg, "Backend")
		assert.Contains(t, msg, `Did you mean "Frontend"?`)
	})

	t.Run("error message is a single line", func(t *testing.T) {
		_, err := resolveServiceName("webb", available)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "\n")
		assert.False(t, strings.HasSuffix(err.Error(), "\n"))
	})
}

func TestFilterServicesByName(t *testing.T) {
	services := []*serviceinfo.ServiceInfo{
		{Name: "api"},
		{Name: "web"},
		{Name: "worker"},
	}

	t.Run("filters to requested services", func(t *testing.T) {
		got, err := filterServicesByName(services, []string{"web", "api"})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "web", got[0].Name)
		assert.Equal(t, "api", got[1].Name)
	})

	t.Run("unknown name returns suggestion error", func(t *testing.T) {
		_, err := filterServicesByName(services, []string{"webb"})
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, `service "webb" not found`)
		assert.Contains(t, msg, `Did you mean "web"?`)
	})
}
