// Package containerauth provides utilities for building and injecting the azd auth
// shim into containers for DefaultAzureCredential support.
package containerauth

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Pre-compiled shim binaries for Linux containers.
// These are built during CI/release by `mage shimBuild` and embedded into the CLI binary.
// To rebuild locally: mage shimBuild
//
//go:embed bin/azd-linux-amd64 bin/azd-linux-arm64
var shimBinaries embed.FS

// ExtractShim extracts the embedded shim binary for the given architecture to a temp directory.
// It returns the path to the extracted binary. The caller is responsible for cleanup.
// The goarch parameter should be "amd64" or "arm64".
func ExtractShim(goarch string) (string, error) {
	name := fmt.Sprintf("bin/azd-linux-%s", goarch)
	data, err := shimBinaries.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("embedded shim binary not found for %s: %w", goarch, err)
	}

	tmpDir, err := os.MkdirTemp("", "azd-auth-shim-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "azd")
	if err := os.WriteFile(outputPath, data, 0o755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write shim binary: %w", err)
	}

	return outputPath, nil
}

// DetectContainerArch returns the GOARCH for the container runtime.
// On Apple Silicon Macs, containers default to arm64.
// On everything else, default to amd64.
func DetectContainerArch() string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "amd64"
}

// DetectContainerHost returns the hostname containers use to reach the host.
// - Docker Desktop (Win/Mac): host.docker.internal (built-in)
// - Native Linux Docker: host.docker.internal (needs --add-host)
// - Podman: host.containers.internal
func DetectContainerHost() string {
	// Check for Podman
	if _, err := exec.LookPath("podman"); err == nil {
		// If podman is found and docker is a symlink to podman, use podman host
		if out, err := exec.Command("docker", "--version").Output(); err == nil {
			if strings.Contains(strings.ToLower(string(out)), "podman") {
				return "host.containers.internal"
			}
		}
	}
	return "host.docker.internal"
}

// NeedsExtraHosts returns true if the container runtime needs --add-host for host access.
// This is needed on native Linux Docker (not Docker Desktop).
func NeedsExtraHosts() bool {
	return runtime.GOOS == "linux"
}

// ExtraHostsEntries returns the --add-host entries needed for the current platform.
func ExtraHostsEntries() []string {
	if NeedsExtraHosts() {
		return []string{"host.docker.internal:host-gateway"}
	}
	return nil
}
