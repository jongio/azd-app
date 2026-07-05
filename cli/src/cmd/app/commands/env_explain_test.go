package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestJoinSourcesHighestFirst(t *testing.T) {
	// Overrides are recorded lowest-first; display reverses to highest-first.
	got := joinSourcesHighestFirst([]service.EnvSource{service.EnvSourceOS, service.EnvSourceAzd, service.EnvSourceDotEnv})
	assert.Equal(t, ".env, azd, os", got)
}

func TestJoinSourcesHighestFirstSingle(t *testing.T) {
	assert.Equal(t, "azd", joinSourcesHighestFirst([]service.EnvSource{service.EnvSourceAzd}))
}

func TestJoinSourcesHighestFirstEmpty(t *testing.T) {
	assert.Equal(t, "", joinSourcesHighestFirst(nil))
}
