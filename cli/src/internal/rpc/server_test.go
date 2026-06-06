package rpc

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
)

// TestMountFailsWithEmptySessionToken verifies the fail-closed behaviour
// introduced by CWE-1188: Mount must return an error when SessionToken is
// empty and the caller has not explicitly opted out via AllowUnauthenticated.
// This prevents accidental unauthenticated deployments caused by wiring
// mistakes that forget to thread the token through Dependencies.
func TestMountFailsWithEmptySessionToken(t *testing.T) {
	mux := http.NewServeMux()
	err := Mount(mux, Dependencies{})
	if err == nil {
		t.Fatal("Mount with empty SessionToken returned nil error; want non-nil")
	}
	const want = "SessionToken is required"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error=%q; want it to contain %q", err.Error(), want)
	}
}

// TestMountSucceedsWithAllowUnauthenticated verifies that the explicit
// opt-out path works so unit tests can mount an unauthenticated server
// without having to fabricate a token. Broadcast must still be non-nil
// because LifecycleService is always mounted; AllowUnauthenticated only
// relaxes the SessionToken requirement.
func TestMountSucceedsWithAllowUnauthenticated(t *testing.T) {
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	err := Mount(mux, Dependencies{Broadcast: mgr, AllowUnauthenticated: true})
	if err != nil {
		t.Fatalf("Mount with AllowUnauthenticated=true returned error: %v", err)
	}
}
