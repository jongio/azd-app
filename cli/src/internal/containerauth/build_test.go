package containerauth

import (
	"os"
	"runtime"
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

func TestExtractShim(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			path, err := ExtractShim(arch)
			if err != nil {
				t.Fatalf("ExtractShim(%q) error: %v", arch, err)
			}
			defer func() { _ = os.RemoveAll(path) }()

			// Verify the file exists and is executable
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("extracted shim not found: %v", err)
			}
			if info.Size() == 0 {
				t.Error("extracted shim is empty")
			}
			// On Unix, check executable bit
			if runtime.GOOS != "windows" {
				if info.Mode()&0o111 == 0 {
					t.Error("extracted shim is not executable")
				}
			}
		})
	}
}

func TestExtractShimInvalidArch(t *testing.T) {
	_, err := ExtractShim("mips")
	if err == nil {
		t.Error("expected error for unsupported architecture, got nil")
	}
}
