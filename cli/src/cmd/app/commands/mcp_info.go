// getAppInfoForMCP / processEnvMap back the `services`, `project info`, and
// `environment variables` MCP tools. They replace the Stage 2 implementation
// that shelled out to `azd app info --output json`; the subprocess path cost
// ~200ms per call and produced nothing the in-process path can't reproduce.
// See docs/adr/0001-connect-go-transport.md Stage 3 for the rationale.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// getAppInfoForMCP returns the same JSON payload shape as `azd app info
// --output json` (see printInfoJSON in info.go). Tries the typed Connect
// client first to pick up runtime state from a running dashboard; falls
// back to the static azure.yaml-based view when no dashboard is reachable.
//
// EnvironmentVars are populated for every service with Azure info: the
// process environment is scanned for SERVICE_<NAME>_* / <NAME>_* keys and
// secret-like values are masked via redactSecretValue. Result is JSON
// round-tripped so MCP tool handlers continue to consume
// map[string]any the way they did when the source was subprocess
// stdout.
func getAppInfoForMCP(ctx context.Context, projectDir string) (map[string]any, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	services, err := collectServiceInfoForMCP(ctx, projectDir)
	if err != nil {
		return nil, err
	}

	azureEnv := processEnvMap()
	outputServices := make([]serviceinfo.ServiceInfo, 0, len(services))
	for _, svc := range services {
		if svc == nil {
			continue
		}
		populateAzureEnv(svc, azureEnv)
		outputServices = append(outputServices, *svc)
	}

	// JSON round-trip so callers see map[string]any (MCP handlers
	// index into this shape); marshalling the typed struct directly would
	// hand back a pointer the handlers aren't prepared to type-assert.
	raw, err := json.Marshal(map[string]any{
		"project":  projectDir,
		"services": outputServices,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode service info: %w", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("failed to decode service info: %w", err)
	}
	return decoded, nil
}

// collectServiceInfoForMCP prefers live dashboard state and falls back to
// azure.yaml parsing. A dashboard error (not running, port not reachable)
// is treated as "use static" rather than fatal: the MCP tool must still
// answer when the user has not started the dashboard.
func collectServiceInfoForMCP(ctx context.Context, projectDir string) ([]*serviceinfo.ServiceInfo, error) {
	if client, err := dashboard.NewClient(ctx, projectDir); err == nil {
		if live, liveErr := client.GetServices(ctx); liveErr == nil {
			return live, nil
		}
	}
	static, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get service info: %w", err)
	}
	return static, nil
}

// populateAzureEnv mirrors the info command's behavior: for services with
// Azure deployment info, surface SERVICE_<NAME>_* and <NAME>_* variables
// from the process environment and mask secret-like values. No-op when the
// service has no Azure info or the environment is empty.
func populateAzureEnv(svc *serviceinfo.ServiceInfo, env map[string]string) {
	if svc == nil || svc.Azure == nil || len(env) == 0 {
		return
	}
	upper := strings.ToUpper(svc.Name)
	prefixA := upper + "_"
	prefixB := "SERVICE_" + upper + "_"
	out := make(map[string]string)
	for k, v := range env {
		ku := strings.ToUpper(k)
		if strings.HasPrefix(ku, prefixA) || strings.HasPrefix(ku, prefixB) {
			out[k] = redactSecretValue(k, v)
		}
	}
	if len(out) > 0 {
		svc.EnvironmentVars = out
	}
}

// processEnvMap is a lightweight os.Environ() -> map[string]string helper
// shared by the MCP info path. Split out so tests can inject a fixture
// without mutating the real process environment.
func processEnvMap() map[string]string {
	env := os.Environ()
	out := make(map[string]string, len(env))
	for _, line := range env {
		if i := strings.IndexByte(line, '='); i > 0 {
			out[line[:i]] = line[i+1:]
		}
	}
	return out
}
