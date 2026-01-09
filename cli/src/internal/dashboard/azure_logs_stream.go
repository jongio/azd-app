// Package dashboard provides API endpoints for the local dashboard.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
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

	// Capture rate limiter early to avoid race with Stop()
	rl := s.rateLimiter

	// Upgrade connection to WebSocket
	rawConn, err := acceptWebSocket(w, r, rl)
	if err != nil {
		if err != http.ErrAbortHandler {
			log.Printf("Azure logs WebSocket upgrade failed: %v", err)
		}
		return
	}

	// Wrap connection with mutex for safe concurrent writes
	// Use request context to properly handle client disconnection
	client := newWSClientWithContext(r.Context(), rawConn)
	conn := &clientConn{client: client}
	clientIP := getClientIP(r)
	defer func() {
		if closeErr := client.closeWithRateLimit(clientIP, rl); closeErr != nil {
			if !isExpectedCloseError(closeErr) {
				log.Printf("Failed to close Azure logs WebSocket: %v", closeErr)
			}
		}
	}()

	// Try to read optional init message with time range configuration
	// Message format: {"type":"init","service":"api","since":"1h"}
	// Use non-blocking read with short timeout
	var initialWindow = 30 * time.Minute // Default
	initServiceName := serviceName

	readCtx, readCancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer readCancel()

	msgType, msgData, err := rawConn.Read(readCtx)
	if err == nil && msgType == websocket.MessageText {
		var initMsg map[string]interface{}
		if jsonErr := json.Unmarshal(msgData, &initMsg); jsonErr == nil {
			// Successfully parsed init message
			if msgTypeStr, ok := initMsg["type"].(string); ok && msgTypeStr == "init" {
				if svc, ok := initMsg["service"].(string); ok && svc != "" {
					initServiceName = svc
				}
				if since, ok := initMsg["since"].(string); ok && since != "" {
					if parsed := parseDuration(since); parsed > 0 {
						initialWindow = parsed
						log.Printf("Azure logs init: service=%s, window=%s", initServiceName, initialWindow)
					}
				}
			}
		}
	}
	// If read times out or fails, just use defaults (backward compatible)

	// Use init message service name if provided, otherwise fall back to query param
	if initServiceName != "" {
		serviceName = initServiceName
	}

	// Track last seen timestamp to avoid duplicates
	lastTimestamp := time.Now().Add(-initialWindow)

	ctx := r.Context()
	log.Printf("Azure logs WebSocket connected for service: %s (realtime=%v, window=%s)", serviceName, realtime, initialWindow)

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

// parseDuration converts time range strings like "1h", "30m", "24h" to time.Duration.
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func streamAzureLogsViaPolling(ctx context.Context, s *Server, serviceName string, conn *clientConn, since time.Time, lastTimestamp *time.Time) {
	// Initialize polling state with exponential backoff
	pollingState := azure.NewPollingState(5 * time.Second)
	lastAttemptTime := time.Now()

	// Sequence tracking for backpressure detection
	var sequenceCounter int64 = 0

	// Create adaptive ticker - will be recreated with new intervals
	ticker := time.NewTicker(pollingState.NextDelay())
	defer ticker.Stop()

	// Send initial status message to indicate streaming is active
	health := pollingState.GetHealth()
	statusMsg := map[string]interface{}{
		"type":   "status",
		"status": health.Status,
		"mode":   "polling",
	}
	if err := conn.writeWebSocketJSON(statusMsg); err != nil {
		log.Printf("Failed to send initial status message: %v", err)
		return
	}

	// Initial fetch
	if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, since, conn, lastTimestamp, &sequenceCounter); err != nil {
		log.Printf("Initial Azure logs fetch failed: %v", err)
		pollingState.RecordFailure(err)
		// Send error status
		sendHealthStatus(conn, pollingState)
	} else {
		pollingState.RecordSuccess()
	}

	for {
		select {
		case <-ticker.C:
			// Check if we should retry based on backoff
			if !pollingState.ShouldRetry(lastAttemptTime) {
				continue // Skip this tick, wait for next interval
			}

			lastAttemptTime = time.Now()
			if err := fetchAndSendAzureLogs(ctx, s.projectDir, serviceName, *lastTimestamp, conn, lastTimestamp, &sequenceCounter); err != nil {
				log.Printf("Azure logs fetch failed: %v", err)
				pollingState.RecordFailure(err)
				sendHealthStatus(conn, pollingState)

				// Adjust ticker interval for exponential backoff
				ticker.Reset(pollingState.NextDelay())
			} else {
				// Success - reset to base interval
				if pollingState.GetHealth().Status != "connected" {
					pollingState.RecordSuccess()
					sendHealthStatus(conn, pollingState)
					ticker.Reset(pollingState.NextDelay())
				} else {
					pollingState.RecordSuccess()
				}
			}
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sendHealthStatus sends connection health status to the WebSocket client.
func sendHealthStatus(conn *clientConn, pollingState *azure.PollingState) {
	health := pollingState.GetHealth()
	statusMsg := map[string]interface{}{
		"type":             "status",
		"status":           health.Status,
		"mode":             "polling",
		"consecutiveFails": health.ConsecutiveFails,
	}
	if health.LastError != "" {
		statusMsg["error"] = health.LastError
	}
	if !health.NextRetry.IsZero() {
		statusMsg["nextRetry"] = health.NextRetry.Format(time.RFC3339)
	}
	if err := conn.writeWebSocketJSON(statusMsg); err != nil {
		log.Printf("Failed to send health status: %v", err)
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

	// Send initial status message to indicate realtime streaming is active
	statusMsg := map[string]interface{}{
		"type":   "status",
		"status": "connected",
		"mode":   "realtime",
	}
	if err := conn.writeWebSocketJSON(statusMsg); err != nil {
		return err
	}

	logsCh := make(chan azure.LogEntry, service.WebSocketLogChannelBuffer)
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

// fetchAndSendAzureLogs fetches logs since lastTimestamp and sends them via WebSocket with sequence numbers.
func fetchAndSendAzureLogs(ctx context.Context, projectDir string, serviceName string, since time.Time, conn *clientConn, lastTimestamp *time.Time, sequenceCounter *int64) error {
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

	// Filter logs newer than last timestamp and send them with sequence numbers
	newTimestamp := *lastTimestamp
	for _, azLog := range azureLogs {
		if azLog.Timestamp.After(since) {
			// Increment sequence for each log entry
			*sequenceCounter++

			entry := service.LogEntry{
				Service:   azLog.Service,
				Message:   azLog.Message,
				Level:     convertAzureLogLevel(azLog.Level),
				Timestamp: azLog.Timestamp,
				Source:    service.LogSourceAzure,
				Sequence:  *sequenceCounter,
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
