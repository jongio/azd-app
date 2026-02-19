// Package yamlutil provides utilities for manipulating YAML files while preserving
// formatting, comments, and structure. This is a thin wrapper re-exporting from azd-core.
package yamlutil

import coreyaml "github.com/jongio/azd-core/yamlutil"

// Re-export types from azd-core/yamlutil.
type ArrayAppendOptions = coreyaml.ArrayAppendOptions

// Re-export functions from azd-core/yamlutil.
var (
	AppendToArraySection    = coreyaml.AppendToArraySection
	UpdateServiceLogsConfig = coreyaml.UpdateServiceLogsConfig
	UpdateServicePort       = coreyaml.UpdateServicePort
)
