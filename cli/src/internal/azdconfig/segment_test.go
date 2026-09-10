package azdconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncodeSegmentLeavesSafeNamesAlone is the backward-compatibility pin. Any
// service name that already worked must produce exactly the key it produced
// before encoding existed, otherwise upgrading orphans the stored port.
func TestEncodeSegmentLeavesSafeNamesAlone(t *testing.T) {
	for _, name := range []string{
		"api",
		"web",
		"web-api",
		"web_api",
		"Service1",
		"a",
		"0",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, name, encodeSegment(name),
				"safe names must encode to themselves so existing keys keep resolving")
		})
	}
}

// TestEncodeSegmentFlattensPathSeparators covers the bug this encoding exists
// for. A dot in a service name used to split the config path and produce a
// nested object that GetAllServicePorts could not read back.
func TestEncodeSegmentFlattensPathSeparators(t *testing.T) {
	tests := map[string]string{
		"foo.bar":     "foo%2Ebar",
		"my service":  "my%20service",
		"web/api":     "web%2Fapi",
		"100%":        "100%25",
		"a.b.c":       "a%2Eb%2Ec",
		"caf\u00e9":   "caf%C3%A9",
		"tab\there":   "tab%09here",
		"trailing.":   "trailing%2E",
		".leading":    "%2Eleading",
		"semi;colon":  "semi%3Bcolon",
		"quote\"mark": "quote%22mark",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got := encodeSegment(input)
			assert.Equal(t, want, got)
			assert.NotContains(t, got, ".",
				"an encoded segment must never contain a path separator")
		})
	}
}

// TestSegmentRoundTrip proves the encoding is lossless, which is what lets
// GetAllServicePorts report the original service name back to the caller.
func TestSegmentRoundTrip(t *testing.T) {
	for _, name := range []string{
		"api",
		"foo.bar",
		"my service",
		"web/api",
		"100%",
		"caf\u00e9",
		"%2E",
		"%",
		"%%",
		"%zz",
		"a%2",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, name, decodeSegment(encodeSegment(name)))
		})
	}
}

// TestDecodeSegmentToleratesMalformedEscapes guards against a hand-edited or
// corrupted config causing a panic or silent truncation.
func TestDecodeSegmentToleratesMalformedEscapes(t *testing.T) {
	tests := map[string]string{
		"plain": "plain",
		"%":     "%",
		"%2":    "%2",
		"%zz":   "%zz",
		"%2E":   ".",
		"%2e":   ".",
		"a%":    "a%",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, want, decodeSegment(input))
		})
	}
}

// TestServicePortPathIsFlat pins the whole point of the change: the config path
// for a service port must have exactly one segment after "ports", no matter
// what the service is called.
func TestServicePortPathIsFlat(t *testing.T) {
	const hash = "abc123"
	prefix := projectConfigPath(hash, "ports.")

	for _, name := range []string{"api", "foo.bar", "a.b.c.d", "my service"} {
		t.Run(name, func(t *testing.T) {
			path := servicePortPath(hash, name)
			require.True(t, len(path) > len(prefix) && path[:len(prefix)] == prefix)

			segment := path[len(prefix):]
			assert.NotContains(t, segment, ".",
				"a dotted service name must not split into extra config path segments")
			assert.Equal(t, name, decodeSegment(segment))
		})
	}
}

// TestDottedServiceNameRoundTripsThroughClient is the end-to-end regression.
// Before encoding, storing a port for "foo.bar" wrote a nested object and
// GetAllServicePorts dropped it entirely.
func TestDottedServiceNameRoundTripsThroughClient(t *testing.T) {
	client := NewInMemoryClient()
	const hash = "projecthash"

	require.NoError(t, client.SetServicePort(hash, "foo.bar", 3000))
	require.NoError(t, client.SetServicePort(hash, "api", 3001))

	port, err := client.GetServicePort(hash, "foo.bar")
	require.NoError(t, err)
	assert.Equal(t, 3000, port)

	all, err := client.GetAllServicePorts(hash)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"foo.bar": 3000, "api": 3001}, all,
		"a dotted service name must survive the write and read back under its original name")

	require.NoError(t, client.ClearServicePort(hash, "foo.bar"))
	all, err = client.GetAllServicePorts(hash)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"api": 3001}, all)
}

// TestDistinctNamesDoNotCollide checks the encoding is injective for inputs
// that differ only in characters the encoder touches. A collision would let one
// service overwrite another's port.
func TestDistinctNamesDoNotCollide(t *testing.T) {
	names := []string{"foo.bar", "foo-bar", "foo_bar", "foo%2Ebar", "foo bar"}

	seen := make(map[string]string, len(names))
	for _, name := range names {
		encoded := encodeSegment(name)
		if prev, dup := seen[encoded]; dup {
			t.Fatalf("names %q and %q both encode to %q", prev, name, encoded)
		}
		seen[encoded] = name
	}
}
