// Package constants provides shared constants for the azd-app CLI.
package constants

// Error message and output limits

// MaxStderrLength is the maximum length of stderr output to capture before truncating.
// This prevents memory issues with extremely verbose command output.
const MaxStderrLength = 10000 // 10KB

// MaxErrorMessageLength is the maximum length of error messages before truncating.
// This ensures error messages remain readable and don't overflow logs.
const MaxErrorMessageLength = 500
