package azure

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetOrCreateLogAnalyticsClient_Success(t *testing.T) {
	// Clean up before test
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}
	workspaceID := "test-workspace-123"

	// First call should create a new client
	client1, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		t.Fatalf("GetOrCreateLogAnalyticsClient failed: %v", err)
	}
	if client1 == nil {
		t.Fatal("Expected non-nil client")
	}

	// Verify client is cached
	if GetCachedClientCount() != 1 {
		t.Errorf("Expected 1 cached client, got %d", GetCachedClientCount())
	}

	// Second call should return the same client (from cache)
	client2, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		t.Fatalf("GetOrCreateLogAnalyticsClient (cached) failed: %v", err)
	}
	if client2 != client1 {
		t.Error("Expected same client instance from cache")
	}

	// Verify cache count didn't increase
	if GetCachedClientCount() != 1 {
		t.Errorf("Expected 1 cached client after reuse, got %d", GetCachedClientCount())
	}
}

func TestGetOrCreateLogAnalyticsClient_EmptyWorkspaceID(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	_, err := GetOrCreateLogAnalyticsClient(ctx, cred, "")
	if err == nil {
		t.Fatal("Expected error for empty workspace ID")
	}

	expectedMsg := "workspace ID cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Verify no client was cached
	if GetCachedClientCount() != 0 {
		t.Errorf("Expected 0 cached clients, got %d", GetCachedClientCount())
	}
}

func TestGetOrCreateLogAnalyticsClient_MultipleDifferentWorkspaces(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	workspaces := []string{"workspace-1", "workspace-2", "workspace-3"}
	clients := make([]*LogAnalyticsClient, len(workspaces))

	// Create clients for different workspaces
	for i, ws := range workspaces {
		client, err := GetOrCreateLogAnalyticsClient(ctx, cred, ws)
		if err != nil {
			t.Fatalf("GetOrCreateLogAnalyticsClient failed for workspace %s: %v", ws, err)
		}
		clients[i] = client
	}

	// Verify all are cached
	if GetCachedClientCount() != 3 {
		t.Errorf("Expected 3 cached clients, got %d", GetCachedClientCount())
	}

	// Verify clients are different instances
	for i := 0; i < len(clients); i++ {
		for j := i + 1; j < len(clients); j++ {
			if clients[i] == clients[j] {
				t.Errorf("Expected different client instances for workspaces %s and %s",
					workspaces[i], workspaces[j])
			}
		}
	}

	// Verify reusing a workspace returns the same client
	client1Again, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaces[0])
	if err != nil {
		t.Fatalf("GetOrCreateLogAnalyticsClient (reuse) failed: %v", err)
	}
	if client1Again != clients[0] {
		t.Error("Expected same client instance when reusing workspace ID")
	}

	// Cache count should not have increased
	if GetCachedClientCount() != 3 {
		t.Errorf("Expected 3 cached clients after reuse, got %d", GetCachedClientCount())
	}
}

func TestClearClientCache(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	// Create multiple clients
	workspaces := []string{"ws-1", "ws-2", "ws-3"}
	for _, ws := range workspaces {
		_, err := GetOrCreateLogAnalyticsClient(ctx, cred, ws)
		if err != nil {
			t.Fatalf("Failed to create client for %s: %v", ws, err)
		}
	}

	// Verify they're cached
	if GetCachedClientCount() != 3 {
		t.Errorf("Expected 3 cached clients before clear, got %d", GetCachedClientCount())
	}

	// Clear the cache
	ClearClientCache()

	// Verify cache is empty
	if GetCachedClientCount() != 0 {
		t.Errorf("Expected 0 cached clients after clear, got %d", GetCachedClientCount())
	}

	// Verify creating a client after clear works
	client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaces[0])
	if err != nil {
		t.Fatalf("Failed to create client after cache clear: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client after cache clear")
	}

	// Verify it's cached again
	if GetCachedClientCount() != 1 {
		t.Errorf("Expected 1 cached client after recreate, got %d", GetCachedClientCount())
	}
}

func TestGetCachedClientCount(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	// Initially should be 0
	if GetCachedClientCount() != 0 {
		t.Errorf("Expected 0 cached clients initially, got %d", GetCachedClientCount())
	}

	ctx := context.Background()
	cred := &mockCredential{}

	// Add clients incrementally and verify count
	for i := 1; i <= 5; i++ {
		workspaceID := "workspace-" + string(rune('0'+i))
		_, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}

		if GetCachedClientCount() != i {
			t.Errorf("Expected %d cached clients, got %d", i, GetCachedClientCount())
		}
	}
}

func TestGetOrCreateLogAnalyticsClient_ThreadSafety(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}
	workspaceID := "concurrent-test-workspace"

	// Number of concurrent goroutines
	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect results
	results := make(chan *LogAnalyticsClient, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines that try to get/create the same client
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
			if err != nil {
				errors <- err
				return
			}
			results <- client
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent creation failed: %v", err)
	}

	// Collect all clients
	clients := make([]*LogAnalyticsClient, 0, numGoroutines)
	for client := range results {
		clients = append(clients, client)
	}

	// Verify all goroutines got a client
	if len(clients) != numGoroutines {
		t.Errorf("Expected %d clients, got %d", numGoroutines, len(clients))
	}

	// Verify all goroutines got the SAME client (double-checked locking worked)
	firstClient := clients[0]
	for i, client := range clients {
		if client != firstClient {
			t.Errorf("Client %d is different from first client (double-checked locking failed)", i)
		}
	}

	// Verify only ONE client was cached
	if GetCachedClientCount() != 1 {
		t.Errorf("Expected exactly 1 cached client, got %d", GetCachedClientCount())
	}
}

func TestGetOrCreateLogAnalyticsClient_ConcurrentDifferentWorkspaces(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	// Create clients for different workspaces concurrently
	numWorkspaces := 10
	var wg sync.WaitGroup
	wg.Add(numWorkspaces)

	errors := make(chan error, numWorkspaces)

	for i := 0; i < numWorkspaces; i++ {
		workspaceID := "workspace-" + string(rune('A'+i))
		go func(wsID string) {
			defer wg.Done()
			_, err := GetOrCreateLogAnalyticsClient(ctx, cred, wsID)
			if err != nil {
				errors <- err
			}
		}(workspaceID)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent creation for different workspaces failed: %v", err)
	}

	// Verify all workspaces were cached
	if GetCachedClientCount() != numWorkspaces {
		t.Errorf("Expected %d cached clients, got %d", numWorkspaces, GetCachedClientCount())
	}
}

func TestClearClientCache_ThreadSafety(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	// Pre-populate cache
	for i := 0; i < 5; i++ {
		workspaceID := "workspace-" + string(rune('0'+i))
		_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	}

	// Concurrently clear cache and create new clients
	var wg sync.WaitGroup
	numOperations := 50
	wg.Add(numOperations)

	for i := 0; i < numOperations; i++ {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				// Clear cache
				ClearClientCache()
			} else {
				// Try to get/create client
				workspaceID := "workspace-" + string(rune('A'+(idx%10)))
				_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
			}
		}(i)
	}

	wg.Wait()

	// Test should complete without panics or data races
	// Final state is non-deterministic but should be valid
	count := GetCachedClientCount()
	if count < 0 {
		t.Errorf("Invalid cache count: %d", count)
	}
}

func TestGetCachedClientCount_ThreadSafety(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	// Concurrently add clients and read count
	var wg sync.WaitGroup
	numReaders := 50
	numWriters := 10
	wg.Add(numReaders + numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = GetCachedClientCount()
			}
		}()
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			workspaceID := "workspace-writer-" + string(rune('0'+idx))
			_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
		}(i)
	}

	wg.Wait()

	// Should have numWriters clients cached
	if GetCachedClientCount() != numWriters {
		t.Errorf("Expected %d cached clients, got %d", numWriters, GetCachedClientCount())
	}
}

func TestClientPool_ContextCancellation(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cred := &mockCredential{}
	workspaceID := "test-workspace"

	// Should still succeed - context is not used in GetOrCreateLogAnalyticsClient
	// (context is used later in actual API calls)
	client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		t.Fatalf("GetOrCreateLogAnalyticsClient failed with cancelled context: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client even with cancelled context")
	}
}

func TestClientPool_NilCredential(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	workspaceID := "test-workspace"

	// Creating a client with nil credential will succeed in the pool function,
	// but will fail when NewLogAnalyticsClient is called
	client, err := GetOrCreateLogAnalyticsClient(ctx, nil, workspaceID)

	// The behavior depends on implementation - either it errors or creates a client
	// Let's just verify it doesn't panic
	if err != nil {
		// Error is acceptable
		t.Logf("GetOrCreateLogAnalyticsClient with nil credential returned error: %v", err)
		// Verify nothing was cached on error
		if GetCachedClientCount() != 0 {
			t.Errorf("Expected 0 cached clients after error, got %d", GetCachedClientCount())
		}
	} else if client != nil {
		// Client creation succeeded - this is also acceptable behavior
		t.Logf("GetOrCreateLogAnalyticsClient with nil credential succeeded (client may fail on actual operations)")
	}
}

func TestClientPool_DoubleCheckedLocking(t *testing.T) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}
	workspaceID := "double-check-workspace"

	// This test verifies the double-checked locking pattern:
	// 1. First goroutine gets read lock, doesn't find client, releases read lock
	// 2. While first goroutine waits for write lock, second goroutine should create the client
	// 3. First goroutine gets write lock, checks again, finds client (created by second goroutine)

	var wg sync.WaitGroup
	wg.Add(2)

	results := make(chan *LogAnalyticsClient, 2)

	// Goroutine 1
	go func() {
		defer wg.Done()
		client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
		if err != nil {
			t.Errorf("Goroutine 1 failed: %v", err)
			return
		}
		results <- client
	}()

	// Goroutine 2 (slight delay to increase chance of race condition)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond)
		client, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
		if err != nil {
			t.Errorf("Goroutine 2 failed: %v", err)
			return
		}
		results <- client
	}()

	wg.Wait()
	close(results)

	// Collect results
	clients := make([]*LogAnalyticsClient, 0, 2)
	for client := range results {
		clients = append(clients, client)
	}

	// Both should have gotten the same client
	if len(clients) != 2 {
		t.Fatalf("Expected 2 clients, got %d", len(clients))
	}
	if clients[0] != clients[1] {
		t.Error("Double-checked locking failed: different clients returned")
	}

	// Only one client should be cached
	if GetCachedClientCount() != 1 {
		t.Errorf("Expected 1 cached client, got %d", GetCachedClientCount())
	}
}

// Benchmark tests
func BenchmarkGetOrCreateLogAnalyticsClient_Cached(b *testing.B) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}
	workspaceID := "benchmark-workspace"

	// Pre-create the client
	_, err := GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	}
}

func BenchmarkGetOrCreateLogAnalyticsClient_Concurrent(b *testing.B) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}
	workspaceID := "benchmark-workspace"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
		}
	})
}

func BenchmarkGetCachedClientCount(b *testing.B) {
	ClearClientCache()
	defer ClearClientCache()

	ctx := context.Background()
	cred := &mockCredential{}

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		workspaceID := "workspace-" + string(rune('0'+i))
		_, _ = GetOrCreateLogAnalyticsClient(ctx, cred, workspaceID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetCachedClientCount()
	}
}
