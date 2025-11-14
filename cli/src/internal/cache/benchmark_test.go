package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkCacheManagerGetCachedResults benchmarks cache retrieval.
func BenchmarkCacheManagerGetCachedResults(b *testing.B) {
	tmpDir := b.TempDir()

	cm, err := NewCacheManagerWithOptions(CacheOptions{
		CacheDir: tmpDir,
		Enabled:  true,
		TTL:      1 * time.Hour,
	})
	if err != nil {
		b.Fatal(err)
	}

	// Create a dummy azure.yaml
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0644); err != nil {
		b.Fatal(err)
	}

	// Save some cached results
	results := []CachedReqResult{
		{Name: "node", Installed: true, Version: "18.0.0", Required: ">=18", Satisfied: true},
		{Name: "python", Installed: true, Version: "3.11.0", Required: ">=3.10", Satisfied: true},
	}
	if err := cm.SaveResults(azureYamlPath, results, true); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := cm.GetCachedResults(azureYamlPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCacheManagerSaveResults benchmarks cache saving.
func BenchmarkCacheManagerSaveResults(b *testing.B) {
	tmpDir := b.TempDir()

	cm, err := NewCacheManagerWithOptions(CacheOptions{
		CacheDir: tmpDir,
		Enabled:  true,
		TTL:      1 * time.Hour,
	})
	if err != nil {
		b.Fatal(err)
	}

	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0644); err != nil {
		b.Fatal(err)
	}

	results := []CachedReqResult{
		{Name: "node", Installed: true, Version: "18.0.0", Required: ">=18", Satisfied: true},
		{Name: "python", Installed: true, Version: "3.11.0", Required: ">=3.10", Satisfied: true},
		{Name: "dotnet", Installed: true, Version: "8.0.0", Required: ">=8", Satisfied: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cm.SaveResults(azureYamlPath, results, true); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculateFileHash benchmarks file hashing.
func BenchmarkCalculateFileHash(b *testing.B) {
	tmpDir := b.TempDir()

	testFile := filepath.Join(tmpDir, "test.yaml")
	content := make([]byte, 10*1024) // 10KB file
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := calculateFileHash(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}
