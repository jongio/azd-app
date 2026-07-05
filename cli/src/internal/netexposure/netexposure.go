// Package netexposure provides an advisory check that flags services configured
// to bind to every network interface during local development. Binding to
// 0.0.0.0 or the IPv6 equivalent makes a local service reachable by anyone on
// the same network, which is rarely what a developer wants on a laptop.
//
// Like the other run preflight checks, this one is advisory. It prints a
// warning and never blocks the run or changes how a service starts.
package netexposure

import (
	"sort"
	"strings"
)

// Finding describes one service configuration value that exposes the service on
// all interfaces.
type Finding struct {
	// Source identifies where the value was read from, for example
	// "azure.yaml (service: api)" or ".env".
	Source string
	// Key is the environment variable name.
	Key string
	// Value is the offending bind value, trimmed.
	Value string
}

// bindKeys are environment variable names whose value is a bind host or a
// host:port pair.
var bindKeys = map[string]bool{
	"HOST": true, "HOSTNAME": true, "BIND": true, "BIND_ADDR": true,
	"BIND_ADDRESS": true, "LISTEN": true, "LISTEN_ADDR": true,
	"LISTEN_ADDRESS": true, "SERVER_HOST": true, "HTTP_HOST": true,
	"APP_HOST": true, "ADDR": true, "ADDRESS": true,
}

// bindSuffixes match variable names that end in a bind-host segment, such as
// ADMIN_HOST or GRPC_LISTEN_ADDR. A wildcard host is only ever a bind target,
// never a connect target, so matching by suffix is safe.
var bindSuffixes = []string{"_HOST", "_HOSTNAME", "_ADDR", "_ADDRESS", "_BIND", "_LISTEN"}

// urlKeys are environment variable names whose value is one or more URLs. Some
// runtimes accept a semicolon-separated list, for example ASPNETCORE_URLS.
var urlKeys = map[string]bool{
	"ASPNETCORE_URLS": true, "URLS": true, "DOTNET_URLS": true,
}

// wildcardHosts are host values that mean "all interfaces".
var wildcardHosts = map[string]bool{
	"0.0.0.0": true, "::": true, "[::]": true, "*": true, "+": true,
}

// Inspect reports whether a single key and value configure an all-interfaces
// bind. It returns false for loopback addresses, specific IPs, and values that
// are not bind hints.
func Inspect(key, value string) bool {
	v := strings.TrimSpace(value)
	if v == "" {
		return false
	}
	upper := strings.ToUpper(strings.TrimSpace(key))

	if isBindKey(upper) {
		return isWildcardHost(hostOnly(v))
	}
	if urlKeys[upper] || strings.Contains(v, "://") {
		for _, part := range strings.Split(v, ";") {
			if isWildcardHost(hostFromURL(part)) {
				return true
			}
		}
	}
	return false
}

// ScanEnv inspects every entry in env and returns the findings, sorted by key.
func ScanEnv(source string, env map[string]string) []Finding {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var findings []Finding
	for _, k := range keys {
		if Inspect(k, env[k]) {
			findings = append(findings, Finding{Source: source, Key: k, Value: strings.TrimSpace(env[k])})
		}
	}
	return findings
}

func isBindKey(upper string) bool {
	if bindKeys[upper] {
		return true
	}
	for _, suffix := range bindSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}
	return false
}

func isWildcardHost(host string) bool {
	return wildcardHosts[strings.TrimSpace(host)]
}

// hostOnly returns the host portion of a bind value that may include a port,
// such as "0.0.0.0:8080" or "[::]:8080". A bare IPv6 address like "::" is
// returned unchanged.
func hostOnly(v string) string {
	if strings.HasPrefix(v, "[") {
		if end := strings.Index(v, "]"); end != -1 {
			return v[:end+1]
		}
	}
	if strings.Count(v, ":") == 1 {
		return v[:strings.IndexByte(v, ':')]
	}
	return v
}

// hostFromURL extracts the host from a URL such as "http://0.0.0.0:5000" or the
// ASP.NET forms "http://+:5000" and "http://*:5000". It returns an empty string
// when raw is not a URL.
func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	i := strings.Index(raw, "://")
	if i == -1 {
		return ""
	}
	rest := raw[i+3:]
	if slash := strings.IndexByte(rest, '/'); slash != -1 {
		rest = rest[:slash]
	}
	return hostOnly(rest)
}
