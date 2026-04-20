// Package dashboard exposes a typed client that CLI cobra commands and MCP
// tool handlers use to query a running azd-app dashboard. As of Stage 4 of
// the Connect-Go migration (see docs/adr/0001-connect-go-transport.md) it
// speaks Connect-over-HTTP exclusively against the azdapp.v1.* services;
// the legacy REST and WebSocket surface has been removed. The CLI and
// dashboard are separate OS processes, so this is NOT an in-process client.
// The public surface - NewClient, Ping, GetServices, StreamLogs,
// GetAzureLogs, GetAzureStatus, StreamAzureLogs - is intentionally preserved
// so logs.go / info.go / MCP handlers did not need rewriting.
//
// Auth posture: no interceptor is attached. The dashboard binds to localhost
// only, matching the trust boundary of the previous REST surface. Stage 5+
// may add authentication; scope-limited here on purpose.
//
// GetAzureStatus is synthesised on top of the AzureService.GetAzureServices
// RPC and mirrors the shape of the historical service.AzureStatus struct
// that logs.go / info.go still consume.
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// Client speaks Connect over HTTP to a running dashboard process.
//
// Two http.Clients are kept: unaryHTTP has a short request timeout suitable
// for non-streaming RPCs; streamHTTP has no timeout because Connect server
// streams are long-lived. Ping / GetServices / GetAzureLogs / GetAzureStatus
// apply a per-call timeout via context.WithTimeout regardless of the
// underlying http.Client to keep behavior stable across wrappers.
type Client struct {
	baseURL    string
	unaryHTTP  *http.Client
	streamHTTP *http.Client

	lifecycle azdappv1connect.LifecycleServiceClient
	services  azdappv1connect.ServicesServiceClient
	logs      azdappv1connect.LogsServiceClient
	azure     azdappv1connect.AzureServiceClient
}

// NewClient creates a typed Connect client bound to the dashboard for the
// given project directory. Returns an error when no dashboard is running.
//
// Port resolution order:
//  1. azd's gRPC config service (available inside extension subprocesses).
//  2. Direct read of ~/.azd/config.json (works when the CLI runs standalone).
func NewClient(ctx context.Context, projectDir string) (*Client, error) {
	projectHash := azdconfig.ProjectHash(projectDir)

	port := 0
	if configClient, err := azdconfig.NewClient(ctx); err == nil {
		dashboardPort, portErr := configClient.GetDashboardPort(projectHash)
		configClient.Close()
		if portErr == nil && dashboardPort > 0 {
			port = dashboardPort
		}
	}

	if port == 0 {
		fallback, err := readDashboardPortFromAzdConfig(projectHash)
		if err != nil {
			return nil, fmt.Errorf("dashboard not running for project: %w", err)
		}
		port = fallback
	}

	if port == 0 {
		return nil, errors.New("dashboard not running for project")
	}

	return newClientForPort(port), nil
}

func newClientForPort(port int) *Client {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	unary := &http.Client{Timeout: constants.DashboardAPITimeout}
	stream := &http.Client{}

	return &Client{
		baseURL:    baseURL,
		unaryHTTP:  unary,
		streamHTTP: stream,
		lifecycle:  azdappv1connect.NewLifecycleServiceClient(unary, baseURL),
		services:   azdappv1connect.NewServicesServiceClient(unary, baseURL),
		// Logs + Azure expose streaming RPCs so they need a no-timeout
		// transport; unary RPCs on these services re-apply a deadline via
		// context.WithTimeout inside each wrapper.
		logs:  azdappv1connect.NewLogsServiceClient(stream, baseURL),
		azure: azdappv1connect.NewAzureServiceClient(stream, baseURL),
	}
}

// Ping checks that the dashboard is reachable.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, constants.DashboardAPITimeout)
	defer cancel()
	_, err := c.lifecycle.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	return err
}

// GetServices returns the merged service list (azure.yaml + runtime state +
// Azure deployment info). Fields that do not survive the proto schema round
// trip are restored from the metadata Struct populated by serviceInfoToProto.
func (c *Client) GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DashboardAPITimeout)
	defer cancel()
	resp, err := c.services.GetServices(ctx, connect.NewRequest(&v1.GetServicesRequest{}))
	if err != nil {
		return nil, err
	}

	out := make([]*serviceinfo.ServiceInfo, 0, len(resp.Msg.GetServices()))
	for _, p := range resp.Msg.GetServices() {
		out = append(out, protoToServiceInfo(p))
	}
	return out, nil
}

// StreamLogs tails one service (or all services when serviceName is empty)
// until the context is cancelled. DroppedNotice events are logged to stderr
// so operators see back-pressure loss without polluting the log channel.
// The caller owns the channel and must close it.
func (c *Client) StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error {
	stream, err := c.logs.StreamLocalLogs(ctx, connect.NewRequest(&v1.StreamLocalLogsRequest{
		ServiceName: serviceName,
	}))
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		msg := stream.Msg()
		switch ev := msg.GetEvent().(type) {
		case *v1.StreamLocalLogsResponse_Entry:
			entry := protoToLogEntry(ev.Entry)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case logs <- entry:
			}
		case *v1.StreamLocalLogsResponse_Dropped:
			noticeDropped(os.Stderr, serviceName, ev.Dropped.GetCount())
		}
	}
	if err := stream.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// GetAzureLogs fetches buffered Azure logs for the given services. The proto
// request tails one service at a time; multi-service callers are served by
// issuing one RPC per service and merging results. Client-side `since`
// filtering matches the legacy REST behavior (since_seconds is not a
// 1:1 replacement because the server clamps it differently).
func (c *Client) GetAzureLogs(ctx context.Context, services []string, tail int, since time.Time) ([]service.LogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DashboardAPITimeout)
	defer cancel()

	serviceList := services
	if len(serviceList) == 0 {
		// Empty list => let the server default to "merged across all
		// SERVICE_*_NAME entries" (matches legacy behavior).
		serviceList = []string{""}
	}

	var all []service.LogEntry
	for _, svc := range serviceList {
		resp, err := c.azure.GetAzureLogs(ctx, connect.NewRequest(&v1.GetAzureLogsRequest{
			Service: svc,
			Tail:    int32(tail),
		}))
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Msg.GetEntries() {
			all = append(all, protoToLogEntry(p))
		}
	}

	if !since.IsZero() {
		filtered := all[:0]
		for _, entry := range all {
			if !entry.Timestamp.Before(since) {
				filtered = append(filtered, entry)
			}
		}
		all = filtered
	}
	return all, nil
}

// GetAzureStatus mirrors the legacy service.AzureStatus shape logs.go /
// info.go consume, by probing GetAzureServices and deriving the minimum
// fields those call sites read (Mode / Enabled / Connected / ResourceCount).
// Returns a disabled-shape response when Azure is not configured.
//
//nolint:staticcheck // service.AzureStatus is deprecated but kept for API compat.
func (c *Client) GetAzureStatus(ctx context.Context) (*service.AzureStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DashboardAPITimeout)
	defer cancel()

	resp, err := c.azure.GetAzureServices(ctx, connect.NewRequest(&v1.GetAzureServicesRequest{}))
	if err != nil {
		// Azure not configured or handler errored -- matches the "return
		// disabled" branch of the old REST client.
		return &service.AzureStatus{
			Mode:      service.LogModeLocal,
			Connected: false,
			Enabled:   false,
		}, nil
	}

	services := resp.Msg.GetServices()
	return &service.AzureStatus{
		Mode:          service.LogModeAzure,
		Enabled:       len(services) > 0,
		Connected:     len(services) > 0,
		ResourceCount: len(services),
	}, nil
}

// StreamAzureLogs streams Azure logs for every service discovered via
// GetAzureServices. The proto StreamAzureLogs RPC requires a non-empty
// service name, so multi-service streaming is achieved by fanning out one
// stream per service and merging into the shared channel. Per-stream status
// transitions and dropped notices are logged to stderr. The caller owns the
// channel and must close it.
func (c *Client) StreamAzureLogs(ctx context.Context, logs chan<- service.LogEntry) error {
	listCtx, listCancel := context.WithTimeout(ctx, constants.DashboardAPITimeout)
	servicesResp, err := c.azure.GetAzureServices(listCtx, connect.NewRequest(&v1.GetAzureServicesRequest{}))
	listCancel()
	if err != nil {
		return fmt.Errorf("failed to list Azure services: %w", err)
	}

	services := servicesResp.Msg.GetServices()
	if len(services) == 0 {
		// No Azure services configured -- legacy behavior was to block
		// until the context is cancelled so the caller's channel merge
		// stays alive. Mirror that here.
		<-ctx.Done()
		return ctx.Err()
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(services))

	for _, svc := range services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.streamOneAzureService(streamCtx, svc, logs); err != nil && !errors.Is(err, context.Canceled) {
				// First error cancels siblings; later errors are logged
				// and discarded because the caller only gets the first.
				select {
				case errCh <- err:
					cancel()
				default:
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	if err, ok := <-errCh; ok {
		return err
	}
	return nil
}

func (c *Client) streamOneAzureService(ctx context.Context, svc string, logs chan<- service.LogEntry) error {
	stream, err := c.azure.StreamAzureLogs(ctx, connect.NewRequest(&v1.StreamAzureLogsRequest{
		Service: svc,
	}))
	if err != nil {
		return fmt.Errorf("failed to open Azure log stream for %q: %w", svc, err)
	}
	defer func() { _ = stream.Close() }()

	tracker := azureStreamStatusTracker{service: svc}
	for stream.Receive() {
		msg := stream.Msg()
		switch ev := msg.GetEvent().(type) {
		case *v1.StreamAzureLogsResponse_Entry:
			entry := protoToLogEntry(ev.Entry)
			if entry.Source == "" {
				entry.Source = service.LogSourceAzure
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case logs <- entry:
			}
		case *v1.StreamAzureLogsResponse_Status:
			tracker.observe(os.Stderr, ev.Status)
		case *v1.StreamAzureLogsResponse_Dropped:
			noticeAzureDropped(os.Stderr, svc, ev.Dropped)
		}
	}
	if err := stream.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// azureStreamStatusTracker prints a line every time the Azure stream status
// transitions (connected -> degraded -> disconnected). Keeps stderr quiet
// when nothing interesting is happening.
type azureStreamStatusTracker struct {
	service string
	last    string
}

func (t *azureStreamStatusTracker) observe(w io.Writer, s *v1.StreamStatus) {
	if s == nil {
		return
	}
	if s.GetStatus() == t.last {
		return
	}
	t.last = s.GetStatus()
	_, _ = fmt.Fprintf(w, "azd-app: azure log stream for %q is %s (mode=%s, fails=%d)%s\n",
		t.service, s.GetStatus(), s.GetMode(), s.GetConsecutiveFails(), formatStreamError(s.GetError()))
}

func formatStreamError(msg string) string {
	if msg == "" {
		return ""
	}
	return ": " + msg
}

func noticeDropped(w io.Writer, svc string, count int64) {
	if count <= 0 {
		return
	}
	target := svc
	if target == "" {
		target = "all services"
	}
	_, _ = fmt.Fprintf(w, "azd-app: dropped %d local log entr%s for %s (subscriber fell behind)\n",
		count, pluralEntries(count), target)
}

func noticeAzureDropped(w io.Writer, svc string, d *v1.AzureDroppedNotice) {
	if d == nil || d.GetCount() <= 0 {
		return
	}
	reason := d.GetReason()
	if reason == "" {
		reason = "realtime_buffer_overflow"
	}
	_, _ = fmt.Fprintf(w, "azd-app: dropped %d azure log entr%s for %q (%s)\n",
		d.GetCount(), pluralEntries(d.GetCount()), svc, reason)
}

func pluralEntries(n int64) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// IsDashboardRunning checks if a dashboard is running for the given project.
func IsDashboardRunning(ctx context.Context, projectDir string) bool {
	client, err := NewClient(ctx, projectDir)
	if err != nil {
		return false
	}
	return client.Ping(ctx) == nil
}

// GetDashboardPort returns the dashboard port for a project, or 0 if not running.
func GetDashboardPort(ctx context.Context, projectDir string) int {
	projectHash := azdconfig.ProjectHash(projectDir)

	if configClient, err := azdconfig.NewClient(ctx); err == nil {
		port, err := configClient.GetDashboardPort(projectHash)
		configClient.Close()
		if err == nil && port > 0 {
			return port
		}
	}

	port, _ := readDashboardPortFromAzdConfig(projectHash)
	return port
}

// azdConfigPath returns the path to the azd config file. Declared as a
// variable so tests can redirect it; see client_test.go.
var azdConfigPath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return fmt.Sprintf("%s/.azd/config.json", homeDir), nil
}

// azdConfig / appConfig / projectConfig mirror the subset of ~/.azd/config.json
// the dashboard port lookup needs.
type azdConfig struct {
	App *appConfig `json:"app"`
}

type appConfig struct {
	Projects map[string]*projectConfig `json:"projects"`
}

type projectConfig struct {
	DashboardPort int `json:"dashboardPort"`
}

// readDashboardPortFromAzdConfig reads the dashboard port directly from
// ~/.azd/config.json. Used as a fallback when the gRPC config service is
// unavailable (running the CLI from a separate terminal).
func readDashboardPortFromAzdConfig(projectHash string) (int, error) {
	configPath, err := azdConfigPath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read azd config: %w", err)
	}

	var cfg azdConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, fmt.Errorf("failed to parse azd config: %w", err)
	}

	if cfg.App == nil || cfg.App.Projects == nil {
		return 0, nil
	}

	proj, ok := cfg.App.Projects[projectHash]
	if !ok || proj == nil {
		return 0, nil
	}

	return proj.DashboardPort, nil
}
