// Package containerauth provides utilities for building and injecting the azd auth
// shim into containers for DefaultAzureCredential support.
package containerauth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildShim cross-compiles the azd auth shim for Linux.
// It returns the path to the built binary. The binary is cached across calls.
// The goarch parameter should be "amd64" or "arm64".
func BuildShim(goarch string) (string, error) {
	// Always rebuild if arch changes — for simplicity, build each time
	// In practice, we could cache per-arch
	tmpDir, err := os.MkdirTemp("", "azd-auth-shim-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "azd")

	// Find the shim source directory
	shimSrc, err := findShimSource()
	if err != nil {
		return "", fmt.Errorf("failed to find shim source: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", outputPath, "-trimpath", "-ldflags=-s -w", ".")
	cmd.Dir = shimSrc
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+goarch,
		"CGO_ENABLED=0",
		"GOWORK=off", // Shim has its own go.mod; ignore parent workspace
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build shim: %s: %w", string(output), err)
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

// findShimSource locates the shim source directory relative to the running binary
// or via a well-known path.
func findShimSource() (string, error) {
	// Try relative to the executable
	exe, err := os.Executable()
	if err == nil {
		// Walk up to find the cli directory
		dir := filepath.Dir(exe)
		for i := 0; i < 10; i++ {
			candidate := filepath.Join(dir, "cli", "src", "internal", "containerauth", "shim")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			candidate = filepath.Join(dir, "src", "internal", "containerauth", "shim")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Try via GOPATH or module cache — for dev, the source is right next to us
	// Try current working directory patterns
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "cli", "src", "internal", "containerauth", "shim"),
		filepath.Join(cwd, "src", "internal", "containerauth", "shim"),
		filepath.Join(cwd, "internal", "containerauth", "shim"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("shim source not found; expected at cli/src/internal/containerauth/shim/")
}
