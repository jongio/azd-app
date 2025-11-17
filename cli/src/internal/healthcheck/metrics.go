package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
)

var (
	healthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "azd_health_check_duration_seconds",
			Help:    "Duration of health checks in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "status", "check_type"},
	)

	healthCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azd_health_check_total",
			Help: "Total number of health checks performed",
		},
		[]string{"service", "status", "check_type"},
	)

	healthCheckErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azd_health_check_errors_total",
			Help: "Total number of health check errors",
		},
		[]string{"service", "error_type"},
	)

	serviceUptime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azd_service_uptime_seconds",
			Help: "Service uptime in seconds since last health check detected it running",
		},
		[]string{"service"},
	)

	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azd_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"service"},
	)

	healthCheckResponseCode = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azd_health_check_http_status_total",
			Help: "HTTP status codes from health checks",
		},
		[]string{"service", "status_code"},
	)
)

// recordHealthCheck records metrics for a health check result.
func recordHealthCheck(result HealthCheckResult) {
	labels := prometheus.Labels{
		"service":    result.ServiceName,
		"status":     string(result.Status),
		"check_type": string(result.CheckType),
	}

	healthCheckDuration.With(labels).Observe(result.ResponseTime.Seconds())
	healthCheckTotal.With(labels).Inc()

	if result.Error != "" {
		errorType := getErrorType(result.Error)
		healthCheckErrors.With(prometheus.Labels{
			"service":    result.ServiceName,
			"error_type": errorType,
		}).Inc()
	}

	if result.StatusCode > 0 {
		healthCheckResponseCode.With(prometheus.Labels{
			"service":     result.ServiceName,
			"status_code": http.StatusText(result.StatusCode),
		}).Inc()
	}

	// Update uptime if service is healthy
	if result.Status == HealthStatusHealthy && result.Uptime > 0 {
		serviceUptime.With(prometheus.Labels{
			"service": result.ServiceName,
		}).Set(result.Uptime.Seconds())
	}
}

// recordCircuitBreakerState records the circuit breaker state.
func recordCircuitBreakerState(serviceName string, state gobreaker.State) {
	var stateValue float64
	switch state {
	case gobreaker.StateClosed:
		stateValue = 0
	case gobreaker.StateHalfOpen:
		stateValue = 1
	case gobreaker.StateOpen:
		stateValue = 2
	}

	circuitBreakerState.With(prometheus.Labels{
		"service": serviceName,
	}).Set(stateValue)
}

// getErrorType categorizes errors for metrics.
func getErrorType(errMsg string) string {
	// Convert to lowercase for case-insensitive matching
	errLower := strings.ToLower(errMsg)
	
	switch {
	case containsAny(errLower, "timeout", "deadline", "timed out"):
		return "timeout"
	case containsAny(errLower, "connection refused", "no connection", "unreachable"):
		return "connection_refused"
	case containsAny(errLower, "circuit breaker", "circuit open", "too many failures"):
		return "circuit_breaker"
	case containsAny(errLower, "context canceled", "canceled"):
		return "canceled"
	case containsAny(errLower, "500", "503", "502", "504"):
		return "server_error"
	case containsAny(errLower, "401", "403"):
		return "auth_error"
	case containsAny(errLower, "404"):
		return "not_found"
	case containsAny(errLower, "process", "pid"):
		return "process_error"
	case containsAny(errLower, "port"):
		return "port_error"
	default:
		return "unknown"
	}
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// MetricsServer holds the metrics HTTP server instance.
type MetricsServer struct {
	server *http.Server
	mu     sync.Mutex
}

var globalMetricsServer *MetricsServer

// ServeMetrics starts a Prometheus metrics HTTP server.
func ServeMetrics(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add health endpoint for the metrics server itself
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	globalMetricsServer = &MetricsServer{server: server}

	log.Info().Int("port", port).Str("endpoint", "/metrics").Msg("Starting Prometheus metrics server")

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StopMetricsServer gracefully shuts down the metrics server.
func StopMetricsServer(ctx context.Context) error {
	if globalMetricsServer == nil {
		return nil
	}

	globalMetricsServer.mu.Lock()
	defer globalMetricsServer.mu.Unlock()

	if globalMetricsServer.server != nil {
		log.Info().Msg("Shutting down metrics server")
		return globalMetricsServer.server.Shutdown(ctx)
	}
	return nil
}
