package dashboard

import (
	"context"
	"os"
	"time"
)

// timeoutContext creates a context with the given timeout, rooted at
// context.Background(). Shared by the kept Azure Monitor query helpers
// (azure_setup.go, azure_logs_health.go) for bounding upstream ARM calls.
func timeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// getEnvironment returns all process environment variables. Extracted as a
// small seam so tests can stub it (currently unused in tests, retained for
// parity with azure_logs_config.go's workspace ID discovery path).
func getEnvironment() []string {
	return os.Environ()
}
