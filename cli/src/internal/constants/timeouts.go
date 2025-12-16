// Package constants provides shared constants for the azd-app CLI.
package constants

import "time"

// HTTP and network timeouts

// HTTPIdleConnTimeout is the maximum amount of time an idle connection will remain open
// in the HTTP connection pool before being closed.
const HTTPIdleConnTimeout = 90 * time.Second

// HTTPDialTimeout is the maximum amount of time to wait when establishing a TCP connection.
const HTTPDialTimeout = 5 * time.Second

// HTTPKeepAliveTimeout is the keep-alive period for an active network connection.
const HTTPKeepAliveTimeout = 30 * time.Second

// HTTPTLSHandshakeTimeout is the maximum amount of time to wait for a TLS handshake.
const HTTPTLSHandshakeTimeout = 5 * time.Second

// HTTPExpectContinueTimeout is the maximum amount of time to wait for a server's first
// response headers after fully writing the request headers if the request has an
// "Expect: 100-continue" header.
const HTTPExpectContinueTimeout = 1 * time.Second

// Test timeouts

// TestShortSleepDuration is a short sleep duration used in integration tests
// to allow services to stabilize between operations.
const TestShortSleepDuration = 2 * time.Second

// TestServiceTimeout is the default timeout for waiting on service operations in tests.
const TestServiceTimeout = 5 * time.Second
