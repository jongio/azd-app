package rpc

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
)

// fakeUnaryRequest implements connect.AnyRequest just enough for the
// interceptor's WrapUnary code path. We only need Spec(); other accessors
// are wired to safe defaults so an accidental call surfaces clearly in a
// test failure rather than a nil-pointer panic deep in the interceptor.
type fakeUnaryRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (f *fakeUnaryRequest) Spec() connect.Spec { return f.spec }

func newPingRequest() connect.AnyRequest {
	return &fakeUnaryRequest{
		spec: connect.Spec{
			Procedure:  "/azdapp.v1.LifecycleService/Ping",
			StreamType: connect.StreamTypeUnary,
		},
	}
}

func TestObservabilityInterceptorPassesThroughSuccess(t *testing.T) {
	interceptor := NewObservabilityInterceptor()
	want := connect.NewResponse(&v1.PingResponse{Status: "ok"})
	called := false

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return want, nil
	})

	got, err := wrapped(context.Background(), newPingRequest())
	if err != nil {
		t.Fatalf("wrapped returned err=%v want nil", err)
	}
	if !called {
		t.Fatal("inner handler was not invoked")
	}
	if got != want {
		t.Errorf("response mismatch: got %#v want %#v", got, want)
	}
}

func TestObservabilityInterceptorPropagatesError(t *testing.T) {
	interceptor := NewObservabilityInterceptor()
	wantErr := connect.NewError(connect.CodeInternal, errors.New("boom"))

	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, wantErr
	})

	_, err := wrapped(context.Background(), newPingRequest())
	if err == nil {
		t.Fatal("wrapped returned nil err; want non-nil")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("error code=%v want Internal", got)
	}
}

func TestObservabilityInterceptorWrapStreamingClientIsNoop(t *testing.T) {
	interceptor := NewObservabilityInterceptor()

	called := false
	stub := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		called = true
		return nil
	}

	wrapped := interceptor.WrapStreamingClient(stub)
	wrapped(context.Background(), connect.Spec{})
	if !called {
		t.Fatal("WrapStreamingClient did not delegate to inner func")
	}
}
