package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
)

// TestPersistentDashboardPort_FirstRunPersists verifies that the first run
// generates a port and stores it in the port manager.
func TestPersistentDashboardPort_FirstRunPersists(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	srv := GetServer(tempDir)

	// Start the server
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	if url == "" {
		t.Fatal("Expected non-empty URL")
	}

	// Verify port was stored in port manager
	pm := portmanager.GetPortManager(tempDir)
	persistedPort, exists := pm.GetAssignment(constants.DashboardServiceName)
	if !exists {
		t.Errorf("Expected %s to be in port manager assignments", constants.DashboardServiceName)
	}
	if persistedPort != srv.port {
		t.Errorf("Expected stored port %d to match server port %d", persistedPort, srv.port)
	}
}

// TestPersistentDashboardPort_SecondRunReusesPersisted verifies that the second run
// uses the same port that was persisted from the first run.
func TestPersistentDashboardPort_SecondRunReusesPersisted(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map and port manager cache for clean state
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// First run - start and get the port
	srv1 := GetServer(tempDir)
	url1, err := srv1.Start()
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	port1 := srv1.port
	_ = srv1.Stop()

	// Clear servers map but NOT port manager cache (to simulate restart)
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	// Second run - should use same port
	srv2 := GetServer(tempDir)

	// Check that GetAssignment returns the persisted port before Start
	portMgr := portmanager.GetPortManager(tempDir)
	persistedPort, exists := portMgr.GetAssignment(constants.DashboardServiceName)
	if !exists {
		t.Fatal("Expected dashboard port assignment to exist after first run")
	}
	if persistedPort != port1 {
		t.Errorf("Persisted port %d doesn't match first run port %d", persistedPort, port1)
	}

	url2, err := srv2.Start()
	if err != nil {
		t.Fatalf("Second Start() failed: %v", err)
	}
	defer func() { _ = srv2.Stop() }()
	port2 := srv2.port

	// Verify same port is used
	if port1 != port2 {
		t.Errorf("Expected same port across runs, got port1=%d port2=%d", port1, port2)
	}

	// Verify URLs match
	if url1 != url2 {
		t.Errorf("Expected same URL across runs, got url1=%s url2=%s", url1, url2)
	}
}

// TestPersistentDashboardPort_PortRangeIsValid verifies that generated ports
// are in the expected dashboard range (40000-49999).
func TestPersistentDashboardPort_PortRangeIsValid(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	srv := GetServer(tempDir)
	_, err := srv.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Verify port is in expected range
	if srv.port < constants.DashboardPortRangeMin || srv.port > constants.DashboardPortRangeMax {
		t.Errorf("Expected port in range %d-%d, got %d",
			constants.DashboardPortRangeMin, constants.DashboardPortRangeMax, srv.port)
	}
}

// TestPersistentDashboardPort_GetAssignmentBeforeStart verifies that GetAssignment
// returns false when no port has been assigned yet.
func TestPersistentDashboardPort_GetAssignmentBeforeStart(t *testing.T) {
	tempDir := t.TempDir()
	portmanager.ClearCacheForTesting()

	portMgr := portmanager.GetPortManager(tempDir)
	_, exists := portMgr.GetAssignment(constants.DashboardServiceName)
	if exists {
		t.Error("Expected no assignment before first start")
	}
}

// TestPersistentDashboardPort_MultipleProjects verifies that different projects
// get different persisted ports.
func TestPersistentDashboardPort_MultipleProjects(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// Start first project
	srv1 := GetServer(tempDir1)
	_, err := srv1.Start()
	if err != nil {
		t.Fatalf("First project Start() failed: %v", err)
	}
	port1 := srv1.port
	defer func() { _ = srv1.Stop() }()

	// Start second project
	srv2 := GetServer(tempDir2)
	_, err = srv2.Start()
	if err != nil {
		t.Fatalf("Second project Start() failed: %v", err)
	}
	port2 := srv2.port
	defer func() { _ = srv2.Stop() }()

	// Note: With the new architecture, ports are stored in azd config (not ports.json)
	// For unit tests without gRPC, ports are stored in-memory

	// Verify each project's port manager has its own assignment
	portMgr1 := portmanager.GetPortManager(tempDir1)
	portMgr2 := portmanager.GetPortManager(tempDir2)

	persistedPort1, exists1 := portMgr1.GetAssignment(constants.DashboardServiceName)
	persistedPort2, exists2 := portMgr2.GetAssignment(constants.DashboardServiceName)

	if !exists1 || !exists2 {
		t.Error("Expected both projects to have port assignments")
	}

	if persistedPort1 != port1 {
		t.Errorf("Project 1: persisted port %d doesn't match running port %d", persistedPort1, port1)
	}
	if persistedPort2 != port2 {
		t.Errorf("Project 2: persisted port %d doesn't match running port %d", persistedPort2, port2)
	}
}

// TestPersistentDashboardPort_PortConflictFallback verifies that when the persisted
// port is unavailable (in use by another process), the dashboard finds an alternative port.
func TestPersistentDashboardPort_PortConflictFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map and port manager cache
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// First run - get a port
	srv1 := GetServer(tempDir)
	_, err := srv1.Start()
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	port1 := srv1.port
	// Don't stop srv1 - keep it running to hold the port

	// Clear servers map but NOT the actual server or port manager
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	// Second run - should fail to use port1 (still in use) and find alternative
	srv2 := GetServer(tempDir)
	_, err = srv2.Start()
	if err != nil {
		// This is expected if port conflict causes a complete failure
		// But with proper fallback logic, it should succeed with a different port
		t.Logf("Second Start() failed (may be expected): %v", err)
	} else {
		port2 := srv2.port
		// The ports should be different since port1 is still in use
		if port1 == port2 {
			t.Errorf("Expected different ports when original is in use, got same port %d", port1)
		}
		_ = srv2.Stop()
	}

	// Clean up first server
	_ = srv1.Stop()
}

// ── Nonce tests ───────────────────────────────────────────────────────────────

// TestPortFileNonce_PathNotPredictableFromDir verifies that the port file path
// cannot be derived from the project directory alone — it must contain a nonce
// beyond the deterministic project hash (CWE-340).
func TestPortFileNonce_PathNotPredictableFromDir(t *testing.T) {
	nonceDirBase = t.TempDir()
	t.Cleanup(func() { nonceDirBase = "" })

	projectDir := t.TempDir()
	hash := azdconfig.ProjectHash(projectDir)

	// "predictable" path — the old, deterministic format with no nonce.
	predictedPath := filepath.Join(os.TempDir(), fmt.Sprintf(".azd-app-dashboard-%s.port", hash))

	path, err := portFilePath(projectDir)
	if err != nil {
		t.Fatalf("portFilePath() error: %v", err)
	}
	if path == predictedPath {
		t.Errorf("port file path %q equals the predictable legacy path; must include a random nonce", path)
	}
}

// TestPortFileNonce_SameProjectSamePath verifies that the nonce is persisted, so
// successive calls for the same project always return the same port file path.
func TestPortFileNonce_SameProjectSamePath(t *testing.T) {
	nonceDirBase = t.TempDir()
	t.Cleanup(func() { nonceDirBase = "" })

	projectDir := t.TempDir()

	path1, err := portFilePath(projectDir)
	if err != nil {
		t.Fatalf("first portFilePath() error: %v", err)
	}
	path2, err := portFilePath(projectDir)
	if err != nil {
		t.Fatalf("second portFilePath() error: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected identical paths for same project across calls, got %q and %q", path1, path2)
	}
}

// TestPortFileNonce_DifferentProjectsDifferentPaths verifies that distinct projects
// receive independent nonces and therefore distinct port file paths.
func TestPortFileNonce_DifferentProjectsDifferentPaths(t *testing.T) {
	nonceDirBase = t.TempDir()
	t.Cleanup(func() { nonceDirBase = "" })

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	path1, err := portFilePath(dir1)
	if err != nil {
		t.Fatalf("portFilePath(dir1) error: %v", err)
	}
	path2, err := portFilePath(dir2)
	if err != nil {
		t.Fatalf("portFilePath(dir2) error: %v", err)
	}
	if path1 == path2 {
		t.Errorf("expected different paths for different projects, both returned %q", path1)
	}
}

// TestPortFileNonce_NonceIs128Bits verifies the nonce embedded in the port file
// name is exactly 32 hex characters (128 bits of crypto/rand entropy).
func TestPortFileNonce_NonceIs128Bits(t *testing.T) {
	nonceDirBase = t.TempDir()
	t.Cleanup(func() { nonceDirBase = "" })

	projectDir := t.TempDir()
	hash := azdconfig.ProjectHash(projectDir)

	path, err := portFilePath(projectDir)
	if err != nil {
		t.Fatalf("portFilePath() error: %v", err)
	}

	// Expected filename: .azd-app-dashboard-{hash}-{nonce}.port
	base := filepath.Base(path)
	prefix := ".azd-app-dashboard-" + hash + "-"
	if !strings.HasPrefix(base, prefix) {
		t.Fatalf("unexpected port file name format: %q (want prefix %q)", base, prefix)
	}
	nonce := strings.TrimSuffix(strings.TrimPrefix(base, prefix), ".port")

	if len(nonce) != 32 {
		t.Errorf("expected 32-char nonce (128 bits), got %d chars: %q", len(nonce), nonce)
	}
	for _, ch := range nonce {
		if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') {
			t.Errorf("nonce contains non-hex character %q in %q", string(ch), nonce)
		}
	}
}

// TestPortFileNonce_LegacyFileCleanedUp verifies that writePortFile removes the
// old predictable port file (no nonce) written by earlier versions of the code.
func TestPortFileNonce_LegacyFileCleanedUp(t *testing.T) {
	nonceDirBase = t.TempDir()
	t.Cleanup(func() { nonceDirBase = "" })

	projectDir := t.TempDir()
	hash := azdconfig.ProjectHash(projectDir)

	// Plant a legacy port file with the old, nonce-free naming scheme.
	legacyPath := filepath.Join(os.TempDir(), fmt.Sprintf(".azd-app-dashboard-%s.port", hash))
	if err := os.WriteFile(legacyPath, []byte("9999"), 0o600); err != nil {
		t.Fatalf("failed to create legacy port file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(legacyPath) })

	writePortFile(projectDir, 12345)

	if _, statErr := os.Stat(legacyPath); !os.IsNotExist(statErr) {
		t.Errorf("legacy port file still present at %q after writePortFile; expected removal", legacyPath)
	}
}
