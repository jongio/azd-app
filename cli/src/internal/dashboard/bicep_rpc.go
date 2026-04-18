// Production wiring for the Connect BicepService handler. Mirrors the
// legacy GET /api/azure/bicep-template path: per-request credential
// acquisition, then a fresh BicepGenerator backed by ResourceDiscovery.
//
// Kept in the dashboard package (not rpc/) because credential acquisition
// uses dashboard-private helpers (newLogAnalyticsCredential) and tying the
// rpc package to those would invert the dependency direction.

package dashboard

import (
	"context"
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/rpc"
)

// newBicepGeneratorFactory returns a factory that builds a fresh
// BicepGenerator for each request. Wrapping the construction in a closure
// (rather than caching a singleton) matches the legacy REST handler which
// rebuilt credential + discovery per call -- preserving the same auth-token
// freshness and per-request scoping.
func newBicepGeneratorFactory(s *Server) rpc.BicepGeneratorFactory {
	return func(_ context.Context, projectDir string) (rpc.BicepGenerator, error) {
		cred, err := newLogAnalyticsCredential()
		if err != nil {
			// Wrap the sentinel so the rpc handler can map this to
			// FailedPrecondition without the caller having to inspect
			// strings. The original error stays attached for logs.
			return nil, fmt.Errorf("%w: %s", rpc.ErrAzureCredentialsUnavailable, err.Error())
		}
		discovery := azure.NewResourceDiscovery(cred, projectDir)
		return azure.NewBicepGenerator(discovery), nil
	}
}
