// Package testports provides shared port constants for testing.
// This eliminates magic numbers in test files and ensures consistency.
package testports

// Common test ports used across integration tests.
const (
	// HTTPTestPort is the default port for HTTP test servers.
	HTTPTestPort = 8080

	// AlternativeHTTPPort is an alternative port for multi-service tests.
	AlternativeHTTPPort = 3000

	// PrometheusMetricsPort is the standard port for Prometheus metrics endpoints.
	PrometheusMetricsPort = 9090

	// PostgresPort is the standard PostgreSQL port.
	PostgresPort = 5432

	// RedisPort is the standard Redis port.
	RedisPort = 6379

	// MongoDBPort is the standard MongoDB port.
	MongoDBPort = 27017
)
