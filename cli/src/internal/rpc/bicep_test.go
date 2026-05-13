package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
)

// =============================================================================
// Stubs
// =============================================================================

// stubBicepGenerator returns canned data so the handler tests stay decoupled
// from azure.NewBicepGenerator (which needs a real ARM credential and walks
// the discovery pipeline).
type stubBicepGenerator struct {
	resp *azure.BicepTemplateResponse
	err  error

	calls int
}

func (s *stubBicepGenerator) GenerateTemplate(_ context.Context) (*azure.BicepTemplateResponse, error) {
	s.calls++
	return s.resp, s.err
}

// stubBicepFactory adapts a generator + optional construction error into a
// BicepGeneratorFactory. The constructed-error path covers the cases where
// credential acquisition or other per-request setup fails before the
// generator runs.
type stubBicepFactory struct {
	gen *stubBicepGenerator
	err error

	calls          int
	gotProjectDir  string
	deadlineWasSet bool
}

func (f *stubBicepFactory) factory() BicepGeneratorFactory {
	return func(ctx context.Context, projectDir string) (BicepGenerator, error) {
		f.calls++
		f.gotProjectDir = projectDir
		_, ok := ctx.Deadline()
		f.deadlineWasSet = ok
		if f.err != nil {
			return nil, f.err
		}
		return f.gen, nil
	}
}

// newBicepTestServer wires a BicepHandler behind an httptest server.
func newBicepTestServer(
	t *testing.T,
	factory BicepGeneratorFactory,
	projectDir string,
) (azdappv1connect.BicepServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:    mgr,
		ProjectDir:   projectDir,
		BicepFactory: factory,
	})

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewBicepServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

// =============================================================================
// Construction guards
// =============================================================================

func TestNewBicepHandlerPanicsOnNilFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil factory")
		}
	}()
	_ = NewBicepHandler(nil, "/p")
}

func TestNewBicepHandlerPanicsOnEmptyProjectDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty projectDir")
		}
	}()
	_ = NewBicepHandler(func(context.Context, string) (BicepGenerator, error) { return nil, nil }, "")
}

// =============================================================================
// Mount conditional
// =============================================================================

// TestBicepNotMountedWhenFactoryMissing: dropping BicepFactory from
// Dependencies should yield CodeUnimplemented, not silently degrade.
func TestBicepNotMountedWhenFactoryMissing(t *testing.T) {
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:  mgr,
		ProjectDir: "/p",
		// BicepFactory deliberately unset.
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := azdappv1connect.NewBicepServiceClient(srv.Client(), srv.URL)

	_, err := client.GetBicepTemplate(context.Background(), connect.NewRequest(&v1.GetBicepTemplateRequest{}))
	if err == nil {
		t.Fatal("expected error from unmounted BicepService")
	}
	if got := connect.CodeOf(err); got != connect.CodeUnimplemented {
		t.Fatalf("expected CodeUnimplemented, got %v: %v", got, err)
	}
}

// =============================================================================
// GetBicepTemplate
// =============================================================================

func TestGetBicepTemplateSuccess(t *testing.T) {
	gen := &stubBicepGenerator{
		resp: &azure.BicepTemplateResponse{
			Template: "// generated bicep",
			Services: []string{"api", "web"},
		},
	}
	stub := &stubBicepFactory{gen: gen}

	before := time.Now().UTC()
	client, cleanup := newBicepTestServer(t, stub.factory(), "/proj")
	defer cleanup()

	resp, err := client.GetBicepTemplate(
		context.Background(),
		connect.NewRequest(&v1.GetBicepTemplateRequest{ServiceNames: []string{}}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := resp.Msg.GetTemplate(); got != "// generated bicep" {
		t.Errorf("template = %q, want %q", got, "// generated bicep")
	}
	if got := resp.Msg.GetIncludedServices(); len(got) != 2 || got[0] != "api" || got[1] != "web" {
		t.Errorf("includedServices = %v, want [api web]", got)
	}
	if got := resp.Msg.GetWorkspaceId(); got != "" {
		t.Errorf("workspaceId = %q, want empty (legacy parity)", got)
	}
	ts := resp.Msg.GetGeneratedAt()
	if ts == nil {
		t.Fatal("generatedAt missing")
	}
	if got := ts.AsTime(); got.Before(before) || got.After(time.Now().UTC().Add(time.Second)) {
		t.Errorf("generatedAt %v not in expected window", got)
	}

	// Factory plumbing checks.
	if stub.calls != 1 || gen.calls != 1 {
		t.Errorf("call counts: factory=%d gen=%d, want 1/1", stub.calls, gen.calls)
	}
	if stub.gotProjectDir != "/proj" {
		t.Errorf("factory got projectDir=%q, want /proj", stub.gotProjectDir)
	}
	if !stub.deadlineWasSet {
		t.Error("expected handler to set a deadline on factory ctx")
	}
}

func TestGetBicepTemplateErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		genErr   error
		wantCode connect.Code
	}{
		{"NoResourcesSentinel", ErrNoAzureResources, connect.CodeNotFound},
		{"DiscoverySentinel", ErrAzureDiscoveryFailed, connect.CodeUnavailable},
		{"WrappedNoResources", fmt.Errorf("ctx: %w", ErrNoAzureResources), connect.CodeNotFound},
		{"NoResourcesString", errors.New("no Azure resources found in environment"), connect.CodeNotFound},
		{"DiscoveryString", errors.New("failed to discover resources: boom"), connect.CodeUnavailable},
		{"Generic", errors.New("disk full"), connect.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gen := &stubBicepGenerator{err: tc.genErr}
			stub := &stubBicepFactory{gen: gen}
			client, cleanup := newBicepTestServer(t, stub.factory(), "/p")
			defer cleanup()

			_, err := client.GetBicepTemplate(
				context.Background(),
				connect.NewRequest(&v1.GetBicepTemplateRequest{}),
			)
			if err == nil {
				t.Fatal("expected error")
			}
			if got := connect.CodeOf(err); got != tc.wantCode {
				t.Fatalf("got code=%v, want %v (err=%v)", got, tc.wantCode, err)
			}
		})
	}
}

// TestGetBicepTemplateFactoryCredentialError verifies that a factory failure
// (e.g. credential acquisition failure) maps to FailedPrecondition when the
// sentinel is returned, mirroring the legacy 401 response semantics in
// Connect-native form.
func TestGetBicepTemplateFactoryCredentialError(t *testing.T) {
	stub := &stubBicepFactory{err: ErrAzureCredentialsUnavailable}
	client, cleanup := newBicepTestServer(t, stub.factory(), "/p")
	defer cleanup()

	_, err := client.GetBicepTemplate(
		context.Background(),
		connect.NewRequest(&v1.GetBicepTemplateRequest{}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("got code=%v, want FailedPrecondition", got)
	}
}

// TestGetBicepTemplateFactoryGenericError: an un-classified factory error
// should not leak as Internal when the upstream cause is e.g. a config
// problem -- but until we have richer sentinels, Internal is the honest
// answer. This test pins that contract so a future change is intentional.
func TestGetBicepTemplateFactoryGenericError(t *testing.T) {
	stub := &stubBicepFactory{err: errors.New("config missing")}
	client, cleanup := newBicepTestServer(t, stub.factory(), "/p")
	defer cleanup()

	_, err := client.GetBicepTemplate(
		context.Background(),
		connect.NewRequest(&v1.GetBicepTemplateRequest{}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("got code=%v, want Internal", got)
	}
}

// TestGetBicepTemplateClassifyDeadlineExceeded covers the explicit
// classifyBicepError ctx-wins path. We exercise it directly to keep the
// test deterministic (network-driven deadline tests are flaky on slow CI).
func TestGetBicepTemplateClassifyDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := classifyBicepError(ctx, errors.New("upstream said something else"))
	if got := connect.CodeOf(err); got != connect.CodeDeadlineExceeded {
		t.Fatalf("got code=%v, want DeadlineExceeded", got)
	}
}

func TestGetBicepTemplateClassifyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := classifyBicepError(ctx, errors.New("upstream"))
	if got := connect.CodeOf(err); got != connect.CodeCanceled {
		t.Fatalf("got code=%v, want Canceled", got)
	}
}
