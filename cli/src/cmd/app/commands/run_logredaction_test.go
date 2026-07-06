package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestRegisterLogRedactionFromConfig(t *testing.T) {
	t.Cleanup(service.ResetLogRedaction)

	azureYaml := &service.AzureYaml{
		Logs: &service.LogsConfig{
			Redaction: &service.LogRedactionConfig{
				Patterns: []string{`tok-[0-9]{4}`},
				Literals: []string{"literal-secret"},
			},
		},
	}

	registerLogRedaction(azureYaml)

	masked := service.MaskSecretsInLogLine("tok-1234 and literal-secret")
	assert.NotContains(t, masked, "tok-1234")
	assert.NotContains(t, masked, "literal-secret")
}

func TestRegisterLogRedactionNilSafe(t *testing.T) {
	t.Cleanup(service.ResetLogRedaction)

	// None of these should panic or register anything.
	registerLogRedaction(nil)
	registerLogRedaction(&service.AzureYaml{})
	registerLogRedaction(&service.AzureYaml{Logs: &service.LogsConfig{}})

	assert.Equal(t, "plain line", service.MaskSecretsInLogLine("plain line"))
}
