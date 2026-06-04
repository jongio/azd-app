package dashboard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
)

// getOrCreateConfigClient returns the cached config client, creating it lazily if needed.
func (s *Server) getOrCreateConfigClient() azdconfig.ConfigClient {
	if s.configClient != nil {
		return s.configClient
	}

	client, err := azdconfig.NewClient(context.Background())
	if err != nil {
		slog.Debug("failed to create azdconfig client, using in-memory fallback", "error", err)
		s.configClient = azdconfig.NewInMemoryClient()
		return s.configClient
	}

	s.configClient = client
	return client
}

// registerPortInConfig stores the dashboard port in azdconfig for discovery by other commands.
func (s *Server) registerPortInConfig(port int) {
	client := s.getOrCreateConfigClient()

	projectHash := azdconfig.ProjectHash(s.projectDir)
	if err := client.SetDashboardPort(projectHash, port); err != nil {
		slog.Debug("failed to register dashboard port in config", "error", err)
	} else {
		slog.Debug("registered dashboard port in config", "port", port, "projectHash", projectHash)
	}

	// Also write port file for cross-process discovery (doesn't depend on gRPC host)
	writePortFile(s.projectDir, port)
}

// clearPortFromConfig removes the dashboard port from azdconfig.
func (s *Server) clearPortFromConfig() {
	client := s.getOrCreateConfigClient()

	projectHash := azdconfig.ProjectHash(s.projectDir)
	if err := client.ClearDashboardPort(projectHash); err != nil {
		slog.Debug("failed to clear dashboard port from config", "error", err)
	} else {
		slog.Debug("cleared dashboard port from config", "projectHash", projectHash)
	}

	// Also remove port file
	removePortFile(s.projectDir)
}

// nonceDirBase is the base directory for per-project nonce state files.
// Defaults to ~/.azd/azd-app. Override in tests to avoid touching the real home directory.
var nonceDirBase string

// nonceStateDir returns the directory that holds the nonce file for a project.
func nonceStateDir(projectHash string) (string, error) {
	base := nonceDirBase
	if base == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		base = filepath.Join(homeDir, ".azd", "azd-app")
	}
	return filepath.Join(base, projectHash), nil
}

// loadOrCreateNonce loads the 128-bit random nonce for a project from
// ~/.azd/azd-app/{hash}/nonce, generating and persisting a new one if absent.
// The nonce is a 32-character hex string (16 bytes / 128 bits of crypto/rand entropy).
func loadOrCreateNonce(projectHash string) (string, error) {
	dir, err := nonceStateDir(projectHash)
	if err != nil {
		return "", err
	}
	nonceFile := filepath.Join(dir, "nonce")

	// Read an existing nonce.
	if data, err := os.ReadFile(nonceFile); err == nil {
		if nonce := strings.TrimSpace(string(data)); len(nonce) == 32 {
			return nonce, nil
		}
		// Corrupt or truncated — fall through to regenerate.
	}

	// Generate 128 bits of randomness.
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(raw[:])

	// Persist the nonce: directory 0o700, file 0o600.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create nonce state directory: %w", err)
	}
	if err := os.WriteFile(nonceFile, []byte(nonce), 0o600); err != nil {
		return "", fmt.Errorf("write nonce state file: %w", err)
	}

	return nonce, nil
}

// portFilePath returns the path to the dashboard port file for a project.
// The file name combines the project hash with a per-project random nonce so that
// the path is unpredictable even when the project directory is known (CWE-340).
// The nonce is persisted in ~/.azd/azd-app/{hash}/nonce so that azd app stop
// can rediscover the correct port file across process restarts.
func portFilePath(projectDir string) (string, error) {
	hash := azdconfig.ProjectHash(projectDir)
	nonce, err := loadOrCreateNonce(hash)
	if err != nil {
		return "", err
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf(".azd-app-dashboard-%s-%s.port", hash, nonce)), nil
}

// cleanupLegacyPortFile removes the old predictable port file
// (.azd-app-dashboard-{hash}.port) written before the nonce was introduced.
// Called opportunistically on every write/remove to eliminate stale files.
func cleanupLegacyPortFile(projectDir string) {
	hash := azdconfig.ProjectHash(projectDir)
	legacy := filepath.Join(os.TempDir(), fmt.Sprintf(".azd-app-dashboard-%s.port", hash))
	_ = os.Remove(legacy)
}

// writePortFile writes the dashboard port to a file for cross-process discovery.
func writePortFile(projectDir string, port int) {
	path, err := portFilePath(projectDir)
	if err != nil {
		slog.Debug("failed to resolve dashboard port file path", "error", err)
		return
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(port)), 0o600); err != nil {
		slog.Debug("failed to write dashboard port file", "path", path, "error", err)
	}
	cleanupLegacyPortFile(projectDir)
}

// removePortFile removes the dashboard port file.
func removePortFile(projectDir string) {
	path, err := portFilePath(projectDir)
	if err != nil {
		slog.Debug("failed to resolve dashboard port file path for removal", "error", err)
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Debug("failed to remove dashboard port file", "path", path, "error", err)
	}
	cleanupLegacyPortFile(projectDir)
}

// ReadPortFile reads the dashboard port from the port file for a project directory.
// Returns 0 if the file doesn't exist or can't be read.
func ReadPortFile(projectDir string) int {
	path, err := portFilePath(projectDir)
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return port
}

// generatePreferredPort returns a preferred port for the dashboard server.
func (s *Server) generatePreferredPort(portMgr *portmanager.PortManager) (int, error) {
	// Check for existing persisted port first to maintain URL consistency across runs
	if existingPort, exists := portMgr.GetAssignment(constants.DashboardServiceName); exists && existingPort > 0 {
		// Use persisted port as preferred - same workspace gets same dashboard URL
		return existingPort, nil
	}

	// First run: generate random port in dashboard range (40000-49999)
	// This range is typically used for ephemeral/dynamic ports to avoid common conflicts
	nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random port: %w", err)
	}
	return 40000 + int(nBig.Int64()), nil
}

// retryWithAlternativePort attempts to start the server on an alternative port.
func (s *Server) retryWithAlternativePort(portMgr *portmanager.PortManager) (int, error) {
	// Release the failed port assignment
	if err := portMgr.ReleasePort(constants.DashboardServiceName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Searching for an available dashboard port...\n")

	// Try to find a new port in the higher range with randomization
	for attempt := 0; attempt < 15; attempt++ {
		var preferredPort int
		if attempt < 5 {
			// First 5 attempts: random ports in 40000-49999 range
			nBig, err := rand.Int(rand.Reader, big.NewInt(10000))
			if err != nil {
				continue
			}
			preferredPort = 40000 + int(nBig.Int64())
		} else {
			// After 5 failed random attempts, try sequential ports
			preferredPort = 40000 + (attempt * 100)
		}

		// Use port reservation to prevent TOCTOU race
		reservation, err := portMgr.FindAndReservePort(constants.DashboardServiceName, preferredPort)
		if err != nil {
			continue
		}

		port := reservation.Port
		s.port = port
		s.server = &http.Server{
			Addr:              fmt.Sprintf("127.0.0.1:%d", port),
			Handler:           s.buildHandler(),
			ReadHeaderTimeout: 10 * time.Second,
		}

		// Release reservation just before binding - this is the atomic handoff
		_ = reservation.Release()

		errChan := make(chan error, 1)
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("dashboard server error on alternative port", "error", err)
				errChan <- err
			}
		}()

		// Monitor for server errors in background
		go func(port int) {
			select {
			case err := <-errChan:
				if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
					slog.Warn("dashboard server encountered port conflict", "port", port, "error", err)
				} else {
					slog.Error("dashboard server encountered error after startup", "port", port, "error", err)
				}
			case <-s.stopChan:
				return
			}
		}(port)

		time.Sleep(100 * time.Millisecond)

		select {
		case <-errChan:
			// This port also failed, try next
			if err := portMgr.ReleasePort(constants.DashboardServiceName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to release port: %v\n", err)
			}
			continue
		default:
			// Successfully started - register the new port in azdconfig
			s.registerPortInConfig(port)
			fmt.Fprintf(os.Stderr, "✓ Dashboard started on alternative port %d\n\n", port)
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to find available port for dashboard after 15 attempts")
}
