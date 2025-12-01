package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	jobInterval   = 5 * time.Second
	jobProcessing = 100 * time.Millisecond
)

func main() {
	slog.Info("Worker service starting...")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker loop
	go runWorker(ctx)

	slog.Info("Worker is running. Processing jobs every 5 seconds...")

	// Wait for shutdown signal
	sig := <-sigChan
	slog.Info("Received signal, shutting down...", "signal", sig)
	cancel()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	slog.Info("Worker stopped")
}

func runWorker(ctx context.Context) {
	ticker := time.NewTicker(jobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processJob(ctx)
		}
	}
}

func processJob(ctx context.Context) {
	jobID := time.Now().UnixNano()
	slog.Info("Processing job...", "jobID", jobID)

	// Simulate work with context cancellation support
	select {
	case <-ctx.Done():
		slog.Info("Job cancelled", "jobID", jobID)
		return
	case <-time.After(jobProcessing):
		slog.Info("Job completed", "jobID", jobID)
	}
}
