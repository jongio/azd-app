// Package version provides the build-time version for the CLI.
// This is a shared internal package so that both cmd and internal packages
// can reference the version without creating layer violations.
package version

// Version is set at build time via -ldflags.
var Version = "dev"
