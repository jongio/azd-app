package dashboard

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
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
			Handler:           s.mux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		// Release reservation just before binding - this is the atomic handoff
		_ = reservation.Release()

		errChan := make(chan error, 1)
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Dashboard server error on alternative port: %v", err)
				errChan <- err
			}
		}()

		// Monitor for server errors in background
		go func(port int) {
			select {
			case err := <-errChan:
				if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
					log.Printf("Dashboard server encountered port conflict on port %d: %v", port, err)
				} else {
					log.Printf("Dashboard server encountered error after startup on port %d: %v", port, err)
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
