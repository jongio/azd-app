package containerauth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectContainerArch(t *testing.T) {
	arch := DetectContainerArch()
	if arch != "amd64" && arch != "arm64" {
		t.Errorf("DetectContainerArch() = %q, want amd64 or arm64", arch)
	}
	// On this platform, verify the correct value
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if arch != "arm64" {
			t.Errorf("expected arm64 on darwin/arm64, got %q", arch)
		}
	} else {
		if arch != "amd64" {
			t.Errorf("expected amd64 on %s/%s, got %q", runtime.GOOS, runtime.GOARCH, arch)
		}
	}
}

func TestDetectContainerHost(t *testing.T) {
	host := DetectContainerHost()
	if host == "" {
		t.Error("DetectContainerHost() returned empty string")
	}
	// Should be one of the known hostnames
	validHosts := []string{"host.docker.internal", "host.containers.internal"}
	found := false
	for _, h := range validHosts {
		if host == h {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DetectContainerHost() = %q, not a recognized host", host)
	}
}

func TestNeedsExtraHosts(t *testing.T) {
	result := NeedsExtraHosts()
	if runtime.GOOS == "linux" && !result {
		t.Error("NeedsExtraHosts() should be true on Linux")
	}
	if runtime.GOOS != "linux" && result {
		t.Error("NeedsExtraHosts() should be false on non-Linux")
	}
}

func TestExtraHostsEntries(t *testing.T) {
	entries := ExtraHostsEntries()
	if runtime.GOOS == "linux" {
		if len(entries) == 0 {
			t.Error("ExtraHostsEntries() should return entries on Linux")
		}
		if entries[0] != "host.docker.internal:host-gateway" {
			t.Errorf("ExtraHostsEntries()[0] = %q, want host.docker.internal:host-gateway", entries[0])
		}
	} else {
		if entries != nil {
			t.Errorf("ExtraHostsEntries() should be nil on non-Linux, got %v", entries)
		}
	}
}

func TestFindShimSource(t *testing.T) {
	// This test verifies findShimSource can locate the shim from CWD
	// It should work when running from the repository root
	src, err := findShimSource()
	if err != nil {
		t.Skipf("shim source not found from current directory (expected in CI): %v", err)
	}
	// Verify the path contains the expected structure
	if !strings.HasSuffix(src, filepath.Join("containerauth", "shim")) {
		t.Errorf("findShimSource() = %q, expected to end with containerauth/shim", src)
	}
	// Verify main.go exists in the found directory
	mainPath := filepath.Join(src, "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Errorf("expected main.go at %s", mainPath)
	}
}
