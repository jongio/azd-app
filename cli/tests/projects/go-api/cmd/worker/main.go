package main

import (
	"context"
	"log"
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
	log.Println("Worker service starting...")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker loop
	go runWorker(ctx)

	log.Println("Worker is running. Processing jobs every 5 seconds...")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	cancel()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	log.Println("Worker stopped")
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
	log.Printf("Processing job %d...", jobID)

	// Simulate work with context cancellation support
	select {
	case <-ctx.Done():
		log.Printf("Job %d cancelled", jobID)
		return
	case <-time.After(jobProcessing):
		log.Printf("Job %d completed", jobID)
	}
}
