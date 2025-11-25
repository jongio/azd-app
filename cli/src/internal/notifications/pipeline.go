// Package notifications provides the event pipeline for routing state changes to notifications
package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/notify"
)

// EventType represents different notification event types
type EventType string

const (
	EventServiceStateChange EventType = "service_state_change"
	EventResourceUpdate     EventType = "resource_update"
	EventDeploymentComplete EventType = "deployment_complete"
	EventHealthCheck        EventType = "health_check"
	EventError              EventType = "error"
)

// Event represents a notification event
type Event struct {
	Type        EventType
	ServiceName string
	OldState    *monitor.ServiceState
	NewState    *monitor.ServiceState
	Message     string
	Severity    string // "critical", "warning", "info"
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// Handler processes notification events
type Handler interface {
	Handle(ctx context.Context, event Event) error
}

// Pipeline manages the notification event flow
type Pipeline struct {
	handlers []Handler
	events   chan Event
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewPipeline creates a new notification pipeline
func NewPipeline(bufferSize int) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		handlers: make([]Handler, 0),
		events:   make(chan Event, bufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler adds a handler to the pipeline
func (p *Pipeline) RegisterHandler(handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = append(p.handlers, handler)
}

// Start begins processing events
func (p *Pipeline) Start() {
	p.wg.Add(1)
	go p.processEvents()
}

// Stop gracefully shuts down the pipeline
func (p *Pipeline) Stop() error {
	p.cancel()
	close(p.events)
	p.wg.Wait()
	return nil
}

// Publish sends an event to the pipeline
func (p *Pipeline) Publish(event Event) error {
	select {
	case p.events <- event:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("pipeline stopped")
	default:
		return fmt.Errorf("event buffer full")
	}
}

// processEvents handles event distribution to handlers
func (p *Pipeline) processEvents() {
	defer p.wg.Done()

	for {
		select {
		case event, ok := <-p.events:
			if !ok {
				return
			}
			p.handleEvent(event)
		case <-p.ctx.Done():
			return
		}
	}
}

// handleEvent distributes an event to all handlers
func (p *Pipeline) handleEvent(event Event) {
	p.mu.RLock()
	handlers := make([]Handler, len(p.handlers))
	copy(handlers, p.handlers)
	p.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.Handle(p.ctx, event); err != nil {
			// Log error but continue processing
			fmt.Printf("Handler error: %v\n", err)
		}
	}
}

// OSNotificationHandler sends events to OS notification system
type OSNotificationHandler struct {
	notifier notify.Notifier
	config   *config.NotificationPreferences
	lastSent map[string]time.Time
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewOSNotificationHandler creates a handler for OS notifications
func NewOSNotificationHandler(notifier notify.Notifier, cfg *config.NotificationPreferences) *OSNotificationHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &OSNotificationHandler{
		notifier: notifier,
		config:   cfg,
		lastSent: make(map[string]time.Time),
		ctx:      ctx,
		cancel:   cancel,
	}
	// Start cleanup goroutine to prevent memory leak
	go h.cleanupOldEntries()
	return h
}

// cleanupOldEntries periodically removes stale rate limit entries
func (h *OSNotificationHandler) cleanupOldEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.Lock()
			cutoff := time.Now().Add(-h.config.GetRateLimitDuration() * 2)
			for key, lastSent := range h.lastSent {
				if lastSent.Before(cutoff) {
					delete(h.lastSent, key)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Close stops the cleanup goroutine
func (h *OSNotificationHandler) Close() error {
	h.cancel()
	return nil
}

// Handle processes an event and sends OS notification if appropriate
func (h *OSNotificationHandler) Handle(ctx context.Context, event Event) error {
	severityStr := string(event.Severity)
	if !h.config.ShouldNotify(event.ServiceName, severityStr) {
		return nil
	}

	// Rate limiting
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

	// Send notification
	notification := notify.Notification{
		Title:     fmt.Sprintf("Azure Dev: %s", event.ServiceName),
		Message:   event.Message,
		Severity:  event.Severity,
		Timestamp: event.Timestamp,
	}

	return h.notifier.Send(ctx, notification)
}

// WebSocketHandler sends events to connected dashboard clients
type WebSocketHandler struct {
	broadcaster func(event Event)
}

// NewWebSocketHandler creates a handler for WebSocket broadcasting
func NewWebSocketHandler(broadcaster func(event Event)) *WebSocketHandler {
	return &WebSocketHandler{broadcaster: broadcaster}
}

// Handle broadcasts event to WebSocket clients
func (h *WebSocketHandler) Handle(ctx context.Context, event Event) error {
	h.broadcaster(event)
	return nil
}

// HistoryHandler stores events in database
type HistoryHandler struct {
	store HistoryStore
}

// HistoryStore interface for notification persistence
type HistoryStore interface {
	Save(ctx context.Context, event Event) error
}

// NewHistoryHandler creates a handler for event persistence
func NewHistoryHandler(store HistoryStore) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// Handle saves event to database
func (h *HistoryHandler) Handle(ctx context.Context, event Event) error {
	return h.store.Save(ctx, event)
}
