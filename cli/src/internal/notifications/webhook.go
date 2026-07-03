// Package notifications provides the event pipeline for routing state changes to notifications
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
)

// defaultWebhookTimeout bounds each webhook request so a slow or unresponsive
// endpoint cannot stall the notification pipeline or other handlers.
const defaultWebhookTimeout = 5 * time.Second

// webhookStatePayload is the JSON representation of a service state transition.
type webhookStatePayload struct {
	Status string `json:"status"`
	Health string `json:"health"`
	PID    int    `json:"pid,omitempty"`
	Port   int    `json:"port,omitempty"`
}

// WebhookEventPayload is the JSON body POSTed to the configured webhook URL.
type WebhookEventPayload struct {
	Type      string               `json:"type"`
	Service   string               `json:"service"`
	Severity  string               `json:"severity"`
	Message   string               `json:"message"`
	Timestamp time.Time            `json:"timestamp"`
	OldState  *webhookStatePayload `json:"oldState,omitempty"`
	NewState  *webhookStatePayload `json:"newState,omitempty"`
}

// WebhookHandler POSTs qualifying notification events as JSON to a configured
// URL. It applies the same severity filter and rate limit as the OS
// notification handler and fails soft: delivery errors are returned for the
// pipeline to log and never block other handlers beyond the request timeout.
type WebhookHandler struct {
	url      string
	headers  map[string]string
	config   *config.NotificationPreferences
	client   *http.Client
	lastSent map[string]time.Time
	mu       sync.Mutex
}

// NewWebhookHandler builds a webhook handler from notification preferences.
// It returns nil when webhook delivery is not configured, so callers can skip
// registration without a separate enabled check.
func NewWebhookHandler(cfg *config.NotificationPreferences) *WebhookHandler {
	if cfg == nil || !cfg.WebhookConfigured() {
		return nil
	}
	return &WebhookHandler{
		url:      cfg.GetWebhookURL(),
		headers:  cfg.GetWebhookHeaders(),
		config:   cfg,
		client:   &http.Client{Timeout: defaultWebhookTimeout},
		lastSent: make(map[string]time.Time),
	}
}

// Handle POSTs the event to the webhook URL when it passes the severity filter
// and rate limit. Delivery failures are returned so the pipeline can log them.
func (h *WebhookHandler) Handle(ctx context.Context, event Event) error {
	if !h.config.ShouldNotify(event.ServiceName, event.Severity) {
		return nil
	}

	// Rate limiting mirrors the OS notification handler: dedupe by
	// service and event type within the configured window.
	h.mu.Lock()
	key := fmt.Sprintf("%s:%s", event.ServiceName, event.Type)
	if lastSent, ok := h.lastSent[key]; ok {
		if time.Since(lastSent) < h.config.GetRateLimitDuration() {
			h.mu.Unlock()
			return nil
		}
	}
	h.lastSent[key] = time.Now()
	h.mu.Unlock()

	body, err := json.Marshal(buildWebhookPayload(event))
	if err != nil {
		return fmt.Errorf("failed to encode webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// buildWebhookPayload maps a pipeline Event to its JSON webhook representation.
func buildWebhookPayload(event Event) WebhookEventPayload {
	return WebhookEventPayload{
		Type:      string(event.Type),
		Service:   event.ServiceName,
		Severity:  event.Severity,
		Message:   event.Message,
		Timestamp: event.Timestamp,
		OldState:  toWebhookState(event.OldState),
		NewState:  toWebhookState(event.NewState),
	}
}

// toWebhookState converts a monitor service state to its webhook payload form.
func toWebhookState(s *monitor.ServiceState) *webhookStatePayload {
	if s == nil {
		return nil
	}
	return &webhookStatePayload{
		Status: s.Status,
		Health: s.Health,
		PID:    s.PID,
		Port:   s.Port,
	}
}
