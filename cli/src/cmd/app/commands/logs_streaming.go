package commands

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

func (e *logsExecutor) followLogs(ctx context.Context, projectDir string, logManager LogManagerInterface, dashboardClient DashboardClient, serviceFilter []string, levelFilter service.LogLevel, logFilter *service.LogFilter, outputWriter io.Writer) error {
	minLevel, minLevelSet := parseMinLevel(e.opts.minLevel)

	// Handle source-specific follow modes
	switch e.opts.source {
	case string(LogSourceAzure):
		return e.followAzureLogs(ctx, dashboardClient, projectDir, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	case string(LogSourceAll):
		return e.followAllLogs(ctx, projectDir, logManager, dashboardClient, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	default: // "local"
		return e.followLocalLogs(ctx, projectDir, logManager, dashboardClient, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	}
}

// followLocalLogs subscribes to local log streams and displays them.
func (e *logsExecutor) followLocalLogs(ctx context.Context, _ string, logManager LogManagerInterface, dashboardClient DashboardClient, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Try in-memory subscriptions first
	subscriptions := make(map[string]chan service.LogEntry)

	if len(serviceFilter) == 0 {
		// Subscribe to all services
		for serviceName, buffer := range logManager.GetAllBuffers() {
			subscriptions[serviceName] = buffer.Subscribe()
		}
	} else {
		// Subscribe to specific services
		for _, serviceName := range serviceFilter {
			buffer, exists := logManager.GetBuffer(serviceName)
			if exists {
				subscriptions[serviceName] = buffer.Subscribe()
			}
		}
	}

	// If no in-memory buffers, try dashboard WebSocket streaming
	if len(subscriptions) == 0 {
		return e.followLogsViaDashboard(ctx, dashboardClient, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	}

	// Use in-memory streaming
	return e.followLogsInMemory(subscriptions, logManager, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
}

// followLogsViaDashboard connects to the dashboard's WebSocket to stream logs.
func (e *logsExecutor) followLogsViaDashboard(ctx context.Context, dashboardClient DashboardClient, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	if dashboardClient == nil {
		return fmt.Errorf("cannot follow logs: dashboard not available (run 'azd app run' first)")
	}

	// Check if dashboard is responding
	if err := dashboardClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot follow logs: dashboard not responding (run 'azd app run' first)")
	}

	cliout.Info("Streaming logs from dashboard...")

	// Create context for streaming that can be canceled
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful exit
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

	// Create channel for log entries
	logs := make(chan service.LogEntry, logChannelBufferSize)

	// Determine service filter (empty string for all)
	serviceName := ""
	if len(serviceFilter) == 1 {
		serviceName = serviceFilter[0]
	}

	// Start streaming in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- dashboardClient.StreamLogs(streamCtx, serviceName, logs)
	}()

	// Display logs as they arrive
	for {
		select {
		case entry := <-logs:
			// Filter by service if multiple specified
			if len(serviceFilter) > 1 {
				found := false
				for _, svc := range serviceFilter {
					if entry.Service == svc {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, minLevel, minLevelSet, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.output == outputFormatJSON {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
			}

		case err := <-errChan:
			if err != nil && err != context.Canceled {
				return fmt.Errorf("log stream error: %w", err)
			}
			return nil

		case <-sigChan:
			cancel()
			return nil
		}
	}
}

// followLogsInMemory uses in-memory log buffer subscriptions.
func (e *logsExecutor) followLogsInMemory(subscriptions map[string]chan service.LogEntry, logManager LogManagerInterface, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Setup signal handling for graceful exit
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

	// Create stop channel for goroutine cleanup
	stopChan := make(chan struct{})

	// Merge all subscription channels with WaitGroup to track completion
	mergedChan := make(chan service.LogEntry, logChannelBufferSize)
	var wg sync.WaitGroup

	for _, ch := range subscriptions {
		wg.Add(1)
		go func(ch chan service.LogEntry) {
			defer wg.Done()
			for {
				select {
				case entry, ok := <-ch:
					if !ok {
						return
					}
					select {
					case mergedChan <- entry:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}(ch)
	}

	// Close mergedChan when all goroutines complete
	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	// Cleanup helper function with sync.Once to prevent double-close panics
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(stopChan)
			wg.Wait() // Ensure all goroutines stopped before unsubscribing
			for serviceName, ch := range subscriptions {
				buffer, exists := logManager.GetBuffer(serviceName)
				if exists {
					buffer.Unsubscribe(ch)
				}
			}
		})
	}

	// Display logs as they arrive
	for {
		select {
		case entry, ok := <-mergedChan:
			if !ok {
				// All sources closed
				cleanup()
				return nil
			}

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, minLevel, minLevelSet, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.output == outputFormatJSON {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
				e.emitAlerts(entry.Service, entry.Message, outputWriter)
			}

		case <-sigChan:
			cleanup()
			return nil
		}
	}
}

// followAzureLogs streams Azure logs via the dashboard's WebSocket.
func (e *logsExecutor) followAzureLogs(ctx context.Context, dashboardClient DashboardClient, projectDir string, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// If dashboard is available, prefer its streaming API
	if dashboardClient != nil {
		if err := dashboardClient.Ping(ctx); err == nil {
			status, err := dashboardClient.GetAzureStatus(ctx)
			if err == nil && status.Enabled {
				return e.followAzureLogsViaDashboard(ctx, dashboardClient, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
			}
		}
	}

	// Fallback: standalone polling (no dashboard needed)
	return e.followAzureLogsStandalone(ctx, projectDir, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
}

func (e *logsExecutor) followAzureLogsViaDashboard(ctx context.Context, dashboardClient DashboardClient, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	// Check if dashboard is responding
	if err := dashboardClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot follow Azure logs: dashboard not responding (run 'azd app run' first)")
	}

	// Check Azure status
	status, err := dashboardClient.GetAzureStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Azure status: %w", err)
	}

	if !status.Enabled {
		return fmt.Errorf("azure logging not configured.\n\nTo enable Azure logs:\n  1. Add a Log Analytics configuration to azure.yaml:\n     logs:\n       analytics:\n         pollingInterval: \"30s\"\n         defaultTimespan: \"1h\"\n  2. Ensure your azd environment has workspace outputs (run 'azd provision' if needed)\n  3. Restart 'azd app run'\n\nFor more info: https://aka.ms/azd-app/azure-logs")
	}

	cliout.Info("Streaming Azure logs (polling every 30s)...")

	// Create context for streaming that can be canceled
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful exit
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

	// Create channel for log entries
	logs := make(chan service.LogEntry, logChannelBufferSize)

	// Start streaming in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- dashboardClient.StreamAzureLogs(streamCtx, logs)
	}()

	// Build service filter set if multiple services specified
	var serviceSet map[string]bool
	if len(serviceFilter) > 1 {
		serviceSet = make(map[string]bool)
		for _, svc := range serviceFilter {
			serviceSet[svc] = true
		}
	}

	// Display logs as they arrive
	for {
		select {
		case entry := <-logs:
			// Filter by service if multiple specified
			if serviceSet != nil {
				if !serviceSet[entry.Service] {
					continue
				}
			} else if len(serviceFilter) == 1 && entry.Service != serviceFilter[0] {
				continue
			}

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, minLevel, minLevelSet, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.output == outputFormatJSON {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
			}

		case err := <-errChan:
			if err != nil && err != context.Canceled {
				return fmt.Errorf("azure log stream error: %w", err)
			}
			return nil

		case <-sigChan:
			cancel()
			return nil
		}
	}
}

// followAzureLogsStandalone streams Azure logs without requiring the dashboard.
func (e *logsExecutor) followAzureLogsStandalone(ctx context.Context, projectDir string, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	cliout.Info("Streaming Azure logs (standalone, polling)...")

	// Create context for streaming that can be canceled
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful exit
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

	logs := make(chan azure.LogEntry, logChannelBufferSize)
	errChan := make(chan error, 1)

	initialWindow := time.Hour
	if e.opts.since != "" {
		if sinceDuration, err := time.ParseDuration(e.opts.since); err == nil {
			if sinceDuration < time.Minute {
				sinceDuration = time.Minute
			}
			initialWindow = sinceDuration
		}
	}

	pollInterval := 30 * time.Second

	go func() {
		errChan <- streamAzureLogsStandalone(streamCtx, azure.StreamConfig{
			ProjectDir:    projectDir,
			Services:      serviceFilter,
			PollInterval:  pollInterval,
			InitialWindow: initialWindow,
		}, logs)
	}()

	var serviceSet map[string]bool
	if len(serviceFilter) > 1 {
		serviceSet = make(map[string]bool)
		for _, svc := range serviceFilter {
			serviceSet[svc] = true
		}
	}

	for {
		select {
		case azEntry := <-logs:
			entry := service.LogEntry{
				Service:   azEntry.Service,
				Message:   azEntry.Message,
				Level:     convertAzureLogLevel(azEntry.Level),
				Timestamp: azEntry.Timestamp,
				Source:    service.LogSourceAzure,
				AzureMetadata: &service.AzureLogMetadata{
					ResourceType:  azEntry.ResourceType,
					ContainerName: azEntry.ContainerName,
					InstanceID:    azEntry.InstanceID,
				},
			}

			if serviceSet != nil && !serviceSet[entry.Service] {
				continue
			}
			if len(serviceFilter) == 1 && entry.Service != serviceFilter[0] {
				continue
			}
			if !e.shouldDisplayEntry(entry, levelFilter, minLevel, minLevelSet, logFilter) {
				continue
			}

			if e.opts.output == outputFormatJSON {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
			}

		case err := <-errChan:
			if err != nil && err != context.Canceled {
				return fmt.Errorf("azure log stream error: %w", err)
			}
			return nil

		case <-sigChan:
			cancel()
			return nil
		}
	}
}

// followAllLogs streams logs from both local and Azure sources.
func (e *logsExecutor) followAllLogs(ctx context.Context, projectDir string, _ LogManagerInterface, dashboardClient DashboardClient, serviceFilter []string, levelFilter, minLevel service.LogLevel, minLevelSet bool, logFilter *service.LogFilter, outputWriter io.Writer) error {
	if dashboardClient == nil {
		cliout.Warning("Dashboard not running; following Azure logs only.")
		return e.followAzureLogsStandalone(ctx, projectDir, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	}

	// Check if dashboard is responding
	if err := dashboardClient.Ping(ctx); err != nil {
		cliout.Warning("Dashboard not responding; following Azure logs only.")
		return e.followAzureLogsStandalone(ctx, projectDir, serviceFilter, levelFilter, minLevel, minLevelSet, logFilter, outputWriter)
	}

	cliout.Info("Streaming logs from local and Azure sources...")

	// Create context for streaming that can be canceled
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful exit
	sigChan, cleanupSignal := e.getOrCreateSignalChan()
	defer cleanupSignal()

	// Create merged channel for all log entries
	mergedLogs := make(chan service.LogEntry, logChannelBufferSize*2)

	// Start local log streaming
	localLogs := make(chan service.LogEntry, logChannelBufferSize)
	serviceName := ""
	if len(serviceFilter) == 1 {
		serviceName = serviceFilter[0]
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Local logs goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(localLogs)
		if err := dashboardClient.StreamLogs(streamCtx, serviceName, localLogs); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("local log stream error: %w", err)
		}
	}()

	// Azure logs goroutine (only if Azure is enabled)
	azureLogs := make(chan service.LogEntry, logChannelBufferSize)
	status, _ := dashboardClient.GetAzureStatus(ctx)
	if status != nil && status.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(azureLogs)
			if err := dashboardClient.StreamAzureLogs(streamCtx, azureLogs); err != nil && err != context.Canceled {
				// Azure errors are non-fatal, just log and continue
				cliout.Warning("Azure log stream disconnected: %s", err)
			}
		}()
	} else {
		close(azureLogs)
	}

	// Merger goroutine - combines local and azure logs
	go func() {
		for {
			select {
			case entry, ok := <-localLogs:
				if ok {
					mergedLogs <- entry
				}
			case entry, ok := <-azureLogs:
				if ok {
					mergedLogs <- entry
				}
			case <-streamCtx.Done():
				return
			}
		}
	}()

	// Build service filter set if multiple services specified
	var serviceSet map[string]bool
	if len(serviceFilter) > 1 {
		serviceSet = make(map[string]bool)
		for _, svc := range serviceFilter {
			serviceSet[svc] = true
		}
	}

	// Display logs as they arrive
	for {
		select {
		case entry := <-mergedLogs:
			// Filter by service if multiple specified
			if serviceSet != nil && !serviceSet[entry.Service] {
				continue
			}

			// Use extracted filter method
			if !e.shouldDisplayEntry(entry, levelFilter, minLevel, minLevelSet, logFilter) {
				continue
			}

			// Display log entry
			if e.opts.output == outputFormatJSON {
				displayLogsJSON([]service.LogEntry{entry}, outputWriter)
			} else {
				displayLogsText([]service.LogEntry{entry}, outputWriter, e.opts.timestamps, e.opts.noColor)
			}

		case err := <-errChan:
			if err != nil {
				return err
			}

		case <-sigChan:
			cancel()
			return nil
		}
	}
}

// readLogsFromFile reads logs from the persisted log file for a service.
// This is used when the in-memory buffer is empty (e.g., when called from a subprocess).
// It also reads from rotated backup files (.log.1, .log.2) if needed.
