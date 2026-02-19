// Package registry provides functionality for managing running service registrations.
// This is a thin wrapper re-exporting from azd-core.
package registry

import core "github.com/jongio/azd-core/registry"

// Re-export types from azd-core/registry.
type ServiceRegistryEntry = core.ServiceRegistryEntry
type ServiceRegistry = core.ServiceRegistry

// Re-export functions from azd-core/registry.
var GetRegistry = core.GetRegistry
