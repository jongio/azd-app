package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCacheManager(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test creating cache manager in new directory
	cm, err := NewCacheManager()
	if err != nil {
		t.Fatalf("NewCacheManager() failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewCacheManager() returned nil")
	}

	// Verify .azure/cache directory was created
	azureDir := filepath.Join(tempDir, ".azure")
	if _, err := os.Stat(azureDir); os.IsNotExist(err) {
		t.Errorf(".azure directory was not created")
	}

	cacheDir := filepath.Join(azureDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf(".azure/cache directory was not created")
	}
}

func TestNewCacheManagerExistingAzureDir(t *testing.T) {
	// Create a temporary directory with .azure
	tempDir := t.TempDir()
	azureDir := filepath.Join(tempDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("failed to create .azure directory: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test creating cache manager with existing .azure directory
	cm, err := NewCacheManager()
	if err != nil {
		t.Fatalf("NewCacheManager() failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewCacheManager() returned nil")
	}

	// Verify cache directory was created inside existing .azure
	cacheDir := filepath.Join(azureDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf(".azure/cache directory was not created")
	}
}

func TestFindAzureDir(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	azureDir := filepath.Join(tempDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("failed to create .azure directory: %v", err)
	}

	subDir := filepath.Join(tempDir, "sub1", "sub2")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name     string
		startDir string
		want     string
	}{
		{
			name:     "find in current directory",
			startDir: tempDir,
			want:     azureDir,
		},
		{
			name:     "find in parent directory",
			startDir: subDir,
			want:     azureDir,
		},
		{
			name:     "not found",
			startDir: os.TempDir(),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAzureDir(tt.startDir)
			if got != tt.want {
				t.Errorf("findAzureDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCachedResultsNoCache(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a test azure.yaml file
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("test: content"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Test with no cache file
	cache, valid, err := cm.GetCachedResults(azureYamlPath)
	if err != nil {
		t.Fatalf("GetCachedResults() error = %v", err)
	}

	if cache != nil {
		t.Errorf("GetCachedResults() cache = %v, want nil", cache)
	}

	if valid {
		t.Errorf("GetCachedResults() valid = %v, want false", valid)
	}
}

func TestGetCachedResultsValid(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a test azure.yaml file
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	content := []byte("test: content")
	if err := os.WriteFile(azureYamlPath, content, 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Calculate hash of the file
	hash, err := calculateFileHash(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to calculate hash: %v", err)
	}

	// Create a valid cache file
	results := []CachedReqResult{
		{
			Name:      "test-tool",
			Installed: true,
			Version:   "1.0.0",
			Required:  "1.0.0",
			Satisfied: true,
		},
	}

	cache := ReqsCache{
		Timestamp:     time.Now(),
		AzureYamlHash: hash,
		Results:       results,
		AllPassed:     true,
	}

	cacheData, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}

	cacheFile := filepath.Join(tempDir, "reqs_cache.json")
	if err := os.WriteFile(cacheFile, cacheData, 0600); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	// Test getting valid cache
	gotCache, valid, err := cm.GetCachedResults(azureYamlPath)
	if err != nil {
		t.Fatalf("GetCachedResults() error = %v", err)
	}

	if !valid {
		t.Errorf("GetCachedResults() valid = %v, want true", valid)
	}

	if gotCache == nil {
		t.Fatal("GetCachedResults() cache = nil, want non-nil")
	}

	if gotCache.AzureYamlHash != hash {
		t.Errorf("GetCachedResults() hash = %v, want %v", gotCache.AzureYamlHash, hash)
	}

	if !gotCache.AllPassed {
		t.Errorf("GetCachedResults() AllPassed = %v, want true", gotCache.AllPassed)
	}

	if len(gotCache.Results) != 1 {
		t.Errorf("GetCachedResults() results length = %v, want 1", len(gotCache.Results))
	}
}

func TestGetCachedResultsInvalidHash(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a test azure.yaml file
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("test: content"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Create a cache file with wrong hash
	cache := ReqsCache{
		Timestamp:     time.Now(),
		AzureYamlHash: "invalid-hash",
		Results:       []CachedReqResult{},
		AllPassed:     true,
	}

	cacheData, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}

	cacheFile := filepath.Join(tempDir, "reqs_cache.json")
	if err := os.WriteFile(cacheFile, cacheData, 0600); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	// Test getting cache with invalid hash
	gotCache, valid, err := cm.GetCachedResults(azureYamlPath)
	if err != nil {
		t.Fatalf("GetCachedResults() error = %v", err)
	}

	if valid {
		t.Errorf("GetCachedResults() valid = %v, want false", valid)
	}

	if gotCache != nil {
		t.Errorf("GetCachedResults() cache = %v, want nil", gotCache)
	}
}

func TestGetCachedResultsExpired(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a test azure.yaml file
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	content := []byte("test: content")
	if err := os.WriteFile(azureYamlPath, content, 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Calculate hash
	hash, err := calculateFileHash(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to calculate hash: %v", err)
	}

	// Create an old cache file
	cache := ReqsCache{
		Timestamp:     time.Now(),
		AzureYamlHash: hash,
		Results:       []CachedReqResult{},
		AllPassed:     true,
	}

	cacheData, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}

	cacheFile := filepath.Join(tempDir, "reqs_cache.json")
	if err := os.WriteFile(cacheFile, cacheData, 0600); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	// Set file modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cacheFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to change file times: %v", err)
	}

	// Test getting expired cache
	gotCache, valid, err := cm.GetCachedResults(azureYamlPath)
	if err != nil {
		t.Fatalf("GetCachedResults() error = %v", err)
	}

	if valid {
		t.Errorf("GetCachedResults() valid = %v, want false", valid)
	}

	if gotCache != nil {
		t.Errorf("GetCachedResults() cache = %v, want nil", gotCache)
	}
}

func TestSaveResults(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a test azure.yaml file
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("test: content"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Create test results
	results := []CachedReqResult{
		{
			Name:       "test-tool",
			Installed:  true,
			Version:    "1.0.0",
			Required:   "1.0.0",
			Satisfied:  true,
			Running:    true,
			CheckedRun: true,
		},
	}

	// Save results
	err := cm.SaveResults(azureYamlPath, results, true)
	if err != nil {
		t.Fatalf("SaveResults() error = %v", err)
	}

	// Verify cache file was created
	cacheFile := filepath.Join(tempDir, "reqs_cache.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Errorf("cache file was not created")
	}

	// Read and verify cache contents
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}

	var cache ReqsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("failed to unmarshal cache: %v", err)
	}

	if !cache.AllPassed {
		t.Errorf("cache.AllPassed = %v, want true", cache.AllPassed)
	}

	if len(cache.Results) != 1 {
		t.Fatalf("cache.Results length = %v, want 1", len(cache.Results))
	}

	if cache.Results[0].Name != "test-tool" {
		t.Errorf("cache.Results[0].Name = %v, want test-tool", cache.Results[0].Name)
	}

	if cache.AzureYamlHash == "" {
		t.Errorf("cache.AzureYamlHash is empty")
	}
}

func TestClearCache(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Create a cache file
	cacheFile := filepath.Join(tempDir, "reqs_cache.json")
	if err := os.WriteFile(cacheFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create cache file: %v", err)
	}

	// Clear cache
	err := cm.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	// Verify cache file was removed
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Errorf("cache file still exists after clearing")
	}
}

func TestClearCacheNoFile(t *testing.T) {
	tempDir := t.TempDir()
	cm := &CacheManager{cacheDir: tempDir}

	// Clear cache when no file exists (should not error)
	err := cm.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache() error = %v, want nil", err)
	}
}

func TestCalculateFileHash(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Calculate hash
	hash, err := calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("calculateFileHash() error = %v", err)
	}

	if hash == "" {
		t.Errorf("calculateFileHash() returned empty hash")
	}

	// Verify same content produces same hash
	hash2, err := calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("calculateFileHash() error = %v", err)
	}

	if hash != hash2 {
		t.Errorf("calculateFileHash() inconsistent: %v != %v", hash, hash2)
	}

	// Verify different content produces different hash
	testFile2 := filepath.Join(tempDir, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("different content"), 0600); err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}

	hash3, err := calculateFileHash(testFile2)
	if err != nil {
		t.Fatalf("calculateFileHash() error = %v", err)
	}

	if hash == hash3 {
		t.Errorf("calculateFileHash() same hash for different content")
	}
}

func TestCalculateFileHashNonexistent(t *testing.T) {
	_, err := calculateFileHash("/nonexistent/file.txt")
	if err == nil {
		t.Errorf("calculateFileHash() expected error for nonexistent file")
	}
}
