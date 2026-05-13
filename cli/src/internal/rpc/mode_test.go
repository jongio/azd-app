package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// memoryModeStore is a thread-safe ModeStore for tests.
type memoryModeStore struct {
	mu      sync.RWMutex
	current service.LogMode
}

func (s *memoryModeStore) GetMode() service.LogMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *memoryModeStore) SetMode(m service.LogMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = m
}

// newModeTestServer wires a ModeHandler behind an httptest server. The
// caller controls both the ModeStore (state) and the ProjectSource
// (azure.yaml shape) so each scenario stays self-contained.
func newModeTestServer(
	t *testing.T,
	store ModeStore,
	source ProjectSource,
	projectDir string,
) (azdappv1connect.ModeServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:  mgr,
		Project:    source,
		ProjectDir: projectDir,
		Mode:       store,
	})

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewModeServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

func azureConfiguredYaml(realtime bool) *service.AzureYaml {
	return &service.AzureYaml{
		Logs: &service.LogsConfig{
			Analytics: &service.AnalyticsConfigGlobal{Realtime: realtime},
		},
	}
}

// --- GetMode ---

func TestGetModeReportsLocalAndAzureDisabledByDefault(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeLocal}
	source := &stubProjectSource{yaml: &service.AzureYaml{}} // logs section absent
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	resp, err := client.GetMode(context.Background(), connect.NewRequest(&v1.GetModeRequest{}))
	if err != nil {
		t.Fatalf("GetMode error: %v", err)
	}
	if got := resp.Msg.GetMode(); got != v1.LogMode_LOG_MODE_LOCAL {
		t.Errorf("Mode=%v want LOCAL", got)
	}
	if resp.Msg.GetAzureEnabled() {
		t.Error("AzureEnabled=true want false")
	}
	if got := resp.Msg.GetAzureStatus(); got != "disabled" {
		t.Errorf("AzureStatus=%q want disabled", got)
	}
	if got := resp.Msg.GetConnectionMessage(); got != "Azure logging not configured in azure.yaml" {
		t.Errorf("ConnectionMessage=%q want missing-config message", got)
	}
}

func TestGetModeReportsAzureConnectedWhenConfigured(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeAzure}
	source := &stubProjectSource{yaml: azureConfiguredYaml(true)}
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	resp, err := client.GetMode(context.Background(), connect.NewRequest(&v1.GetModeRequest{}))
	if err != nil {
		t.Fatalf("GetMode error: %v", err)
	}
	if got := resp.Msg.GetMode(); got != v1.LogMode_LOG_MODE_AZURE {
		t.Errorf("Mode=%v want AZURE", got)
	}
	if !resp.Msg.GetAzureEnabled() {
		t.Error("AzureEnabled=false want true")
	}
	if got := resp.Msg.GetAzureStatus(); got != "connected" {
		t.Errorf("AzureStatus=%q want connected", got)
	}
	if !resp.Msg.GetAzureRealtime() {
		t.Error("AzureRealtime=false want true")
	}
	if got := resp.Msg.GetConnectionMessage(); got != "" {
		t.Errorf("ConnectionMessage=%q want empty when connected", got)
	}
}

func TestGetModeSurfacesAzureYamlLoadError(t *testing.T) {
	// A parse error must be reported via connection_message, NOT as an
	// Internal error. The dashboard still needs to render the current
	// log-source toggle even when azure.yaml is broken.
	store := &memoryModeStore{current: service.LogModeLocal}
	source := &stubProjectSource{err: errors.New("read azure.yaml: permission denied")}
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	resp, err := client.GetMode(context.Background(), connect.NewRequest(&v1.GetModeRequest{}))
	if err != nil {
		t.Fatalf("GetMode unexpected error: %v", err)
	}
	if got := resp.Msg.GetAzureStatus(); got != "disabled" {
		t.Errorf("AzureStatus=%q want disabled on load error", got)
	}
	if got := resp.Msg.GetConnectionMessage(); got != "Could not load azure.yaml: read azure.yaml: permission denied" {
		t.Errorf("ConnectionMessage=%q want load-error message", got)
	}
}

// --- SetMode ---

func TestSetModeSwitchesToLocalAndPersists(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeAzure}
	source := &stubProjectSource{yaml: azureConfiguredYaml(false)}
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	resp, err := client.SetMode(context.Background(), connect.NewRequest(&v1.SetModeRequest{
		Mode: v1.LogMode_LOG_MODE_LOCAL,
	}))
	if err != nil {
		t.Fatalf("SetMode error: %v", err)
	}
	if got := resp.Msg.GetMode(); got != v1.LogMode_LOG_MODE_LOCAL {
		t.Errorf("response Mode=%v want LOCAL", got)
	}
	if got := store.GetMode(); got != service.LogModeLocal {
		t.Errorf("store mode=%q want local", got)
	}
}

func TestSetModeToAzureSucceedsWhenConfigured(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeLocal}
	source := &stubProjectSource{yaml: azureConfiguredYaml(true)}
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	resp, err := client.SetMode(context.Background(), connect.NewRequest(&v1.SetModeRequest{
		Mode: v1.LogMode_LOG_MODE_AZURE,
	}))
	if err != nil {
		t.Fatalf("SetMode error: %v", err)
	}
	if got := resp.Msg.GetMode(); got != v1.LogMode_LOG_MODE_AZURE {
		t.Errorf("response Mode=%v want AZURE", got)
	}
	if !resp.Msg.GetAzureEnabled() {
		t.Error("AzureEnabled=false want true after Set")
	}
	if !resp.Msg.GetAzureRealtime() {
		t.Error("AzureRealtime=false want true after Set")
	}
	if got := store.GetMode(); got != service.LogModeAzure {
		t.Errorf("store mode=%q want azure", got)
	}
}

func TestSetModeToAzureRejectedAndDoesNotMutateWhenNotConfigured(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeLocal}
	source := &stubProjectSource{yaml: &service.AzureYaml{}} // no logs.analytics
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	_, err := client.SetMode(context.Background(), connect.NewRequest(&v1.SetModeRequest{
		Mode: v1.LogMode_LOG_MODE_AZURE,
	}))
	if err == nil {
		t.Fatal("expected error when switching to AZURE without configuration")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("error code=%v want FailedPrecondition; full err=%v", got, err)
	}
	// State must be untouched on rejection.
	if got := store.GetMode(); got != service.LogModeLocal {
		t.Errorf("store mode=%q want local (unchanged)", got)
	}
}

func TestSetModeRejectsUnspecified(t *testing.T) {
	store := &memoryModeStore{current: service.LogModeLocal}
	source := &stubProjectSource{yaml: &service.AzureYaml{}}
	client, cleanup := newModeTestServer(t, store, source, "/abs/anywhere")
	defer cleanup()

	_, err := client.SetMode(context.Background(), connect.NewRequest(&v1.SetModeRequest{
		Mode: v1.LogMode_LOG_MODE_UNSPECIFIED,
	}))
	if err == nil {
		t.Fatal("expected error for UNSPECIFIED mode")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("error code=%v want InvalidArgument; full err=%v", got, err)
	}
}

// --- Construction guards ---

func TestNewModeHandlerPanicsOnNilStore(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when store is nil")
		}
	}()
	_ = NewModeHandler(nil, &stubProjectSource{}, "/x")
}

func TestNewModeHandlerPanicsOnNilProjectSource(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when source is nil")
		}
	}()
	_ = NewModeHandler(&memoryModeStore{}, nil, "/x")
}

func TestNewModeHandlerPanicsOnEmptyProjectDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when projectDir is empty")
		}
	}()
	_ = NewModeHandler(&memoryModeStore{}, &stubProjectSource{}, "")
}

// --- Mounting ---

func TestModeServiceNotMountedWithoutStore(t *testing.T) {
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:  mgr,
		Project:    &stubProjectSource{yaml: &service.AzureYaml{}},
		ProjectDir: "/abs/anywhere",
		// Mode intentionally absent.
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + azdappv1connect.ModeServiceGetModeProcedure)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d want 404 when ModeService is not mounted", resp.StatusCode)
	}
}

// --- ModeStoreFuncs adapter ---

func TestModeStoreFuncsRoundTripsThroughInjectedFunctions(t *testing.T) {
	stored := service.LogModeLocal
	adapter := ModeStoreFuncs{
		Get: func() service.LogMode { return stored },
		Set: func(m service.LogMode) { stored = m },
	}

	if got := adapter.GetMode(); got != service.LogModeLocal {
		t.Errorf("initial GetMode=%q want local", got)
	}
	adapter.SetMode(service.LogModeAzure)
	if stored != service.LogModeAzure {
		t.Errorf("backing var=%q want azure after SetMode", stored)
	}
	if got := adapter.GetMode(); got != service.LogModeAzure {
		t.Errorf("post-set GetMode=%q want azure", got)
	}
}
