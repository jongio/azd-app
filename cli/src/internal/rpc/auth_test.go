package rpc

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
)

// authRequest implements connect.AnyRequest for the auth interceptor, which
// needs both Header() and Spec(). The embedded interface keeps the unexported
// internalOnly() method satisfied; any accessor we do not override panics with
// a nil-pointer dereference, which surfaces an unexpected call immediately.
type authRequest struct {
	connect.AnyRequest
	header http.Header
	spec   connect.Spec
}

func (r *authRequest) Header() http.Header { return r.header }
func (r *authRequest) Spec() connect.Spec  { return r.spec }

// newAuthRequest builds a unary request carrying the supplied session token.
// A token of "" produces a request with no X-Session-Token header at all,
// which is the shape an unauthenticated caller sends.
func newAuthRequest(token string) connect.AnyRequest {
	h := http.Header{}
	if token != "" {
		h.Set(SessionTokenHeader, token)
	}
	return &authRequest{
		header: h,
		spec: connect.Spec{
			Procedure:  "/azdapp.v1.LifecycleService/Ping",
			StreamType: connect.StreamTypeUnary,
		},
	}
}

// authStreamConn implements connect.StreamingHandlerConn for the streaming
// auth path, which reads RequestHeader() rather than Header().
type authStreamConn struct {
	connect.StreamingHandlerConn
	header http.Header
	spec   connect.Spec
}

func (c *authStreamConn) RequestHeader() http.Header { return c.header }
func (c *authStreamConn) Spec() connect.Spec         { return c.spec }

func newAuthStreamConn(token string) connect.StreamingHandlerConn {
	h := http.Header{}
	if token != "" {
		h.Set(SessionTokenHeader, token)
	}
	return &authStreamConn{
		header: h,
		spec: connect.Spec{
			Procedure:  "/azdapp.v1.LogsService/StreamLogs",
			StreamType: connect.StreamTypeServer,
		},
	}
}

func TestGenerateSessionTokenFormat(t *testing.T) {
	token := GenerateSessionToken()

	// 32 random bytes hex-encoded is always 64 characters.
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
	raw, err := hex.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("decoded token = %d bytes, want 32", len(raw))
	}
}

func TestGenerateSessionTokenIsUnique(t *testing.T) {
	// A collision across this many draws from a 256-bit space would mean the
	// generator is not actually random, which is the failure worth catching.
	const draws = 100
	seen := make(map[string]struct{}, draws)
	for i := 0; i < draws; i++ {
		token := GenerateSessionToken()
		if _, dup := seen[token]; dup {
			t.Fatalf("duplicate token generated on draw %d: %s", i, token)
		}
		seen[token] = struct{}{}
	}
}

func TestAuthInterceptorUnary(t *testing.T) {
	const expected = "b8f1c2d3e4a5968778695a4b3c2d1e0fb8f1c2d3e4a5968778695a4b3c2d1e0f"

	tests := []struct {
		name      string
		sent      string
		wantAllow bool
	}{
		{name: "exact token allowed", sent: expected, wantAllow: true},
		{name: "missing token rejected", sent: "", wantAllow: false},
		{name: "wrong token rejected", sent: "deadbeef", wantAllow: false},
		{name: "prefix of token rejected", sent: expected[:32], wantAllow: false},
		{name: "token with trailing space rejected", sent: expected + " ", wantAllow: false},
		{name: "case-flipped token rejected", sent: "B8F1C2D3E4A5968778695A4B3C2D1E0FB8F1C2D3E4A5968778695A4B3C2D1E0F", wantAllow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(expected)
			called := false
			want := connect.NewResponse(&v1.PingResponse{Status: "ok"})

			wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				called = true
				return want, nil
			})

			got, err := wrapped(context.Background(), newAuthRequest(tt.sent))

			if tt.wantAllow {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if !called {
					t.Error("inner handler was not invoked for a valid token")
				}
				if got != want {
					t.Errorf("response = %#v, want %#v", got, want)
				}
				return
			}

			if err == nil {
				t.Fatal("err = nil, want unauthenticated error")
			}
			if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
				t.Errorf("code = %v, want %v", code, connect.CodeUnauthenticated)
			}
			if called {
				t.Error("inner handler ran despite failed authentication")
			}
			if got != nil {
				t.Errorf("response = %#v, want nil on rejection", got)
			}
		})
	}
}

// TestAuthInterceptorRejectionLeaksNoDetail pins the deliberate choice to send
// a nil underlying error, so the rejection reveals nothing about the expected
// token to an unauthenticated caller.
func TestAuthInterceptorRejectionLeaksNoDetail(t *testing.T) {
	interceptor := NewAuthInterceptor("expected-token")

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("inner should not run")
	})

	_, err := wrapped(context.Background(), newAuthRequest("wrong"))
	if err == nil {
		t.Fatal("err = nil, want unauthenticated error")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("err type = %T, want *connect.Error", err)
	}
	if msg := connectErr.Message(); msg != "" {
		t.Errorf("message = %q, want empty so no token detail is disclosed", msg)
	}
}

func TestAuthInterceptorStreamingHandler(t *testing.T) {
	const expected = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name      string
		sent      string
		wantAllow bool
	}{
		{name: "exact token allowed", sent: expected, wantAllow: true},
		{name: "missing token rejected", sent: "", wantAllow: false},
		{name: "wrong token rejected", sent: "not-the-token", wantAllow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(expected)
			called := false

			wrapped := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
				called = true
				return nil
			})

			err := wrapped(context.Background(), newAuthStreamConn(tt.sent))

			if tt.wantAllow {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if !called {
					t.Error("inner stream handler was not invoked for a valid token")
				}
				return
			}

			if err == nil {
				t.Fatal("err = nil, want unauthenticated error")
			}
			if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
				t.Errorf("code = %v, want %v", code, connect.CodeUnauthenticated)
			}
			if called {
				t.Error("inner stream handler ran despite failed authentication")
			}
		})
	}
}

// TestAuthInterceptorStreamingHandlerPropagatesInnerError confirms the
// interceptor forwards the handler's own error untouched once auth succeeds,
// rather than masking it as an auth failure.
func TestAuthInterceptorStreamingHandlerPropagatesInnerError(t *testing.T) {
	const expected = "token"
	interceptor := NewAuthInterceptor(expected)
	wantErr := errors.New("stream blew up")

	wrapped := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return wantErr
	})

	err := wrapped(context.Background(), newAuthStreamConn(expected))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

// TestAuthInterceptorWrapStreamingClientIsPassthrough documents that outbound
// client streams are not authenticated by this interceptor; it is a
// server-side gate only.
func TestAuthInterceptorWrapStreamingClientIsPassthrough(t *testing.T) {
	interceptor := NewAuthInterceptor("token")

	called := false
	stub := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		called = true
		return nil
	}

	wrapped := interceptor.WrapStreamingClient(stub)
	wrapped(context.Background(), connect.Spec{})

	if !called {
		t.Error("WrapStreamingClient did not delegate to the inner func")
	}
}

// TestAuthInterceptorEmptyExpectedTokenAcceptsEmptyHeader pins a sharp edge:
// subtle.ConstantTimeCompare reports a match for two zero-length slices, so an
// interceptor built with an empty expected token authenticates every caller
// that sends no token at all.
//
// This is why Mount refuses to start when SessionToken is empty unless the
// caller explicitly sets AllowUnauthenticated (see server.go, CWE-1188). If
// that fail-closed guard were relaxed to simply pass the empty token through,
// authentication would silently succeed for every caller instead of failing.
// TestMountFailsWithEmptySessionToken covers the guard itself.
func TestAuthInterceptorEmptyExpectedTokenAcceptsEmptyHeader(t *testing.T) {
	interceptor := NewAuthInterceptor("")
	called := false

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return connect.NewResponse(&v1.PingResponse{Status: "ok"}), nil
	})

	if _, err := wrapped(context.Background(), newAuthRequest("")); err != nil {
		t.Fatalf("err = %v, want nil (documents the empty-token match)", err)
	}
	if !called {
		t.Error("inner handler was not invoked; empty-vs-empty is expected to match")
	}

	// A non-empty token against an empty expectation must still be rejected.
	called = false
	if _, err := wrapped(context.Background(), newAuthRequest("anything")); err == nil {
		t.Error("err = nil for non-empty token against empty expectation, want rejection")
	}
	if called {
		t.Error("inner handler ran for a non-empty token against an empty expectation")
	}
}

// newAuthenticatedTestServer mounts a real Connect server that requires the
// returned session token, so tests exercise the interceptor through the full
// HTTP stack rather than calling WrapUnary directly.
func newAuthenticatedTestServer(t *testing.T) (token string, url string, cleanup func()) {
	t.Helper()
	mgr := broadcast.New()
	token = GenerateSessionToken()

	mux := http.NewServeMux()
	if err := Mount(mux, Dependencies{Broadcast: mgr, Version: "auth-test", SessionToken: token}); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	srv := httptest.NewServer(mux)
	return token, srv.URL, func() {
		srv.Close()
		mgr.StopAll()
	}
}

// TestMountedServerRejectsUnauthenticatedCall is the end-to-end proof that a
// server built with a SessionToken denies callers that do not present it.
func TestMountedServerRejectsUnauthenticatedCall(t *testing.T) {
	_, url, cleanup := newAuthenticatedTestServer(t)
	defer cleanup()

	client := azdappv1connect.NewLifecycleServiceClient(http.DefaultClient, url)

	_, err := client.Ping(context.Background(), connect.NewRequest(&v1.PingRequest{}))
	if err == nil {
		t.Fatal("Ping succeeded without a session token; want unauthenticated error")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want %v", code, connect.CodeUnauthenticated)
	}
}

// TestMountedServerAcceptsAuthenticatedCall confirms the same server succeeds
// once the token is attached, so the interceptor is not rejecting everything.
func TestMountedServerAcceptsAuthenticatedCall(t *testing.T) {
	token, url, cleanup := newAuthenticatedTestServer(t)
	defer cleanup()

	client := azdappv1connect.NewLifecycleServiceClient(
		http.DefaultClient,
		url,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					req.Header().Set(SessionTokenHeader, token)
					return next(ctx, req)
				}
			},
		)),
	)

	resp, err := client.Ping(context.Background(), connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping with valid token returned error: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != "ok" {
		t.Errorf("Status = %q, want %q", got, "ok")
	}
}

// TestMountedServerRejectsWrongToken guards against an interceptor that only
// checks for the header's presence rather than its value.
func TestMountedServerRejectsWrongToken(t *testing.T) {
	_, url, cleanup := newAuthenticatedTestServer(t)
	defer cleanup()

	client := azdappv1connect.NewLifecycleServiceClient(
		http.DefaultClient,
		url,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					req.Header().Set(SessionTokenHeader, GenerateSessionToken())
					return next(ctx, req)
				}
			},
		)),
	)

	_, err := client.Ping(context.Background(), connect.NewRequest(&v1.PingRequest{}))
	if err == nil {
		t.Fatal("Ping succeeded with a different token; want unauthenticated error")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want %v", code, connect.CodeUnauthenticated)
	}
}
