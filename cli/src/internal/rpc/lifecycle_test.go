package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
)

// newTestServer wires a LifecycleHandler with a fresh broadcast.Manager
// behind an httptest server. Returns the manager so tests can drive
// events, the connect client, and a cleanup func.
func newTestServer(t *testing.T, version string) (*broadcast.Manager, azdappv1connect.LifecycleServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	if err := Mount(mux, Dependencies{Broadcast: mgr, Version: version, AllowUnauthenticated: true}); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewLifecycleServiceClient(srv.Client(), srv.URL)
	return mgr, client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

func TestPingReturnsOkAndVersion(t *testing.T) {
	_, client, cleanup := newTestServer(t, "test-version-1.2.3")
	defer cleanup()

	resp, err := client.Ping(context.Background(), connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != "ok" {
		t.Errorf("Status=%q want %q", got, "ok")
	}
	if got := resp.Msg.GetVersion(); got != "test-version-1.2.3" {
		t.Errorf("Version=%q want %q", got, "test-version-1.2.3")
	}
	if resp.Msg.GetServerTime() == nil {
		t.Error("ServerTime is nil; want a populated timestamp")
	}
}

func TestPingRejectsGetRequest(t *testing.T) {
	_, _, cleanup := newTestServer(t, "")
	defer cleanup()

	// Spin up a separate server to GET against the unary procedure path.
	mgr := broadcast.New()
	mux := http.NewServeMux()
	if err := Mount(mux, Dependencies{Broadcast: mgr, Version: "", AllowUnauthenticated: true}); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer mgr.StopAll()

	// Connect unary procedures only accept POST. A GET must not return 200.
	resp, err := http.Get(srv.URL + azdappv1connect.LifecycleServicePingProcedure)
	if err != nil {
		t.Fatalf("http.Get error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("GET /Ping returned 200 OK; expected method-not-allowed")
	}
}

func TestGetEnvironmentReadsProcessEnv(t *testing.T) {
	t.Setenv("CODESPACE_NAME", "")
	t.Setenv("CODESPACES", "")
	t.Setenv("AZURE_ENV_NAME", "ci-env")

	_, client, cleanup := newTestServer(t, "")
	defer cleanup()

	resp, err := client.GetEnvironment(context.Background(), connect.NewRequest(&v1.GetEnvironmentRequest{}))
	if err != nil {
		t.Fatalf("GetEnvironment returned error: %v", err)
	}
	if resp.Msg.GetEnvironmentName() != "ci-env" {
		t.Errorf("EnvironmentName=%q want ci-env", resp.Msg.GetEnvironmentName())
	}
	if resp.Msg.GetCodespace() == nil {
		t.Fatal("Codespace is nil; want populated message")
	}
	if resp.Msg.GetCodespace().GetEnabled() {
		t.Error("Codespace.Enabled=true outside Codespace")
	}
}

func TestStreamBroadcastDeliversMatchingEvents(t *testing.T) {
	mgr, client, cleanup := newTestServer(t, "")
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect server-streaming over HTTP/1 doesn't flush response headers
	// until the handler's first Send. client.StreamBroadcast blocks inside
	// http.Client.Do waiting for those headers, so we can't drive emits
	// synchronously after the call. Pump events from a goroutine that
	// waits for the server-side subscription, then fires.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && mgr.Count() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		mgr.Emit(broadcast.Event{
			Type:    "services-changed",
			Payload: map[string]any{"services": []any{}},
		})
		mgr.Emit(broadcast.Event{
			Type:    "ignored", // filtered out by EventTypes
			Payload: map[string]any{"x": 1},
		})
		mgr.Emit(broadcast.Event{
			Type:    "services-changed",
			Payload: map[string]any{"services": []any{map[string]any{"name": "api"}}},
		})
	}()

	stream, err := client.StreamBroadcast(ctx, connect.NewRequest(&v1.StreamBroadcastRequest{
		EventTypes: []string{"services-changed"},
	}))
	if err != nil {
		t.Fatalf("StreamBroadcast: %v", err)
	}

	// Receive first event.
	if !stream.Receive() {
		t.Fatalf("stream.Receive returned false; err=%v", stream.Err())
	}
	if got := stream.Msg().GetEvent().GetType(); got != "services-changed" {
		t.Errorf("event[0].Type=%q want services-changed", got)
	}

	// Receive second matching event (filter dropped the middle one).
	if !stream.Receive() {
		t.Fatalf("stream.Receive returned false on second event; err=%v", stream.Err())
	}
	got := stream.Msg().GetEvent()
	if got.GetType() != "services-changed" {
		t.Errorf("event[1].Type=%q want services-changed", got.GetType())
	}
	services, ok := got.GetPayload().AsMap()["services"].([]any)
	if !ok || len(services) != 1 {
		t.Errorf("payload.services=%v want 1-element slice", got.GetPayload().AsMap()["services"])
	}
}

func TestStreamBroadcastClosesOnServerShutdown(t *testing.T) {
	mgr, client, cleanup := newTestServer(t, "")
	// Defer cleanup but call StopAll explicitly mid-test to simulate shutdown.
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Same blocking-headers issue as above: trigger StopAll from a
	// goroutine once the handler subscribes. The server returns
	// CodeUnavailable, which Connect surfaces to the client as either
	// an error from StreamBroadcast itself or via stream.Err() after
	// Receive returns false.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && mgr.Count() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		mgr.StopAll()
	}()

	stream, err := client.StreamBroadcast(ctx, connect.NewRequest(&v1.StreamBroadcastRequest{}))
	// StopAll fires before any Send, so connect-go may surface the
	// error from the call itself OR from Receive(). Accept either path.
	if err != nil {
		if got := connect.CodeOf(err); got != connect.CodeUnavailable {
			t.Errorf("StreamBroadcast err code=%v want Unavailable; full err=%v", got, err)
		}
		return
	}

	if stream.Receive() {
		t.Fatalf("stream.Receive returned true after StopAll; want false")
	}
	rerr := stream.Err()
	if rerr == nil {
		t.Fatal("stream.Err() returned nil after StopAll; want non-nil")
	}
	if got := connect.CodeOf(rerr); got != connect.CodeUnavailable {
		t.Errorf("error code=%v want Unavailable; full err=%v", got, rerr)
	}
}

func TestStreamBroadcastClientCancelExitsCleanly(t *testing.T) {
	mgr, client, cleanup := newTestServer(t, "")
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel from a goroutine once the subscription is active. Cancel
	// before the first Send aborts the in-flight HTTP request, which
	// connect-go surfaces as either an error from StreamBroadcast or
	// from stream.Receive; both are valid client-cancel paths.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && mgr.Count() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	stream, err := client.StreamBroadcast(ctx, connect.NewRequest(&v1.StreamBroadcastRequest{}))
	if err == nil {
		// If the call returned a stream, Receive must report cancel.
		if stream.Receive() {
			t.Fatalf("Receive returned true after cancel; want false")
		}
		if rerr := stream.Err(); rerr != nil &&
			!strings.Contains(strings.ToLower(rerr.Error()), "cancel") &&
			connect.CodeOf(rerr) != connect.CodeCanceled {
			// Some Connect builds surface CodeCanceled, others surface ctx.Err
			// directly. Either is acceptable; we just want the cancellation
			// path to wake the handler up.
			t.Logf("stream err on cancel: %v", rerr)
		}
	}

	// Subscriber must be unregistered server-side eventually.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && mgr.Count() != 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if mgr.Count() != 0 {
		t.Errorf("subscriber count=%d after client cancel; want 0", mgr.Count())
	}
}

func TestNewLifecycleHandlerPanicsOnNilBroadcast(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Broadcast is nil")
		}
	}()
	_ = NewLifecycleHandler(Dependencies{Broadcast: nil})
}

func TestPayloadToStructHandlesNestedStructs(t *testing.T) {
	// Demonstrates that the JSON-roundtrip helper accepts payloads
	// containing arbitrary structs (the production case for service
	// registry slices, which structpb.NewStruct alone rejects).
	type entry struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	payload := map[string]any{
		"services": []any{
			entry{Name: "api", Port: 8080},
			entry{Name: "web", Port: 3000},
		},
	}

	got, err := payloadToStruct(payload)
	if err != nil {
		t.Fatalf("payloadToStruct returned error: %v", err)
	}
	asMap := got.AsMap()
	services, ok := asMap["services"].([]any)
	if !ok || len(services) != 2 {
		t.Fatalf("services=%v want 2-element slice", asMap["services"])
	}
	first, ok := services[0].(map[string]any)
	if !ok {
		t.Fatalf("first service=%v want map", services[0])
	}
	if first["name"] != "api" {
		t.Errorf("first service name=%v want api", first["name"])
	}
}

func TestPayloadToStructEmptyPayload(t *testing.T) {
	got, err := payloadToStruct(nil)
	if err != nil {
		t.Fatalf("nil payload returned error: %v", err)
	}
	if len(got.GetFields()) != 0 {
		t.Errorf("empty payload produced %d fields, want 0", len(got.GetFields()))
	}
}
