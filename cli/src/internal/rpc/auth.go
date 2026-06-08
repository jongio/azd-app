package rpc

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"

	"connectrpc.com/connect"
)

const (
	// SessionTokenHeader is the HTTP header the dashboard sends to authenticate RPC calls.
	SessionTokenHeader = "X-Session-Token"
)

// GenerateSessionToken creates a cryptographically random session token.
// Called once at server startup; the token is passed to the embedded dashboard
// and validated on every RPC call.
func GenerateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: this should never happen on any supported OS
		panic("rpc: failed to generate session token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// NewAuthInterceptor returns a Connect interceptor that validates the
// X-Session-Token header on every inbound request. Requests without a valid
// token receive connect.CodeUnauthenticated. This prevents local processes
// (or DNS-rebinding attacks) from calling the RPC server without the token
// that only the legitimate dashboard client possesses.
func NewAuthInterceptor(expectedToken string) connect.Interceptor {
	return &authInterceptor{token: expectedToken}
}

type authInterceptor struct {
	token string
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := a.validate(req.Header().Get(SessionTokenHeader), req.Spec().Procedure); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := a.validate(conn.RequestHeader().Get(SessionTokenHeader), conn.Spec().Procedure); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (a *authInterceptor) validate(token, procedure string) error {
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) == 1 {
		return nil
	}
	slog.Debug("rpc auth rejected", "procedure", procedure, "reason", "invalid or missing session token")
	return connect.NewError(connect.CodeUnauthenticated, nil)
}
