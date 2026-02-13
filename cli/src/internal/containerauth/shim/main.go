package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Check for "auth token" subcommand; pass through other commands to real azd if available
	isAuthToken := len(os.Args) >= 3 && os.Args[1] == "auth" && os.Args[2] == "token"
	if !isAuthToken {
		// Try to find a real azd binary to handle this command
		for _, realAzdPath := range []string{"/usr/share/azd/azd", "/opt/azd/azd"} {
			if _, err := os.Stat(realAzdPath); err == nil {
				if err := syscall.Exec(realAzdPath, os.Args, os.Environ()); err != nil {
					fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to exec %s: %v\n", realAzdPath, err)
					os.Exit(1)
				}
			}
		}
		cmd := ""
		if len(os.Args) > 1 {
			cmd = os.Args[1]
		}
		fmt.Fprintf(os.Stderr, "azd-auth-shim: command '%s' is not supported by the auth shim. Install the full azd CLI for non-auth commands.\n", cmd)
		os.Exit(1)
	}

	// Parse --scope flag
	var scope string
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--scope" && i+1 < len(os.Args) {
			scope = os.Args[i+1]
			i++ // skip the next arg
		} else if os.Args[i] == "--output" && i+1 < len(os.Args) {
			// we expect json, but don't need to validate it
			i++ // skip the next arg
		}
	}

	if scope == "" {
		fmt.Fprintln(os.Stderr, "azd-auth-shim: --scope is required")
		os.Exit(1)
	}

	// Validate scope format (defense in depth — server also validates)
	if !strings.HasPrefix(scope, "https://") || len(scope) > 512 {
		fmt.Fprintln(os.Stderr, "azd-auth-shim: invalid scope format, expected https:// URL")
		os.Exit(1)
	}

	// Read connection info from env vars
	host := os.Getenv("AZD_AUTH_HOST")
	if host == "" {
		host = "host.docker.internal"
	}

	port := os.Getenv("AZD_AUTH_PORT")
	if port == "" {
		fmt.Fprintln(os.Stderr, "azd-auth-shim: AZD_AUTH_PORT environment variable is required")
		os.Exit(1)
	}

	// Read cert directory
	certsDir := os.Getenv("AZD_AUTH_CERTS_DIR")
	if certsDir == "" {
		certsDir = "/run/secrets/azd-auth"
	}

	// Load client cert and key
	clientCertPath := filepath.Join(certsDir, "client.pem")
	clientKeyPath := filepath.Join(certsDir, "client-key.pem")
	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to load client certificate: %v\n", err)
		os.Exit(1)
	}

	// Load CA cert
	caCertPath := filepath.Join(certsDir, "ca.pem")
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to read CA certificate: %v\n", err)
		os.Exit(1)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		fmt.Fprintln(os.Stderr, "azd-auth-shim: failed to parse CA certificate")
		os.Exit(1)
	}

	// Build mTLS HTTP client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Prepare request body
	requestBody := map[string][]string{
		"scopes": {scope},
	}
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to marshal request: %v\n", err)
		os.Exit(1)
	}

	// Make POST request
	url := fmt.Sprintf("https://%s:%s/token", host, port)
	resp, err := client.Post(url, "application/json", bytes.NewReader(requestJSON))
	if err != nil {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to request token: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response (limit to 1MB to prevent memory exhaustion from a compromised server)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "azd-auth-shim: token server returned status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Output response to stdout in azd format
	// Response from server should already be in format: {"token":"...","expiresOn":"..."}
	fmt.Println(string(body))
}
