package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// =============================================================================
// Stubs
// =============================================================================

// stubServiceLister captures inputs and returns canned outputs so the
// handler tests stay independent of the real serviceinfo package (which
// reads azure.yaml from disk and walks the registry).
type stubServiceLister struct {
	infos []*serviceinfo.ServiceInfo
	err   error

	gotProjectDir string
	calls         int
}

func (s *stubServiceLister) ListServices(_ context.Context, projectDir string) ([]*serviceinfo.ServiceInfo, error) {
	s.calls++
	s.gotProjectDir = projectDir
	return s.infos, s.err
}

// stubServiceLifecycle records the call shape and returns whatever the
// test pre-loaded for that operation. Reusing one stub for all three RPCs
// (vs three stubs) keeps the test wiring symmetric and lets a single
// "configure once, call any of them" pattern drive the table tests below.
type stubServiceLifecycle struct {
	startMessage   string
	startErr       error
	stopMessage    string
	stopErr        error
	restartMessage string
	restartErr     error

	calls []lifecycleCall
}

type lifecycleCall struct {
	op     string
	name   string
	noWait bool
	force  bool
}

func (s *stubServiceLifecycle) StartService(_ context.Context, name string, noWait bool) (string, error) {
	s.calls = append(s.calls, lifecycleCall{op: "start", name: name, noWait: noWait})
	return s.startMessage, s.startErr
}

func (s *stubServiceLifecycle) StopService(_ context.Context, name string, force bool) (string, error) {
	s.calls = append(s.calls, lifecycleCall{op: "stop", name: name, force: force})
	return s.stopMessage, s.stopErr
}

func (s *stubServiceLifecycle) RestartService(_ context.Context, name string) (string, error) {
	s.calls = append(s.calls, lifecycleCall{op: "restart", name: name})
	return s.restartMessage, s.restartErr
}

// newServicesTestServer wires a ServicesHandler behind an httptest server.
// Tests mutate the supplied stubs to drive each scenario; the cleanup
// closure shuts down both the HTTP server and the broadcast manager so
// the goroutine count stays flat across tests.
func newServicesTestServer(
	t *testing.T,
	lister ServiceLister,
	lifecycle ServiceLifecycle,
	projectDir string,
) (azdappv1connect.ServicesServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:         mgr,
		ProjectDir:        projectDir,
		ServicesLister:    lister,
		ServicesLifecycle: lifecycle,
	})

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewServicesServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

// =============================================================================
// Construction guards
// =============================================================================

func TestNewServicesHandlerPanicsOnNilLister(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil lister")
		}
	}()
	_ = NewServicesHandler(nil, &stubServiceLifecycle{}, "/p")
}

func TestNewServicesHandlerPanicsOnNilLifecycle(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil lifecycle")
		}
	}()
	_ = NewServicesHandler(&stubServiceLister{}, nil, "/p")
}

func TestNewServicesHandlerPanicsOnEmptyProjectDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty projectDir")
		}
	}()
	_ = NewServicesHandler(&stubServiceLister{}, &stubServiceLifecycle{}, "")
}

// =============================================================================
// Mounting
// =============================================================================

// TestServicesNotMountedWhenDepsMissing verifies the conditional Mount
// behavior: a Connect call to ServicesService should 404 when the
// dashboard wires only a partial dependency set. Catching this here keeps
// the production wiring honest -- accidentally dropping ServicesLifecycle
// from Dependencies should fail tests, not silently degrade.
func TestServicesNotMountedWhenDepsMissing(t *testing.T) {
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:  mgr,
		ProjectDir: "/p",
		// ServicesLister + ServicesLifecycle deliberately unset.
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := azdappv1connect.NewServicesServiceClient(srv.Client(), srv.URL)

	_, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err == nil {
		t.Fatal("expected error from unmounted ServicesService")
	}
	if got := connect.CodeOf(err); got != connect.CodeUnimplemented {
		t.Fatalf("expected CodeUnimplemented, got %v: %v", got, err)
	}
}

// =============================================================================
// GetServices
// =============================================================================

func TestGetServicesEmpty(t *testing.T) {
	lister := &stubServiceLister{infos: nil}
	client, cleanup := newServicesTestServer(t, lister, &stubServiceLifecycle{}, "/proj")
	defer cleanup()

	resp, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Msg.GetServices()); got != 0 {
		t.Fatalf("expected 0 services, got %d", got)
	}
	if lister.gotProjectDir != "/proj" {
		t.Errorf("lister got projectDir=%q, want %q", lister.gotProjectDir, "/proj")
	}
	if lister.calls != 1 {
		t.Errorf("expected lister to be called once, got %d", lister.calls)
	}
}

// TestGetServicesTranslatesRichServiceInfo locks down the field-by-field
// mapping from serviceinfo.ServiceInfo to the proto wire type. This is
// the highest-leverage test in this file: a translation regression here
// would silently drop fields the dashboard renders.
func TestGetServicesTranslatesRichServiceInfo(t *testing.T) {
	startTime := time.Date(2025, 6, 1, 12, 30, 0, 0, time.UTC)
	lastChecked := time.Date(2025, 6, 1, 12, 30, 5, 0, time.UTC)
	infos := []*serviceinfo.ServiceInfo{
		{
			Name:      "api",
			Host:      "containerapp",
			Language:  "python",
			Framework: "flask",
			Project:   "/proj",
			EnvironmentVars: map[string]string{
				"DB_HOST": "db.example.com",
			},
			Local: &serviceinfo.LocalServiceInfo{
				Status:      "running",
				Health:      "healthy",
				URL:         "http://localhost:5000",
				CustomURL:   "https://api.ngrok.io",
				Port:        5000,
				PID:         12345,
				StartTime:   &startTime,
				LastChecked: &lastChecked,
				ServiceType: "http",
				ServiceMode: "watch",
			},
			Azure: &serviceinfo.AzureServiceInfo{
				URL:                "https://api.azurewebsites.net",
				CustomDomain:       "api.example.com",
				CustomDomainSource: "user",
				ResourceName:       "rg-myapp/api",
				ImageName:          "myacr/api:latest",
			},
		},
	}
	lister := &stubServiceLister{infos: infos}
	client, cleanup := newServicesTestServer(t, lister, &stubServiceLifecycle{}, "/proj")
	defer cleanup()

	resp, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := resp.Msg.GetServices()
	if len(got) != 1 {
		t.Fatalf("want 1 service, got %d", len(got))
	}
	svc := got[0]
	if svc.GetName() != "api" || svc.GetFramework() != "flask" || svc.GetLanguage() != "python" {
		t.Errorf("name/framework/language mismatch: %+v", svc)
	}
	if svc.GetKind() != "containerapp" {
		t.Errorf("kind: got %q want %q", svc.GetKind(), "containerapp")
	}
	if svc.GetUrl() != "https://api.ngrok.io" {
		// CustomURL must take precedence over URL.
		t.Errorf("url: got %q want CustomURL https://api.ngrok.io", svc.GetUrl())
	}
	if svc.GetStatus() != v1.ServiceStatus_SERVICE_STATUS_READY {
		t.Errorf("status: got %v want READY", svc.GetStatus())
	}
	if svc.GetHealth() != v1.HealthState_HEALTH_STATE_HEALTHY {
		t.Errorf("health: got %v want HEALTHY", svc.GetHealth())
	}
	if svc.GetPort() != 5000 || svc.GetPid() != 12345 {
		t.Errorf("port/pid mismatch: port=%d pid=%d", svc.GetPort(), svc.GetPid())
	}
	if svc.GetStartTime() == nil || !svc.GetStartTime().AsTime().Equal(startTime) {
		t.Errorf("startTime mismatch: %v", svc.GetStartTime())
	}
	if svc.GetProjectDir() != "/proj" {
		t.Errorf("projectDir: got %q want /proj", svc.GetProjectDir())
	}
	if env := svc.GetEnvironment(); env["DB_HOST"] != "db.example.com" {
		t.Errorf("environment missing DB_HOST: %v", env)
	}
	azure := svc.GetAzure()
	if azure == nil {
		t.Fatal("azure deployment info missing")
	}
	if azure.GetResourceId() != "rg-myapp/api" {
		t.Errorf("azure ResourceId: got %q", azure.GetResourceId())
	}
	if azure.GetResourceType() != "containerapp" {
		t.Errorf("azure ResourceType: got %q want containerapp", azure.GetResourceType())
	}

	// Metadata Struct should preserve fields with no typed home.
	meta := svc.GetMetadata()
	if meta == nil {
		t.Fatal("metadata struct missing")
	}
	mfields := meta.GetFields()
	if mfields["serviceType"].GetStringValue() != "http" {
		t.Errorf("metadata.serviceType: got %v", mfields["serviceType"])
	}
	if mfields["serviceMode"].GetStringValue() != "watch" {
		t.Errorf("metadata.serviceMode: got %v", mfields["serviceMode"])
	}
	if mfields["autoUrl"].GetStringValue() != "http://localhost:5000" {
		t.Errorf("metadata.autoUrl: got %v", mfields["autoUrl"])
	}
	azureMeta := mfields["azure"].GetStructValue().GetFields()
	if azureMeta["customDomain"].GetStringValue() != "api.example.com" {
		t.Errorf("metadata.azure.customDomain: got %v", azureMeta["customDomain"])
	}
	if azureMeta["imageName"].GetStringValue() != "myacr/api:latest" {
		t.Errorf("metadata.azure.imageName: got %v", azureMeta["imageName"])
	}
}

// TestGetServicesURLFallsBackToLocalURL covers the inverse of the
// CustomURL-precedence rule: when CustomURL is empty, URL must surface.
// The two cases together pin the precedence contract.
func TestGetServicesURLFallsBackToLocalURL(t *testing.T) {
	infos := []*serviceinfo.ServiceInfo{
		{
			Name: "web",
			Local: &serviceinfo.LocalServiceInfo{
				Status: "stopped",
				URL:    "http://localhost:3000",
			},
		},
	}
	lister := &stubServiceLister{infos: infos}
	client, cleanup := newServicesTestServer(t, lister, &stubServiceLifecycle{}, "/p")
	defer cleanup()

	resp, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Msg.GetServices()[0].GetUrl(); got != "http://localhost:3000" {
		t.Errorf("url: got %q want http://localhost:3000", got)
	}
	if got := resp.Msg.GetServices()[0].GetStatus(); got != v1.ServiceStatus_SERVICE_STATUS_STOPPED {
		t.Errorf("status: got %v want STOPPED", got)
	}
}

func TestGetServicesSkipsNilEntries(t *testing.T) {
	infos := []*serviceinfo.ServiceInfo{
		{Name: "a"},
		nil,
		{Name: "b"},
	}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{infos: infos}, &stubServiceLifecycle{}, "/p")
	defer cleanup()

	resp, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.Msg.GetServices()); got != 2 {
		t.Errorf("want 2 services after nil skip, got %d", got)
	}
}

func TestGetServicesPropagatesListerError(t *testing.T) {
	lister := &stubServiceLister{err: errors.New("disk on fire")}
	client, cleanup := newServicesTestServer(t, lister, &stubServiceLifecycle{}, "/p")
	defer cleanup()

	_, err := client.GetServices(context.Background(), connect.NewRequest(&v1.GetServicesRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("code: got %v want Internal", got)
	}
	if !strings.Contains(err.Error(), "disk on fire") {
		t.Errorf("error must include underlying message; got %q", err.Error())
	}
}

// =============================================================================
// Lifecycle ops -- success
// =============================================================================

func TestStartServiceSuccessSingle(t *testing.T) {
	lc := &stubServiceLifecycle{startMessage: "1 service(s) started, 0 failed"}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
	defer cleanup()

	resp, err := client.StartService(context.Background(), connect.NewRequest(&v1.StartServiceRequest{
		ServiceName: "api",
		NoWait:      true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.Msg.GetResult()
	if r == nil || !r.GetSuccess() {
		t.Fatal("result should be success=true")
	}
	if r.GetMessage() != "1 service(s) started, 0 failed" {
		t.Errorf("message: got %q", r.GetMessage())
	}
	if r.GetOperationId() == "" {
		t.Error("operation_id should be set")
	}
	if r.GetCompletedAt() == nil {
		t.Error("completed_at should be set")
	}

	if len(lc.calls) != 1 || lc.calls[0].op != "start" || lc.calls[0].name != "api" || !lc.calls[0].noWait {
		t.Errorf("lifecycle call mismatch: %+v", lc.calls)
	}
}

func TestStartServiceSuccessBulk(t *testing.T) {
	lc := &stubServiceLifecycle{startMessage: "3 service(s) started, 0 failed"}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
	defer cleanup()

	resp, err := client.StartService(context.Background(), connect.NewRequest(&v1.StartServiceRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetResult().GetMessage() != "3 service(s) started, 0 failed" {
		t.Errorf("bulk message: got %q", resp.Msg.GetResult().GetMessage())
	}
	if lc.calls[0].name != "" {
		t.Errorf("bulk call must pass empty name, got %q", lc.calls[0].name)
	}
}

func TestStopServiceSuccess(t *testing.T) {
	lc := &stubServiceLifecycle{stopMessage: "1 service(s) stopped, 0 failed"}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
	defer cleanup()

	resp, err := client.StopService(context.Background(), connect.NewRequest(&v1.StopServiceRequest{
		ServiceName: "api",
		Force:       true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetResult().GetSuccess() {
		t.Error("expected success")
	}
	if !lc.calls[0].force {
		t.Error("force flag should be propagated")
	}
}

func TestRestartServiceSuccess(t *testing.T) {
	lc := &stubServiceLifecycle{restartMessage: "1 service(s) restarted, 0 failed"}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
	defer cleanup()

	resp, err := client.RestartService(context.Background(), connect.NewRequest(&v1.RestartServiceRequest{
		ServiceName: "api",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetResult().GetSuccess() {
		t.Error("expected success")
	}
	if lc.calls[0].op != "restart" || lc.calls[0].name != "api" {
		t.Errorf("call: %+v", lc.calls[0])
	}
}

// =============================================================================
// Lifecycle ops -- error code mapping
// =============================================================================

func TestLifecycleErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
		{"NotFound", ErrServiceNotFound, connect.CodeNotFound},
		{"WrappedNotFound", errors.New("boom: " + ErrServiceNotFound.Error()), connect.CodeInternal}, // unwrapped strings != errors.Is
		{"InvalidState", ErrServiceInvalidState, connect.CodeFailedPrecondition},
		{"OpInProgress", ErrServiceOpInProgress, connect.CodeAlreadyExists},
		{"Generic", errors.New("disk full"), connect.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lc := &stubServiceLifecycle{startErr: tc.err}
			client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
			defer cleanup()

			_, err := client.StartService(context.Background(), connect.NewRequest(&v1.StartServiceRequest{
				ServiceName: "x",
			}))
			if err == nil {
				t.Fatal("expected error")
			}
			if got := connect.CodeOf(err); got != tc.wantCode {
				t.Errorf("code: got %v want %v (err=%v)", got, tc.wantCode, err)
			}
		})
	}
}

// TestLifecycleSentinelWrappingHonorsErrorsIs ensures a wrapped sentinel
// (the actual production pattern -- the dashboard adapter wraps with
// fmt.Errorf("%w: %s", sentinel, name)) still maps to the right code.
func TestLifecycleSentinelWrappingHonorsErrorsIs(t *testing.T) {
	wrapped := errors.New("wrapped via Unwrap chain")
	wrappedErr := errorChain{outer: wrapped, inner: ErrServiceNotFound}

	lc := &stubServiceLifecycle{startErr: wrappedErr}
	client, cleanup := newServicesTestServer(t, &stubServiceLister{}, lc, "/p")
	defer cleanup()

	_, err := client.StartService(context.Background(), connect.NewRequest(&v1.StartServiceRequest{ServiceName: "x"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Errorf("wrapped sentinel must map to NotFound, got %v", got)
	}
}

// errorChain is a minimal Unwrap-supporting wrapper used by the test
// above. Lives in the test file because no production code needs it.
type errorChain struct {
	outer error
	inner error
}

func (e errorChain) Error() string { return e.outer.Error() }
func (e errorChain) Unwrap() error { return e.inner }
