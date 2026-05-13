package rpc

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// NewObservabilityInterceptor returns an interceptor that emits structured
// logs around every unary call and stream lifecycle event. It exists today
// purely as the always-mounted observability slot; future cross-cutting
// concerns (auth, tracing, rate limiting) plug in here without touching
// every handler.
//
// Logging policy:
//   - One slog.Debug at start with procedure + peer protocol.
//   - One slog.Info on success with elapsed time.
//   - One slog.Warn on error with elapsed time and the connect.Code (NOT
//     the underlying error message, to avoid leaking internal details into
//     ops dashboards). Operators correlate by procedure + code.
//
// Performance: the interceptor allocates one time.Time and one log.Record
// per call. On the dashboard's traffic profile (interactive UI, low QPS)
// this is invisible; if a future high-QPS service needs zero-allocation
// observability it can opt out via per-handler options.
func NewObservabilityInterceptor() connect.Interceptor {
	return &observabilityInterceptor{}
}

type observabilityInterceptor struct{}

// WrapUnary wraps every unary handler call with start/end logging.
func (i *observabilityInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		spec := req.Spec()
		slog.Debug("rpc unary start", "procedure", spec.Procedure, "stream_type", spec.StreamType.String())

		resp, err := next(ctx, req)
		elapsed := time.Since(start)
		if err != nil {
			slog.Warn("rpc unary error",
				"procedure", spec.Procedure,
				"code", connect.CodeOf(err).String(),
				"elapsed_ms", elapsed.Milliseconds(),
			)
			return resp, err
		}
		slog.Info("rpc unary ok", "procedure", spec.Procedure, "elapsed_ms", elapsed.Milliseconds())
		return resp, nil
	}
}

// WrapStreamingClient is required by the connect.Interceptor interface but
// is a no-op for server-side observability. The dashboard server only acts
// as a Connect server; client-side instrumentation, when added, will live
// next to the Connect transport singleton in cli/dashboard.
func (i *observabilityInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler logs streaming RPC lifecycle (start, end, error).
// The interceptor cannot log per-message events without buffering, which
// would change back-pressure behavior, so we deliberately log only the
// outer lifecycle. Per-message instrumentation belongs in the handler.
func (i *observabilityInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		spec := conn.Spec()
		slog.Debug("rpc stream start", "procedure", spec.Procedure, "stream_type", spec.StreamType.String())

		err := next(ctx, conn)
		elapsed := time.Since(start)
		if err != nil {
			slog.Warn("rpc stream error",
				"procedure", spec.Procedure,
				"code", connect.CodeOf(err).String(),
				"elapsed_ms", elapsed.Milliseconds(),
			)
			return err
		}
		slog.Info("rpc stream ok", "procedure", spec.Procedure, "elapsed_ms", elapsed.Milliseconds())
		return nil
	}
}
