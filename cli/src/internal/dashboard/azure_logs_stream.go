// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// handleAzureLogsStream handles WebSocket streaming of Azure logs via polling.
// WS /api/azure/logs/stream?service=<name>
func (s *Server) handleAzureLogsStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	realtimeParam := r.URL.Query().Get("realtime")

	realtime := false
	if realtimeParam != "" {
		realtime = parseBoolQueryParam(realtimeParam)
	} else {
		// Default from azure.yaml when not explicitly specified
		classificationsMu.RLock()
		azureYaml, err := loadAzureYaml(s.projectDir)
		classificationsMu.RUnlock()
		if err == nil && azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
			realtime = azureYaml.Logs.Analytics.Realtime
		}
	}

	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("Azure logs WebSocket upgrade failed: %v", err)
		}
		return
	}

	// Wrap connection with mutex for safe concurrent writes
	client := newWSClient(rawConn)
	conn := &clientConn{client: client}
	defer client.close()

	// Track last seen timestamp to avoid duplicates
	lastTimestamp := time.Now().Add(-30 * time.Minute) // Start with 30m ago

	ctx := r.Context()
	log.Printf("Azure logs WebSocket connected for service: %s (realtime=%v)", serviceName, realtime)

	// If realtime is requested and a specific service is selected, attempt service-specific streaming.
	if realtime && serviceName != "" {
		if err := streamAzureLogsRealtime(ctx, s.projectDir, serviceName, conn); err != nil {
			log.Printf("Azure realtime streaming failed; falling back to polling: %v", err)
			// fall back to polling below
		} else {
			return
		}
	}

	// Polling fallback (default behavior)
	streamAzureLogsViaPolling(ctx, s, serviceName, conn, lastTimestamp, &lastTimestamp)
}

func parseBoolQueryParam(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func streamAzureLogsViaPolling(ctx context.Context, s *Server, serviceName string, conn *clientConn, since time.Time, lastTimestamp *time.Time) {
	// Poll for new logs every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial fetch
	if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, since, conn, lastTimestamp); err != nil {
		log.Printf("Initial Azure logs fetch failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, *lastTimestamp, conn, lastTimestamp); err != nil {
				log.Printf("Azure logs fetch failed: %v", err)
				return
			}
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func streamAzureLogsRealtime(ctx context.Context, projectDir string, serviceName string, conn *clientConn) error {
	cred, err := azure.NewAzureCredential()
	if err != nil {
		return err
	}

	discovery := azure.NewResourceDiscovery(cred, projectDir)
	resource, err := discovery.GetResource(ctx, serviceName)
	if err != nil {
		return err
	}

	streamer, err := azure.NewRealtimeStreamer(resource.ResourceType, azure.StreamerConfig{
		SubscriptionID: resource.SubscriptionID,
		ResourceGroup:  resource.ResourceGroup,
		ResourceName:   resource.Name,
		ServiceName:    serviceName,
		Credential:     cred,
	})
	if err != nil {
		return err
	}
	defer func() {
		if stopErr := streamer.Stop(); stopErr != nil {
			log.Printf("Error stopping streamer: %v", stopErr)
		}
	}()

	logsCh := make(chan azure.LogEntry, 250)
	errCh := make(chan error, 1)

	go func() {
		errCh <- streamer.Start(ctx, logsCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err == nil {
				return nil
			}
			return err
		case azLog, ok := <-logsCh:
			if !ok {
				return nil
			}
			entry := service.LogEntry{
				Service:   azLog.Service,
				Message:   azLog.Message,
				Level:     convertAzureLogLevel(azLog.Level),
				Timestamp: azLog.Timestamp,
				Source:    service.LogSourceAzure,
				AzureMetadata: &service.AzureLogMetadata{
					ResourceType:  azLog.ResourceType,
					ContainerName: azLog.ContainerName,
					InstanceID:    azLog.InstanceID,
				},
			}
			if err := conn.writeWebSocketJSON(entry); err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("Azure logs WebSocket write error: %v", err)
				}
				return err
			}
		}
	}
}

// fetchAndSendAzureLogs fetches logs since lastTimestamp and sends them via WebSocket.
func fetchAndSendAzureLogs(ctx context.Context, projectDir string, serviceName string, since time.Time, conn *clientConn, lastTimestamp *time.Time) error {
	var services []string
	if serviceName != "" {
		services = []string{serviceName}
	}

	config := azure.StandaloneLogsConfig{
		ProjectDir: projectDir,
		Services:   services,
		Since:      time.Since(since),
		Limit:      100,
	}

	azureLogs, err := fetchAzureLogsStandalone(ctx, config)
	if err != nil {
		// Send error message to client
		errMsg := map[string]string{
			"error": fmt.Sprintf("Failed to fetch Azure logs: %v", err),
		}
		if writeErr := conn.writeWebSocketJSON(errMsg); writeErr != nil {
			return writeErr
		}
		return err
	}

	// Filter logs newer than last timestamp and send them
	newTimestamp := *lastTimestamp
	for _, azLog := range azureLogs {
		if azLog.Timestamp.After(since) {
			entry := service.LogEntry{
				Service:   azLog.Service,
				Message:   azLog.Message,
				Level:     convertAzureLogLevel(azLog.Level),
				Timestamp: azLog.Timestamp,
				Source:    service.LogSourceAzure,
				AzureMetadata: &service.AzureLogMetadata{
					ResourceType:  azLog.ResourceType,
					ContainerName: azLog.ContainerName,
					InstanceID:    azLog.InstanceID,
				},
			}

			if err := conn.writeWebSocketJSON(entry); err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("Azure logs WebSocket write error: %v", err)
				}
				return err
			}

			// Track latest timestamp
			if azLog.Timestamp.After(newTimestamp) {
				newTimestamp = azLog.Timestamp
			}
		}
	}

	// Update last timestamp
	if newTimestamp.After(*lastTimestamp) {
		*lastTimestamp = newTimestamp
	}

	return nil
}
