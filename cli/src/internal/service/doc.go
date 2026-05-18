// Package service provides runtime detection, orchestration, and lifecycle
// management for application services defined in azure.yaml.
//
// This package encompasses several internal domains that work together:
//
//   - Detection: identifies services from project configuration and filesystem
//   - Orchestration: coordinates service start/stop/restart with dependency order
//   - Process management: monitors and controls service processes (service_process.go)
//   - Logging: captures and manages structured service log output
//   - Health: integrates with healthcheck package for service health reporting
//
// Note: The internal/executor package provides low-level command execution
// (exec.Command wrappers). Service lifecycle operations live here instead.
//
// Future work: Consider extracting detection and logging into sub-packages
// once the 43+ importers can be migrated incrementally (see issue #190).
package service