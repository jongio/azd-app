package commands

import (
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

// registerLogRedaction installs any project-specific log redaction rules
// declared under logs.redaction in azure.yaml. Invalid regular expressions are
// skipped with a warning so the run still starts. With no config present, the
// built-in redaction patterns continue to apply unchanged.
func registerLogRedaction(azureYaml *service.AzureYaml) {
	if azureYaml == nil || azureYaml.Logs == nil || azureYaml.Logs.Redaction == nil {
		return
	}
	cfg := azureYaml.Logs.Redaction

	errs := service.RegisterLogRedaction(cfg.Patterns, cfg.Literals)
	for _, err := range errs {
		cliout.Warning("Skipping log redaction rule: %v", err)
	}

	if !cliout.IsJSON() && (len(cfg.Patterns) > 0 || len(cfg.Literals) > 0) {
		cliout.Info("Applying custom log redaction rules from azure.yaml.")
	}
}
